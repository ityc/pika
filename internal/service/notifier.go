package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dushixiang/pika/internal/models"
	"go.uber.org/zap"
)

// Notifier 告警通知服务
type Notifier struct {
	logger *zap.Logger
}

func NewNotifier(logger *zap.Logger) *Notifier {
	return &Notifier{
		logger: logger,
	}
}

// SendNotification 发送告警通知
func (n *Notifier) SendNotification(ctx context.Context, config *models.AlertConfig, record *models.AlertRecord, agent *models.Agent) error {
	n.logger.Info("发送告警通知",
		zap.String("agentId", agent.ID),
		zap.String("agentName", agent.Name),
		zap.String("alertType", record.AlertType),
		zap.String("status", record.Status),
	)

	notification := config.Notification

	// 构造通知消息内容
	message := n.buildMessage(agent, record)

	var errs []error

	// 发送钉钉通知
	if notification.DingTalkEnabled && notification.DingTalkWebhook != "" {
		if err := n.sendDingTalk(ctx, notification.DingTalkWebhook, notification.DingTalkSecret, message); err != nil {
			n.logger.Error("发送钉钉通知失败", zap.Error(err))
			errs = append(errs, err)
		}
	}

	// 发送企业微信通知
	if notification.WeComEnabled && notification.WeComWebhook != "" {
		if err := n.sendWeCom(ctx, notification.WeComWebhook, message); err != nil {
			n.logger.Error("发送企业微信通知失败", zap.Error(err))
			errs = append(errs, err)
		}
	}

	// 发送飞书通知
	if notification.FeishuEnabled && notification.FeishuWebhook != "" {
		if err := n.sendFeishu(ctx, notification.FeishuWebhook, message); err != nil {
			n.logger.Error("发送飞书通知失败", zap.Error(err))
			errs = append(errs, err)
		}
	}

	// 发送自定义Webhook
	if notification.CustomWebhookEnabled && notification.CustomWebhookURL != "" {
		if err := n.sendCustomWebhook(ctx, notification.CustomWebhookURL, agent, record); err != nil {
			n.logger.Error("发送自定义Webhook失败", zap.Error(err))
			errs = append(errs, err)
		}
	}

	// TODO: 实现邮件通知
	// if notification.EmailEnabled && len(notification.EmailAddresses) > 0 {
	// 	if err := n.sendEmail(ctx, notification.EmailAddresses, message); err != nil {
	// 		n.logger.Error("发送邮件通知失败", zap.Error(err))
	// 		errs = append(errs, err)
	// 	}
	// }

	if len(errs) > 0 {
		return fmt.Errorf("部分通知发送失败: %v", errs)
	}

	return nil
}

// buildMessage 构建告警消息文本
func (n *Notifier) buildMessage(agent *models.Agent, record *models.AlertRecord) string {
	var message string

	// 告警级别图标
	levelIcon := ""
	switch record.Level {
	case "info":
		levelIcon = "ℹ️"
	case "warning":
		levelIcon = "⚠️"
	case "critical":
		levelIcon = "🚨"
	}

	// 告警类型名称
	alertTypeName := ""
	switch record.AlertType {
	case "cpu":
		alertTypeName = "CPU告警"
	case "memory":
		alertTypeName = "内存告警"
	case "disk":
		alertTypeName = "磁盘告警"
	case "network":
		alertTypeName = "网络断开告警"
	}

	if record.Status == "firing" {
		// 告警触发消息
		message = fmt.Sprintf(
			"%s %s\n\n"+
				"探针: %s (%s)\n"+
				"主机: %s\n"+
				"IP: %s\n"+
				"告警类型: %s\n"+
				"告警消息: %s\n"+
				"阈值: %.2f%%\n"+
				"当前值: %.2f%%\n"+
				"触发时间: %s",
			levelIcon,
			alertTypeName,
			agent.Name,
			agent.ID,
			agent.Hostname,
			agent.IP,
			record.AlertType,
			record.Message,
			record.Threshold,
			record.ActualValue,
			time.Unix(record.FiredAt/1000, 0).Format("2006-01-02 15:04:05"),
		)
	} else if record.Status == "resolved" {
		// 告警恢复消息
		message = fmt.Sprintf(
			"✅ %s已恢复\n\n"+
				"探针: %s (%s)\n"+
				"主机: %s\n"+
				"IP: %s\n"+
				"告警类型: %s\n"+
				"当前值: %.2f%%\n"+
				"恢复时间: %s",
			alertTypeName,
			agent.Name,
			agent.ID,
			agent.Hostname,
			agent.IP,
			record.AlertType,
			record.ActualValue,
			time.Unix(record.ResolvedAt/1000, 0).Format("2006-01-02 15:04:05"),
		)
	}

	return message
}

// sendDingTalk 发送钉钉通知
func (n *Notifier) sendDingTalk(ctx context.Context, webhook, secret, message string) error {
	// 构造钉钉消息体
	body := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": message,
		},
	}

	// 如果有加签密钥，计算签名
	timestamp := time.Now().UnixMilli()
	if secret != "" {
		sign := n.calculateDingTalkSign(timestamp, secret)
		webhook = fmt.Sprintf("%s&timestamp=%d&sign=%s", webhook, timestamp, sign)
	}

	return n.sendJSONRequest(ctx, webhook, body)
}

// calculateDingTalkSign 计算钉钉加签
func (n *Notifier) calculateDingTalkSign(timestamp int64, secret string) string {
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// sendWeCom 发送企业微信通知
func (n *Notifier) sendWeCom(ctx context.Context, webhook, message string) error {
	body := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": message,
		},
	}

	return n.sendJSONRequest(ctx, webhook, body)
}

// sendFeishu 发送飞书通知
func (n *Notifier) sendFeishu(ctx context.Context, webhook, message string) error {
	body := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]string{
			"text": message,
		},
	}

	return n.sendJSONRequest(ctx, webhook, body)
}

// sendCustomWebhook 发送自定义Webhook
func (n *Notifier) sendCustomWebhook(ctx context.Context, webhook string, agent *models.Agent, record *models.AlertRecord) error {
	// 发送完整的告警记录和探针信息
	body := map[string]interface{}{
		"agent":  agent,
		"record": record,
	}

	return n.sendJSONRequest(ctx, webhook, body)
}

// sendJSONRequest 发送JSON请求
func (n *Notifier) sendJSONRequest(ctx context.Context, url string, body interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("序列化请求体失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	n.logger.Info("通知发送成功", zap.String("url", url), zap.String("response", string(respBody)))
	return nil
}
