package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dushixiang/pika/internal/models"
	"github.com/dushixiang/pika/internal/repo"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AlertService 告警服务
type AlertService struct {
	alertRepo *repo.AlertRepo
	agentRepo *repo.AgentRepo
	notifier  *Notifier
	logger    *zap.Logger

	// 告警状态缓存（内存中维护）
	states map[string]*models.AlertState
	mu     sync.RWMutex
}

func NewAlertService(alertRepo *repo.AlertRepo, agentRepo *repo.AgentRepo, notifier *Notifier, logger *zap.Logger) *AlertService {
	return &AlertService{
		alertRepo: alertRepo,
		agentRepo: agentRepo,
		notifier:  notifier,
		logger:    logger,
		states:    make(map[string]*models.AlertState),
	}
}

// CreateAlertConfig 创建告警配置
func (s *AlertService) CreateAlertConfig(ctx context.Context, config *models.AlertConfig) error {
	config.ID = uuid.New().String()
	config.CreatedAt = time.Now().UnixMilli()
	config.UpdatedAt = time.Now().UnixMilli()

	return s.alertRepo.CreateAlertConfig(ctx, config)
}

// UpdateAlertConfig 更新告警配置
func (s *AlertService) UpdateAlertConfig(ctx context.Context, config *models.AlertConfig) error {
	config.UpdatedAt = time.Now().UnixMilli()
	return s.alertRepo.UpdateAlertConfig(ctx, config)
}

// DeleteAlertConfig 删除告警配置
func (s *AlertService) DeleteAlertConfig(ctx context.Context, id string) error {
	// 清理该配置相关的状态
	s.mu.Lock()
	for key, state := range s.states {
		if state.ConfigID == id {
			delete(s.states, key)
		}
	}
	s.mu.Unlock()

	return s.alertRepo.DeleteAlertConfig(ctx, id)
}

// GetAlertConfig 获取告警配置
func (s *AlertService) GetAlertConfig(ctx context.Context, id string) (*models.AlertConfig, error) {
	return s.alertRepo.GetAlertConfig(ctx, id)
}

// ListAlertConfigsByAgent 列出探针的告警配置
func (s *AlertService) ListAlertConfigsByAgent(ctx context.Context, agentID string) ([]models.AlertConfig, error) {
	return s.alertRepo.FindByAgentID(ctx, agentID)
}

// ListAlertRecords 列出告警记录
func (s *AlertService) ListAlertRecords(ctx context.Context, agentID string, limit int, offset int) ([]models.AlertRecord, int64, error) {
	return s.alertRepo.ListAlertRecords(ctx, agentID, limit, offset)
}

// CheckMetrics 检查指标并触发告警
func (s *AlertService) CheckMetrics(ctx context.Context, agentID string, cpu, memory, disk float64) error {
	// 获取全局告警配置
	globalConfigs, err := s.alertRepo.FindEnabledByAgentID(ctx, "global")
	if err != nil {
		s.logger.Error("获取全局告警配置失败", zap.Error(err))
		return err
	}

	// 过滤出适用于该探针的配置
	var configs []models.AlertConfig
	for _, config := range globalConfigs {
		// 如果 AgentIDs 为空，表示监控所有探针
		if len(config.AgentIDs) == 0 {
			configs = append(configs, config)
			continue
		}
		// 如果 AgentIDs 不为空，检查当前探针是否在列表中
		for _, id := range config.AgentIDs {
			if id == agentID {
				configs = append(configs, config)
				break
			}
		}
	}

	if len(configs) == 0 {
		return nil
	}

	// 获取探针信息（用于发送通知）
	agent, err := s.agentRepo.FindById(ctx, agentID)
	if err != nil {
		s.logger.Error("获取探针信息失败", zap.Error(err))
		return err
	}

	now := time.Now().UnixMilli()

	// 检查每个配置的告警规则
	for _, config := range configs {
		// 检查 CPU 告警
		if config.Rules.CPUEnabled {
			s.checkAlert(ctx, &config, &agent, "cpu", cpu, config.Rules.CPUThreshold, config.Rules.CPUDuration, now)
		}

		// 检查内存告警
		if config.Rules.MemoryEnabled {
			s.checkAlert(ctx, &config, &agent, "memory", memory, config.Rules.MemoryThreshold, config.Rules.MemoryDuration, now)
		}

		// 检查磁盘告警
		if config.Rules.DiskEnabled {
			s.checkAlert(ctx, &config, &agent, "disk", disk, config.Rules.DiskThreshold, config.Rules.DiskDuration, now)
		}
	}

	return nil
}

// checkAlert 检查单个告警规则
func (s *AlertService) checkAlert(ctx context.Context, config *models.AlertConfig, agent *models.Agent, alertType string, currentValue, threshold float64, duration int, now int64) {
	stateKey := fmt.Sprintf("%s:%s:%s", config.AgentID, config.ID, alertType)

	s.mu.Lock()
	state, exists := s.states[stateKey]
	if !exists {
		state = &models.AlertState{
			AgentID:   config.AgentID,
			ConfigID:  config.ID,
			AlertType: alertType,
			Threshold: threshold,
			Duration:  duration,
		}
		s.states[stateKey] = state
	}
	s.mu.Unlock()

	// 更新当前值和检查时间
	state.Value = currentValue
	state.LastCheckTime = now

	// 判断是否超过阈值
	if currentValue >= threshold {
		// 超过阈值
		if state.StartTime == 0 {
			// 首次超过阈值，记录开始时间
			state.StartTime = now
		}

		// 计算已持续时间（秒）
		elapsedSeconds := (now - state.StartTime) / 1000

		// 判断是否达到持续时间要求
		if elapsedSeconds >= int64(duration) {
			// 达到持续时间要求，触发告警
			if !state.IsFiring {
				// 从未触发状态变为触发状态
				s.fireAlert(ctx, config, agent, state)
			}
		}
	} else {
		// 未超过阈值
		if state.IsFiring {
			// 从触发状态变为恢复状态
			s.resolveAlert(ctx, config, agent, state)
		}

		// 重置开始时间
		state.StartTime = 0
	}
}

