package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/dushixiang/pika"
	"github.com/dushixiang/pika/internal/models"
	"github.com/dushixiang/pika/internal/protocol"
	"github.com/dushixiang/pika/internal/service"
	ws "github.com/dushixiang/pika/internal/websocket"
	"github.com/dushixiang/pika/pkg/version"
	"github.com/go-orz/orz"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type AgentHandler struct {
	logger       *zap.Logger
	agentService *service.AgentService
	wsManager    *ws.Manager
	upgrader     websocket.Upgrader
}

func NewAgentHandler(logger *zap.Logger, agentService *service.AgentService, wsManager *ws.Manager) *AgentHandler {

	h := &AgentHandler{
		logger:       logger,
		agentService: agentService,
		wsManager:    wsManager,
	}

	// 初始化upgrader，需要在创建handler之后因为需要引用h.checkOrigin
	h.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024 * 32,
		WriteBufferSize: 1024 * 32,
	}

	// 设置WebSocket消息处理器
	wsManager.SetMessageHandler(h.handleWebSocketMessage)

	return h
}

// HandleWebSocket 处理WebSocket连接
func (h *AgentHandler) HandleWebSocket(c echo.Context) error {
	conn, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		h.logger.Error("failed to upgrade websocket", zap.Error(err))
		return err
	}

	// 等待探针发送注册信息
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, message, err := conn.ReadMessage()
	if err != nil {
		h.logger.Error("failed to read register message", zap.Error(err))
		conn.Close()
		return err
	}

	// 清除读取超时，后续由心跳机制控制
	conn.SetReadDeadline(time.Time{})

	// 解析注册消息
	var msg protocol.Message
	if err := json.Unmarshal(message, &msg); err != nil {
		h.logger.Error("failed to parse register message", zap.Error(err))
		conn.Close()
		return err
	}

	if msg.Type != protocol.MessageTypeRegister {
		h.logger.Error("first message must be register", zap.String("type", string(msg.Type)))
		conn.Close()
		return echo.NewHTTPError(http.StatusBadRequest, "首条消息必须是注册消息")
	}

	// 解析探针注册信息
	var registerReq protocol.RegisterRequest
	if err := json.Unmarshal(msg.Data, &registerReq); err != nil {
		h.logger.Error("failed to parse register request", zap.Error(err))
		conn.Close()
		return err
	}

	// 注册探针 - 使用独立的context,不依赖HTTP请求的context
	ctx := context.Background()
	agent, err := h.agentService.RegisterAgent(ctx, c.RealIP(), &registerReq.AgentInfo, registerReq.ApiKey)
	if err != nil {
		h.logger.Error("failed to register agent", zap.Error(err))

		// 发送注册失败响应
		h.sendRegisterError(conn, err.Error())
		conn.Close()
		return err
	}

	defer func() {
		// 设置探针状态为离线
		_ = h.agentService.UpdateAgentStatus(ctx, agent.ID, 0)
	}()

	// 发送注册成功响应
	if err := h.sendRegisterSuccess(conn, agent.ID); err != nil {
		h.logger.Error("failed to send register ack", zap.Error(err))
		conn.Close()
		return err
	}

	// 创建客户端并注册到管理器
	client := &ws.Client{
		ID:         agent.ID,
		Conn:       conn,
		Send:       make(chan []byte, 256),
		Manager:    h.wsManager,
		LastActive: time.Now(),
	}

	h.wsManager.Register(client)

	// 启动读写协程
	go client.WritePump()
	go client.ReadPump(ctx)

	return nil
}

