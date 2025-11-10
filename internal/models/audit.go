package models

// AuditResult 审计结果模型
type AuditResult struct {
	ID        int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	AgentID   string `gorm:"type:varchar(64);not null;index" json:"agentId"`
	Type      string `gorm:"type:varchar(32);not null" json:"type"` // vps_audit
	Result    string `gorm:"type:text;not null" json:"result"`      // JSON格式的审计结果
	StartTime int64  `gorm:"not null" json:"startTime"`
	EndTime   int64  `gorm:"not null" json:"endTime"`
	CreatedAt int64  `gorm:"not null" json:"createdAt"`
}

// TableName 表名
func (AuditResult) TableName() string {
	return "audit_results"
}