// fireAlert 触发告警
func (s *AlertService) fireAlert(ctx context.Context, config *models.AlertConfig, agent *models.Agent, state *models.AlertState) {
	s.logger.Info("触发告警",
		zap.String("agentId", agent.ID),
		zap.String("agentName", agent.Name),
		zap.String("configId", config.ID),
		zap.String("alertType", state.AlertType),
		zap.Float64("value", state.Value),
		zap.Float64("threshold", state.Threshold),
	)

	// 创建告警记录
	record := &models.AlertRecord{
		AgentID:     agent.ID,
		ConfigID:    config.ID,
		ConfigName:  config.Name,
		AlertType:   state.AlertType,
		Message:     s.buildAlertMessage(state),
		Threshold:   state.Threshold,
		ActualValue: state.Value,
		Level:       s.calculateLevel(state.Value, state.Threshold),
		Status:      "firing",
		FiredAt:     time.Now().UnixMilli(),
		CreatedAt:   time.Now().UnixMilli(),
	}

	err := s.alertRepo.CreateAlertRecord(ctx, record)
	if err != nil {
		s.logger.Error("创建告警记录失败", zap.Error(err))
		return
	}

	// 更新状态
	state.IsFiring = true
	state.LastRecordID = record.ID

	// 发送通知
	go func() {
		if err := s.notifier.SendNotification(context.Background(), config, record, agent); err != nil {
			s.logger.Error("发送告警通知失败", zap.Error(err))
		}
	}()
}

// resolveAlert 恢复告警
func (s *AlertService) resolveAlert(ctx context.Context, config *models.AlertConfig, agent *models.Agent, state *models.AlertState) {
	s.logger.Info("告警恢复",
		zap.String("agentId", agent.ID),
		zap.String("agentName", agent.Name),
		zap.String("configId", config.ID),
		zap.String("alertType", state.AlertType),
		zap.Float64("value", state.Value),
	)

	// 更新告警记录状态
	if state.LastRecordID > 0 {
		// 先查询完整记录
		existingRecord, err := s.alertRepo.GetLatestAlertRecord(ctx, config.ID, state.AlertType)
		if err == nil && existingRecord != nil {
			existingRecord.Status = "resolved"
			existingRecord.ActualValue = state.Value
			existingRecord.ResolvedAt = time.Now().UnixMilli()
			existingRecord.UpdatedAt = time.Now().UnixMilli()

			err = s.alertRepo.UpdateAlertRecord(ctx, existingRecord)
			if err != nil {
				s.logger.Error("更新告警记录失败", zap.Error(err))
			} else {
				// 发送恢复通知
				go func() {
					if err := s.notifier.SendNotification(context.Background(), config, existingRecord, agent); err != nil {
						s.logger.Error("发送恢复通知失败", zap.Error(err))
					}
				}()
			}
		}
	}

	// 更新状态
	state.IsFiring = false
	state.LastRecordID = 0
}

// buildAlertMessage 构建告警消息
func (s *AlertService) buildAlertMessage(state *models.AlertState) string {
	var alertTypeName string
	switch state.AlertType {
	case "cpu":
		alertTypeName = "CPU使用率"
	case "memory":
		alertTypeName = "内存使用率"
	case "disk":
		alertTypeName = "磁盘使用率"
	case "network":
		alertTypeName = "网络连接"
	default:
		alertTypeName = state.AlertType
	}

	return fmt.Sprintf("%s持续%d秒超过%.2f%%，当前值%.2f%%",
		alertTypeName,
		state.Duration,
		state.Threshold,
		state.Value,
	)
}

// calculateLevel 计算告警级别
func (s *AlertService) calculateLevel(value, threshold float64) string {
	diff := value - threshold

	if diff < 20 {
		return "info"
	} else if diff < 50 {
		return "warning"
	} else {
		return "critical"
	}
}

// TestNotification 测试告警通知
func (s *AlertService) TestNotification(ctx context.Context, configID string) error {
	config, err := s.alertRepo.GetAlertConfig(ctx, configID)
	if err != nil {
		return fmt.Errorf("获取告警配置失败: %w", err)
	}

	agent, err := s.agentRepo.FindById(ctx, config.AgentID)
	if err != nil {
		return fmt.Errorf("获取探针信息失败: %w", err)
	}

	// 构造测试告警记录
	testRecord := &models.AlertRecord{
		AgentID:     agent.ID,
		ConfigID:    config.ID,
		ConfigName:  config.Name,
		AlertType:   "cpu",
		Message:     "这是一条测试告警消息",
		Threshold:   70.0,
		ActualValue: 75.0,
		Level:       "warning",
		Status:      "firing",
		FiredAt:     time.Now().UnixMilli(),
	}

	return s.notifier.SendNotification(ctx, config, testRecord, &agent)
}