// handleWebSocketMessage 处理WebSocket消息
func (h *AgentHandler) handleWebSocketMessage(ctx context.Context, agentID string, messageType string, data json.RawMessage) error {
	switch protocol.MessageType(messageType) {
	case protocol.MessageTypeHeartbeat:
		// 心跳消息，更新探针状态
		return h.agentService.UpdateAgentStatus(ctx, agentID, 1)

	case protocol.MessageTypeMetrics:
		// 指标数据
		var metricsWrapper protocol.MetricsWrapper
		if err := json.Unmarshal(data, &metricsWrapper); err != nil {
			return err
		}
		return h.agentService.HandleMetricData(ctx, agentID, string(metricsWrapper.Type), metricsWrapper.Data)

	case protocol.MessageTypeCommandResp:
		// 指令响应
		var cmdResp protocol.CommandResponse
		if err := json.Unmarshal(data, &cmdResp); err != nil {
			return err
		}
		return h.agentService.HandleCommandResponse(ctx, agentID, &cmdResp)

	default:
		h.logger.Warn("unknown message type", zap.String("type", messageType))
		return nil
	}
}

// sendRegisterSuccess 发送注册成功响应
func (h *AgentHandler) sendRegisterSuccess(conn *websocket.Conn, agentID string) error {
	resp := protocol.RegisterResponse{
		AgentID: agentID,
		Status:  "success",
	}
	respData, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	msg := protocol.Message{
		Type: protocol.MessageTypeRegisterAck,
		Data: respData,
	}
	msgData, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, msgData)
}

// sendRegisterError 发送注册失败响应
func (h *AgentHandler) sendRegisterError(conn *websocket.Conn, errMsg string) error {
	resp := protocol.RegisterResponse{
		Status:  "error",
		Message: errMsg,
	}
	respData, _ := json.Marshal(resp)

	msg := protocol.Message{
		Type: protocol.MessageTypeRegisterErr,
		Data: respData,
	}
	msgData, _ := json.Marshal(msg)

	return conn.WriteMessage(websocket.TextMessage, msgData)
}

// Paging 探针分页查询
func (h *AgentHandler) Paging(c echo.Context) error {
	hostname := c.QueryParam("hostname")
	ip := c.QueryParam("ip")
	status := c.QueryParam("status")

	pr := orz.GetPageRequest(c, "last_seen_at", "hostname")

	builder := orz.NewPageBuilder(h.agentService.Repository).
		PageRequest(pr).
		Contains("hostname", hostname).
		Contains("ip", ip)

	// 处理状态筛选
	if status == "online" {
		builder.Equal("status", "1")
	} else if status == "offline" {
		builder.Equal("status", "0")
	}

	ctx := c.Request().Context()
	page, err := builder.Execute(ctx)
	if err != nil {
		return err
	}
	return orz.Ok(c, page)
}

// Get 获取探针详情（公开接口，隐藏敏感信息）
func (h *AgentHandler) Get(c echo.Context) error {
	id := c.Param("id")
	ctx := c.Request().Context()

	agent, err := h.agentService.GetAgent(ctx, id)
	if err != nil {
		return err
	}
	agent.IP = ""
	agent.Hostname = ""

	return orz.Ok(c, agent)
}

// GetForAdmin 获取探针详情（管理员接口，显示完整信息）
func (h *AgentHandler) GetForAdmin(c echo.Context) error {
	id := c.Param("id")
	ctx := c.Request().Context()

	agent, err := h.agentService.GetAgent(ctx, id)
	if err != nil {
		return err
	}

	return orz.Ok(c, agent)
}

