package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/dushixiang/pika/internal/protocol"
	"github.com/dushixiang/pika/pkg/agent/audit"
	"github.com/dushixiang/pika/pkg/agent/collector"
	"github.com/dushixiang/pika/pkg/agent/config"
	"github.com/dushixiang/pika/pkg/agent/id"
	"github.com/dushixiang/pika/pkg/version"
	"github.com/gorilla/websocket"
	"github.com/jpillora/backoff"
)

// safeConn 线程安全的 WebSocket 连接包装器
type safeConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

// WriteJSON 线程安全地写入 JSON 消息
func (sc *safeConn) WriteJSON(v interface{}) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.conn.WriteJSON(v)
}

// WriteMessage 线程安全地写入消息
func (sc *safeConn) WriteMessage(messageType int, data []byte) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.conn.WriteMessage(messageType, data)
}

// ReadJSON 读取 JSON 消息（读操作本身是安全的）
func (sc *safeConn) ReadJSON(v interface{}) error {
	return sc.conn.ReadJSON(v)
}

// Close 关闭连接
func (sc *safeConn) Close() error {
	return sc.conn.Close()
}

// Agent 探针服务
type Agent struct {
	cfg    *config.Config
	idMgr  *id.Manager
	cancel context.CancelFunc
}

// New 创建 Agent 实例
func New(cfg *config.Config) *Agent {
	return &Agent{
		cfg:   cfg,
		idMgr: id.NewManager(),
	}
}

// Start 启动探针服务
func (a *Agent) Start(ctx context.Context) error {
	// 创建可取消的 context
	ctx, cancel := context.WithCancel(ctx)
	a.cancel = cancel

	// 启动探针主循环
	b := &backoff.Backoff{
		Min:    5 * time.Second,
		Max:    5 * time.Minute,
		Factor: 2,
		Jitter: true,
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		if err := a.runOnce(ctx); err != nil {
			retryAfter := b.Duration()
			log.Printf("⚠️  探针运行出错: %v，将在 %v 后重试", err, retryAfter)

			select {
			case <-time.After(retryAfter):
				continue
			case <-ctx.Done():
				return nil
			}
		}

		// 正常断开，重置退避
		b.Reset()
		log.Println("连接已断开，准备重连...")
		time.Sleep(3 * time.Second)
	}
}

// Stop 停止探针服务
func (a *Agent) Stop() {
	if a.cancel != nil {
		a.cancel()
	}
}

// runOnce 运行一次探针连接
func (a *Agent) runOnce(ctx context.Context) error {
	wsURL := a.cfg.GetWebSocketURL()
	log.Printf("🔌 正在连接到服务器: %s", wsURL)

	// 连接到服务器
	rawConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}
	defer rawConn.Close()

	// 创建线程安全的连接包装器
	conn := &safeConn{conn: rawConn}

	// 设置 Ping 处理器，自动响应服务端的 Ping
	rawConn.SetPingHandler(func(appData string) error {
		// WriteControl 有内置锁，可以安全调用
		err := rawConn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(time.Second))
		if err == nil {
			log.Println("💓 收到 Ping，已发送 Pong")
		}
		return err
	})

	// 发送注册消息
	if err := a.registerAgent(conn); err != nil {
		return fmt.Errorf("注册失败: %w", err)
	}

	log.Println("✅ 探针注册成功，开始监控...")

	// 创建采集器管理器
	collectorManager := collector.NewManager(a.cfg)

	// 创建完成通道
	done := make(chan struct{})
	errChan := make(chan error, 3)

	// 启动读取循环（处理服务端的 Ping/Pong 等控制消息）
	go func() {
		if err := a.readLoop(rawConn, done); err != nil {
			errChan <- fmt.Errorf("读取失败: %w", err)
		}
	}()

	// 启动心跳和数据发送
	go func() {
		if err := a.heartbeatLoop(ctx, conn, done); err != nil {
			errChan <- fmt.Errorf("心跳失败: %w", err)
		}
	}()

	go func() {
		if err := a.metricsLoop(ctx, conn, collectorManager, done); err != nil {
			errChan <- fmt.Errorf("数据采集失败: %w", err)
		}
	}()

	// 等待错误或上下文取消
	select {
	case err := <-errChan:
		close(done)
		return err
	case <-ctx.Done():
		close(done)
		// 优雅关闭连接
		closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
		if err := conn.WriteMessage(websocket.CloseMessage, closeMsg); err != nil {
			log.Printf("⚠️  关闭连接失败: %v", err)
		}
		time.Sleep(time.Second)
		return nil
	}
}

