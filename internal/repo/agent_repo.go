package repo

import (
	"context"
	"time"

	"github.com/dushixiang/pika/internal/models"
	"github.com/go-orz/orz"
	"gorm.io/gorm"
)

type AgentRepo struct {
	orz.Repository[models.Agent, string]
	db *gorm.DB
}

func NewAgentRepo(db *gorm.DB) *AgentRepo {
	return &AgentRepo{
		Repository: orz.NewRepository[models.Agent, string](db),
		db:         db,
	}
}

// UpdateStatus 更新探针状态
func (r *AgentRepo) UpdateStatus(ctx context.Context, agentID string, status int) error {
	return r.db.WithContext(ctx).
		Model(&models.Agent{}).
		Where("id = ?", agentID).
		Updates(map[string]interface{}{
			"status":       status,
			"last_seen_at": time.Now().UnixMilli(),
		}).Error
}

// FindOnlineAgents 查找所有在线探针
func (r *AgentRepo) FindOnlineAgents(ctx context.Context) ([]models.Agent, error) {
	var agents []models.Agent
	err := r.db.WithContext(ctx).
		Where("status = ?", 1).
		Find(&agents).Error
	return agents, err
}

// FindByIP 根据IP查找探针
func (r *AgentRepo) FindByIP(ctx context.Context, ip string) (*models.Agent, error) {
	var agent models.Agent
	err := r.db.WithContext(ctx).
		Where("ip = ?", ip).
		First(&agent).Error
	if err != nil {
		return nil, err
	}
	return &agent, nil
}

// FindByHostname 根据主机名查找探针
func (r *AgentRepo) FindByHostname(ctx context.Context, hostname string) (*models.Agent, error) {
	var agent models.Agent
	err := r.db.WithContext(ctx).
		Where("hostname = ?", hostname).
		First(&agent).Error
	if err != nil {
		return nil, err
	}
	return &agent, nil
}

// FindByHostnameAndIP 根据主机名和IP查找探针
func (r *AgentRepo) FindByHostnameAndIP(ctx context.Context, hostname, ip string) (*models.Agent, error) {
	var agent models.Agent
	err := r.db.WithContext(ctx).
		Where("hostname = ? AND ip = ?", hostname, ip).
		First(&agent).Error
	if err != nil {
		return nil, err
	}
	return &agent, nil
}

// SaveAuditResult 保存审计结果
func (r *AgentRepo) SaveAuditResult(ctx context.Context, audit *models.AuditResult) error {
	return r.db.WithContext(ctx).Create(audit).Error
}

// GetLatestAuditResult 获取最新的审计结果
func (r *AgentRepo) GetLatestAuditResult(ctx context.Context, agentID string) (*models.AuditResult, error) {
	var audit models.AuditResult
	err := r.db.WithContext(ctx).
		Where("agent_id = ?", agentID).
		Order("created_at DESC").
		First(&audit).Error
	if err != nil {
		return nil, err
	}
	return &audit, nil
}

// ListAuditResults 获取审计结果列表
func (r *AgentRepo) ListAuditResults(ctx context.Context, agentID string) ([]models.AuditResult, error) {
	var audits []models.AuditResult
	err := r.db.WithContext(ctx).
		Where("agent_id = ?", agentID).
		Order("created_at DESC").
		Limit(50).
		Find(&audits).Error
	return audits, err
}

// UpdateName 更新探针名称
func (r *AgentRepo) UpdateName(ctx context.Context, agentID string, name string) error {
	return r.db.WithContext(ctx).
		Model(&models.Agent{}).
		Where("id = ?", agentID).
		Update("name", name).Error
}

// UpdateInfo 更新探针信息（名称、平台、位置、到期时间）
func (r *AgentRepo) UpdateInfo(ctx context.Context, agentID string, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).
		Model(&models.Agent{}).
		Where("id = ?", agentID).
		Updates(updates).Error
}

// GetStatistics 获取探针统计数据
func (r *AgentRepo) GetStatistics(ctx context.Context) (total int64, online int64, err error) {
	// 获取总数
	err = r.db.WithContext(ctx).
		Model(&models.Agent{}).
		Count(&total).Error
	if err != nil {
		return 0, 0, err
	}

	// 获取在线数量
	err = r.db.WithContext(ctx).
		Model(&models.Agent{}).
		Where("status = ?", 1).
		Count(&online).Error

	return total, online, err
}