// GetMetrics 获取探针聚合指标
func (h *AgentHandler) GetMetrics(c echo.Context) error {
	agentID := c.Param("id")
	metricType := c.QueryParam("type")
	rangeParam := c.QueryParam("range")

	// 验证指标类型
	validTypes := map[string]bool{
		"cpu": true, "memory": true, "disk": true, "network": true, "load": true,
	}
	if metricType == "" {
		return orz.NewError(400, "指标类型不能为空")
	}
	if !validTypes[metricType] {
		return orz.NewError(400, "无效的指标类型")
	}

	// 根据 range 参数自动计算时间范围
	var start, end int64
	end = time.Now().UnixMilli()

	if rangeParam == "" {
		rangeParam = "1h" // 默认1小时
	}

	switch rangeParam {
	case "1h":
		start = end - 1*60*60*1000
	case "6h":
		start = end - 6*60*60*1000
	case "12h":
		start = end - 12*60*60*1000
	case "24h", "1d":
		start = end - 24*60*60*1000
	case "3d":
		start = end - 3*24*60*60*1000
	case "7d", "1w":
		start = end - 7*24*60*60*1000
	case "30d", "1M":
		start = end - 30*24*60*60*1000
	default:
		return orz.NewError(400, "无效的时间范围，支持: 1h, 6h, 12h, 24h, 3d, 7d, 30d, 90d")
	}

	// 服务端自动计算最优聚合间隔
	interval := service.CalculateInterval(start, end)

	ctx := c.Request().Context()
	metrics, err := h.agentService.GetMetrics(ctx, agentID, metricType, start, end, interval)
	if err != nil {
		return err
	}

	return orz.Ok(c, orz.Map{
		"agentId":  agentID,
		"type":     metricType,
		"range":    rangeParam,
		"start":    start,
		"end":      end,
		"interval": interval,
		"metrics":  metrics,
	})
}

// GetLatestMetrics 获取探针最新指标
func (h *AgentHandler) GetLatestMetrics(c echo.Context) error {
	id := c.Param("id")
	ctx := c.Request().Context()

	metrics, err := h.agentService.GetLatestMetrics(ctx, id)
	if err != nil {
		return err
	}

	return orz.Ok(c, metrics)
}

// GetAgents 获取探针列表
func (h *AgentHandler) GetAgents(c echo.Context) error {
	ctx := c.Request().Context()
	agents, err := h.agentService.ListAgents(ctx)
	if err != nil {
		return err
	}

	slices.SortFunc(agents, func(a, b models.Agent) int {
		if a.Status == b.Status {
			return strings.Compare(a.Name, b.Name)
		}
		return strings.Compare(strconv.Itoa(a.Status), strconv.Itoa(b.Status))
	})

	// 添加连接状态和最新指标
	result := make([]map[string]interface{}, 0, len(agents))
	for _, agent := range agents {
		item := map[string]interface{}{
			"id":         agent.ID,
			"name":       agent.Name,
			"hostname":   agent.Hostname,
			"ip":         agent.IP,
			"os":         agent.OS,
			"arch":       agent.Arch,
			"version":    agent.Version,
			"platform":   agent.Platform,
			"location":   agent.Location,
			"expireTime": agent.ExpireTime,
			"status":     agent.Status,
			"lastSeenAt": agent.LastSeenAt,
		}

		// 获取最新指标数据
		metrics, err := h.agentService.GetLatestMetrics(ctx, agent.ID)
		if err == nil && metrics != nil {
			item["metrics"] = metrics
		}

		result = append(result, item)
	}

	return orz.Ok(c, orz.Map{
		"items": result,
		"total": len(result),
	})
}

// GetAgentVersion 获取 Agent 版本信息
func (h *AgentHandler) GetAgentVersion(c echo.Context) error {
	return orz.Ok(c, orz.Map{
		"version": version.GetVersion(),
	})
}

// DownloadAgent 下载 Agent 二进制文件
func (h *AgentHandler) DownloadAgent(c echo.Context) error {
	filename := c.Param("filename")

	// 从嵌入的文件系统读取
	agentFile, err := pika.AgentFS().Open(fmt.Sprintf("pika-%s", filename))
	if err != nil {
		h.logger.Error("agent binary not found", zap.String("filename", filename), zap.Error(err))
		return orz.NewError(404, "未找到对应平台的 Agent 二进制文件")
	}

	// 设置响应头
	c.Response().Header().Set("Content-Type", "application/octet-stream")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	return c.Stream(http.StatusOK, "application/octet-stream", agentFile)
}