// readLoop 读取服务端消息（主要用于处理 Ping/Pong 和指令）
func (a *Agent) readLoop(conn *websocket.Conn, done chan struct{}) error {
	for {
		select {
		case <-done:
			return nil
		default:
		}

		// 读取消息（这会触发 PingHandler）
		_, message, err := conn.ReadMessage()
		if err != nil {
			// 检查是否是正常关闭
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Println("服务端正常关闭连接")
				return nil
			}
			// 其他错误
			return fmt.Errorf("读取消息失败: %w", err)
		}

		// 解析消息
		var msg protocol.Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("⚠️  解析消息失败: %v", err)
			continue
		}

		// 处理指令消息
		if msg.Type == protocol.MessageTypeCommand {
			go a.handleCommand(conn, msg.Data)
		}
	}
}

// registerAgent 注册探针
func (a *Agent) registerAgent(conn *safeConn) error {
	// 加载或生成探针 ID
	agentID, err := a.idMgr.Load()
	if err != nil {
		return fmt.Errorf("加载 agent ID 失败: %w", err)
	}
	log.Printf("🆔 Agent ID: %s (存储在: %s)", agentID, a.idMgr.GetPath())

	// 获取主机信息
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}

	// 使用配置或默认值
	agentName := a.cfg.Agent.Name
	if agentName == "" {
		agentName = hostname
	}

	// 构建注册请求
	registerReq := protocol.RegisterRequest{
		AgentInfo: protocol.AgentInfo{
			ID:       agentID,
			Name:     agentName,
			Hostname: hostname,
			OS:       runtime.GOOS,
			Arch:     runtime.GOARCH,
			Version:  version.GetVersion(),
		},
		ApiKey: a.cfg.Server.APIKey,
	}

	reqData, err := json.Marshal(registerReq)
	if err != nil {
		return fmt.Errorf("序列化注册请求失败: %w", err)
	}

	msg := protocol.Message{
		Type: protocol.MessageTypeRegister,
		Data: reqData,
	}

	if err := conn.WriteJSON(msg); err != nil {
		return fmt.Errorf("发送注册消息失败: %w", err)
	}

	// 读取注册响应
	var response protocol.Message
	if err := conn.ReadJSON(&response); err != nil {
		return fmt.Errorf("读取注册响应失败: %w", err)
	}

	// 检查响应类型
	if response.Type == protocol.MessageTypeRegisterErr {
		var errResp protocol.RegisterResponse
		if err := json.Unmarshal(response.Data, &errResp); err == nil {
			return fmt.Errorf("注册失败: %s", errResp.Message)
		}
		return fmt.Errorf("注册失败: 未知错误")
	}

	if response.Type != protocol.MessageTypeRegisterAck {
		return fmt.Errorf("注册失败: 收到未知响应类型 %s", response.Type)
	}

	var registerResp protocol.RegisterResponse
	if err := json.Unmarshal(response.Data, &registerResp); err != nil {
		return fmt.Errorf("解析注册响应失败: %w", err)
	}

	log.Printf("注册成功: AgentID=%s, Status=%s", registerResp.AgentID, registerResp.Status)
	return nil
}

// heartbeatLoop 心跳循环
func (a *Agent) heartbeatLoop(ctx context.Context, conn *safeConn, done chan struct{}) error {
	ticker := time.NewTicker(a.cfg.GetHeartbeatInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			msg := protocol.Message{
				Type: protocol.MessageTypeHeartbeat,
				Data: json.RawMessage(`{}`),
			}
			if err := conn.WriteJSON(msg); err != nil {
				return fmt.Errorf("发送心跳失败: %w", err)
			}
			log.Println("💓 心跳已发送")
		case <-done:
			return nil
		case <-ctx.Done():
			return nil
		}
	}
}

// metricsLoop 指标采集循环
func (a *Agent) metricsLoop(ctx context.Context, conn *safeConn, manager *collector.Manager, done chan struct{}) error {
	// 立即采集一次动态数据
	if err := a.collectAndSendAllMetrics(conn, manager); err != nil {
		log.Printf("⚠️  初始数据采集失败: %v", err)
	}

	// 定时采集动态指标
	ticker := time.NewTicker(a.cfg.GetCollectorInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 采集并发送各种动态指标
			if err := a.collectAndSendAllMetrics(conn, manager); err != nil {
				return fmt.Errorf("数据采集失败: %w", err)
			}
		case <-done:
			return nil
		case <-ctx.Done():
			return nil
		}
	}
}

