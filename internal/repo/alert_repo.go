package repo

import (
	"context"
	"encoding/json"

	"github.com/dushixiang/pika/internal/models"
	"github.com/go-orz/orz"
	"gorm.io/gorm"
)

type AlertRepo struct {
	orz.Repository[models.AlertConfig, string]
	db *gorm.DB
}

func NewAlertRepo(db *gorm.DB) *AlertRepo {
	return &AlertRepo{
		Repository: orz.NewRepository[models.AlertConfig, string](db),
		db:         db,
	}
}

// FindByAgentID 根据探针ID查找告警配置
func (r *AlertRepo) FindByAgentID(ctx context.Context, agentID string) ([]models.AlertConfig, error) {
	var configs []models.AlertConfig
	err := r.db.WithContext(ctx).
		Where("agent_id = ?", agentID).
		Find(&configs).Error

	// 处理 EmailAddresses 和 AgentIDs 的反序列化
	for i := range configs {
		if configs[i].Notification.EmailAddressesStr != "" {
			_ = json.Unmarshal([]byte(configs[i].Notification.EmailAddressesStr), &configs[i].Notification.EmailAddresses)
		}
		if configs[i].AgentIDsStr != "" {
			_ = json.Unmarshal([]byte(configs[i].AgentIDsStr), &configs[i].AgentIDs)
		}
	}

	return configs, err
}

// FindEnabledByAgentID 根据探针ID查找已启用的告警配置
func (r *AlertRepo) FindEnabledByAgentID(ctx context.Context, agentID string) ([]models.AlertConfig, error) {
	var configs []models.AlertConfig
	err := r.db.WithContext(ctx).
		Where("agent_id = ? AND enabled = ?", agentID, true).
		Find(&configs).Error

	// 处理 EmailAddresses 和 AgentIDs 的反序列化
	for i := range configs {
		if configs[i].Notification.EmailAddressesStr != "" {
			_ = json.Unmarshal([]byte(configs[i].Notification.EmailAddressesStr), &configs[i].Notification.EmailAddresses)
		}
		if configs[i].AgentIDsStr != "" {
			_ = json.Unmarshal([]byte(configs[i].AgentIDsStr), &configs[i].AgentIDs)
		}
	}

	return configs, err
}

// FindAllEnabled 查找所有已启用的告警配置
func (r *AlertRepo) FindAllEnabled(ctx context.Context) ([]models.AlertConfig, error) {
	var configs []models.AlertConfig
	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Find(&configs).Error

	// 处理 EmailAddresses 和 AgentIDs 的反序列化
	for i := range configs {
		if configs[i].Notification.EmailAddressesStr != "" {
			_ = json.Unmarshal([]byte(configs[i].Notification.EmailAddressesStr), &configs[i].Notification.EmailAddresses)
		}
		if configs[i].AgentIDsStr != "" {
			_ = json.Unmarshal([]byte(configs[i].AgentIDsStr), &configs[i].AgentIDs)
		}
	}

	return configs, err
}

// CreateAlertConfig 创建告警配置
func (r *AlertRepo) CreateAlertConfig(ctx context.Context, config *models.AlertConfig) error {
	// 序列化 EmailAddresses 和 AgentIDs
	if len(config.Notification.EmailAddresses) > 0 {
		data, _ := json.Marshal(config.Notification.EmailAddresses)
		config.Notification.EmailAddressesStr = string(data)
	}
	if len(config.AgentIDs) > 0 {
		data, _ := json.Marshal(config.AgentIDs)
		config.AgentIDsStr = string(data)
	}

	return r.db.WithContext(ctx).Create(config).Error
}

// UpdateAlertConfig 更新告警配置
func (r *AlertRepo) UpdateAlertConfig(ctx context.Context, config *models.AlertConfig) error {
	// 序列化 EmailAddresses 和 AgentIDs
	if len(config.Notification.EmailAddresses) > 0 {
		data, _ := json.Marshal(config.Notification.EmailAddresses)
		config.Notification.EmailAddressesStr = string(data)
	}
	if len(config.AgentIDs) > 0 {
		data, _ := json.Marshal(config.AgentIDs)
		config.AgentIDsStr = string(data)
	}

	return r.db.WithContext(ctx).Save(config).Error
}

// DeleteAlertConfig 删除告警配置
func (r *AlertRepo) DeleteAlertConfig(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.AlertConfig{}, "id = ?", id).Error
}

// GetAlertConfig 获取告警配置
func (r *AlertRepo) GetAlertConfig(ctx context.Context, id string) (*models.AlertConfig, error) {
	var config models.AlertConfig
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&config).Error
	if err != nil {
		return nil, err
	}

	// 处理 EmailAddresses 和 AgentIDs 的反序列化
	if config.Notification.EmailAddressesStr != "" {
		_ = json.Unmarshal([]byte(config.Notification.EmailAddressesStr), &config.Notification.EmailAddresses)
	}
	if config.AgentIDsStr != "" {
		_ = json.Unmarshal([]byte(config.AgentIDsStr), &config.AgentIDs)
	}

	return &config, nil
}

// CreateAlertRecord 创建告警记录
func (r *AlertRepo) CreateAlertRecord(ctx context.Context, record *models.AlertRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

// UpdateAlertRecord 更新告警记录
func (r *AlertRepo) UpdateAlertRecord(ctx context.Context, record *models.AlertRecord) error {
	return r.db.WithContext(ctx).Save(record).Error
}

// ListAlertRecords 列出告警记录
func (r *AlertRepo) ListAlertRecords(ctx context.Context, agentID string, limit int, offset int) ([]models.AlertRecord, int64, error) {
	var records []models.AlertRecord
	var total int64

	query := r.db.WithContext(ctx).Model(&models.AlertRecord{})

	if agentID != "" {
		query = query.Where("agent_id = ?", agentID)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Order("fired_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&records).Error

	return records, total, err
}

// GetLatestAlertRecord 获取最新的告警记录
func (r *AlertRepo) GetLatestAlertRecord(ctx context.Context, configID string, alertType string) (*models.AlertRecord, error) {
	var record models.AlertRecord
	err := r.db.WithContext(ctx).
		Where("config_id = ? AND alert_type = ? AND status = ?", configID, alertType, "firing").
		Order("fired_at DESC").
		First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}