// SendCommand 向探针发送指令
func (h *AgentHandler) SendCommand(c echo.Context) error {
	agentID := c.Param("id")
	cmdType := c.QueryParam("type")

	if cmdType == "" {
		return orz.NewError(400, "指令类型不能为空")
	}

	// 检查agent是否在线
	_, exists := h.wsManager.GetClient(agentID)
	if !exists {
		return orz.NewError(400, "探针未连接")
	}

	// 生成指令ID
	cmdID := fmt.Sprintf("%s_%d", cmdType, time.Now().UnixMilli())

	// 构建指令请求
	cmdReq := protocol.CommandRequest{
		ID:   cmdID,
		Type: cmdType,
	}

	reqData, err := json.Marshal(cmdReq)
	if err != nil {
		return err
	}

	msg := protocol.Message{
		Type: protocol.MessageTypeCommand,
		Data: reqData,
	}

	msgData, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// 发送指令
	if err := h.wsManager.SendToClient(agentID, msgData); err != nil {
		return orz.NewError(500, "发送指令失败")
	}

	h.logger.Info("command sent", zap.String("agentID", agentID), zap.String("cmdID", cmdID), zap.String("type", cmdType))

	return orz.Ok(c, orz.Map{
		"commandId": cmdID,
		"status":    "sent",
	})
}

// GetAuditResult 获取审计结果
func (h *AgentHandler) GetAuditResult(c echo.Context) error {
	agentID := c.Param("id")
	ctx := c.Request().Context()

	result, err := h.agentService.GetAuditResult(ctx, agentID)
	if err != nil {
		return err
	}

	return orz.Ok(c, result)
}

// ListAuditResults 获取审计结果列表
func (h *AgentHandler) ListAuditResults(c echo.Context) error {
	agentID := c.Param("id")
	ctx := c.Request().Context()

	results, err := h.agentService.ListAuditResults(ctx, agentID)
	if err != nil {
		return err
	}

	return orz.Ok(c, orz.Map{
		"items": results,
		"total": len(results),
	})
}

// UpdateName 更新探针名称
func (h *AgentHandler) UpdateName(c echo.Context) error {
	agentID := c.Param("id")

	var req struct {
		Name string `json:"name" validate:"required"`
	}
	if err := c.Bind(&req); err != nil {
		return orz.NewError(400, "请求参数错误")
	}
	if err := c.Validate(&req); err != nil {
		return orz.NewError(400, "名称不能为空")
	}

	ctx := c.Request().Context()
	if err := h.agentService.UpdateAgentName(ctx, agentID, req.Name); err != nil {
		return err
	}

	return orz.Ok(c, orz.Map{
		"message": "更新成功",
	})
}

// UpdateInfo 更新探针信息（名称、平台、位置、到期时间）
func (h *AgentHandler) UpdateInfo(c echo.Context) error {
	agentID := c.Param("id")

	var req struct {
		Name       *string `json:"name"`
		Platform   *string `json:"platform"`
		Location   *string `json:"location"`
		ExpireTime *int64  `json:"expireTime"`
	}
	if err := c.Bind(&req); err != nil {
		return orz.NewError(400, "请求参数错误")
	}

	// 构建更新字段
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Platform != nil {
		updates["platform"] = *req.Platform
	}
	if req.Location != nil {
		updates["location"] = *req.Location
	}
	if req.ExpireTime != nil {
		updates["expire_time"] = *req.ExpireTime
	}

	if len(updates) == 0 {
		return orz.NewError(400, "没有需要更新的字段")
	}

	ctx := c.Request().Context()
	if err := h.agentService.UpdateAgentInfo(ctx, agentID, updates); err != nil {
		return err
	}

	return orz.Ok(c, orz.Map{
		"message": "更新成功",
	})
}

// GetStatistics 获取探针统计数据
func (h *AgentHandler) GetStatistics(c echo.Context) error {
	ctx := c.Request().Context()
	stats, err := h.agentService.GetStatistics(ctx)
	if err != nil {
		return err
	}

	return orz.Ok(c, stats)
}