// collectAndSendAllMetrics 采集并发送所有动态指标
func (a *Agent) collectAndSendAllMetrics(conn *safeConn, manager *collector.Manager) error {
	var hasError bool

	// CPU 动态指标
	if err := manager.CollectAndSendCPU(conn); err != nil {
		log.Printf("⚠️  发送CPU指标失败: %v", err)
		hasError = true
	}

	// 内存动态指标
	if err := manager.CollectAndSendMemory(conn); err != nil {
		log.Printf("⚠️  发送内存指标失败: %v", err)
		hasError = true
	}

	// 磁盘指标
	if err := manager.CollectAndSendDisk(conn); err != nil {
		log.Printf("⚠️  发送磁盘指标失败: %v", err)
		hasError = true
	}

	// 磁盘 IO 指标
	if err := manager.CollectAndSendDiskIO(conn); err != nil {
		log.Printf("⚠️  发送磁盘IO指标失败: %v", err)
		hasError = true
	}

	// 网络指标
	if err := manager.CollectAndSendNetwork(conn); err != nil {
		log.Printf("⚠️  发送网络指标失败: %v", err)
		hasError = true
	}

	// 系统负载指标
	if err := manager.CollectAndSendLoad(conn); err != nil {
		log.Printf("⚠️  发送负载指标失败: %v", err)
		hasError = true
	}

	// 主机信息
	if err := manager.CollectAndSendHost(conn); err != nil {
		log.Printf("⚠️  发送主机信息失败: %v", err)
		hasError = true
	}

	// GPU 信息（可选）
	if err := manager.CollectAndSendGPU(conn); err != nil {
		log.Printf("ℹ️  发送GPU信息失败: %v", err)
	}

	// 温度信息（可选）
	if err := manager.CollectAndSendTemperature(conn); err != nil {
		log.Printf("ℹ️  发送温度信息失败: %v", err)
	}

	if hasError {
		return fmt.Errorf("部分指标采集失败")
	}

	return nil
}

// handleCommand 处理服务端下发的指令
func (a *Agent) handleCommand(conn *websocket.Conn, data json.RawMessage) {
	var cmdReq protocol.CommandRequest
	if err := json.Unmarshal(data, &cmdReq); err != nil {
		log.Printf("⚠️  解析指令失败: %v", err)
		return
	}

	log.Printf("📥 收到指令: %s (ID: %s)", cmdReq.Type, cmdReq.ID)

	// 发送运行中状态
	a.sendCommandResponse(conn, cmdReq.ID, cmdReq.Type, "running", "", "")

	switch cmdReq.Type {
	case "vps_audit":
		a.handleVPSAudit(conn, cmdReq.ID)
	default:
		log.Printf("⚠️  未知指令类型: %s", cmdReq.Type)
		a.sendCommandResponse(conn, cmdReq.ID, cmdReq.Type, "error", "未知指令类型", "")
	}
}

// handleVPSAudit 处理VPS安全审计指令
func (a *Agent) handleVPSAudit(conn *websocket.Conn, cmdID string) {
	// 导入 audit 包
	result, err := a.runVPSAudit()
	if err != nil {
		log.Printf("❌ VPS安全审计失败: %v", err)
		a.sendCommandResponse(conn, cmdID, "vps_audit", "error", err.Error(), "")
		return
	}

	// 将结果序列化为JSON
	resultJSON, err := json.Marshal(result)
	if err != nil {
		log.Printf("❌ 序列化审计结果失败: %v", err)
		a.sendCommandResponse(conn, cmdID, "vps_audit", "error", "序列化结果失败", "")
		return
	}

	log.Println("✅ VPS安全审计完成")
	a.sendCommandResponse(conn, cmdID, "vps_audit", "success", "", string(resultJSON))
}

// runVPSAudit 运行VPS安全审计
func (a *Agent) runVPSAudit() (*protocol.VPSAuditResult, error) {
	return audit.RunAudit()
}

// sendCommandResponse 发送指令响应
func (a *Agent) sendCommandResponse(conn *websocket.Conn, cmdID, cmdType, status, errMsg, result string) {
	resp := protocol.CommandResponse{
		ID:     cmdID,
		Type:   cmdType,
		Status: status,
		Error:  errMsg,
		Result: result,
	}

	respData, err := json.Marshal(resp)
	if err != nil {
		log.Printf("⚠️  序列化指令响应失败: %v", err)
		return
	}

	msg := protocol.Message{
		Type: protocol.MessageTypeCommandResp,
		Data: respData,
	}

	msgData, err := json.Marshal(msg)
	if err != nil {
		log.Printf("⚠️  序列化消息失败: %v", err)
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, msgData); err != nil {
		log.Printf("⚠️  发送指令响应失败: %v", err)
	}
}

// GetVersion 获取版本号
func GetVersion() string {
	return version.GetVersion()
}
