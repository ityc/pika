package models

// AlertConfig 告警配置
type AlertConfig struct {
	ID          string   `gorm:"primaryKey" json:"id"`                  // 告警配置ID (UUID)
	AgentID     string   `gorm:"index" json:"agentId"`                  // 探针ID（全局配置使用"global"）
	AgentIDs    []string `gorm:"-" json:"agentIds"`                     // 监控的探针ID列表（前端使用）
	AgentIDsStr string   `json:"-" gorm:"column:agent_ids"`             // 数据库存储（JSON序列化）
	Name        string   `json:"name"`                                  // 告警配置名称
	Enabled     bool     `json:"enabled"`                               // 是否启用
	CreatedAt   int64    `json:"createdAt"`                             // 创建时间（时间戳毫秒）
	UpdatedAt   int64    `json:"updatedAt" gorm:"autoUpdateTime:milli"` // 更新时间（时间戳毫秒）

	// 告警规则
	Rules AlertRules `gorm:"embedded;embeddedPrefix:rule_" json:"rules"`

	// 告警通知配置
	Notification NotificationConfig `gorm:"embedded;embeddedPrefix:notify_" json:"notification"`
}

// AlertRules 告警规则
type AlertRules struct {
	// CPU 告警配置
	CPUEnabled   bool    `json:"cpuEnabled"`   // 是否启用CPU告警
	CPUThreshold float64 `json:"cpuThreshold"` // CPU使用率阈值(0-100)
	CPUDuration  int     `json:"cpuDuration"`  // 持续时间（秒）

	// 内存告警配置
	MemoryEnabled   bool    `json:"memoryEnabled"`   // 是否启用内存告警
	MemoryThreshold float64 `json:"memoryThreshold"` // 内存使用率阈值(0-100)
	MemoryDuration  int     `json:"memoryDuration"`  // 持续时间（秒）

	// 磁盘告警配置
	DiskEnabled   bool    `json:"diskEnabled"`   // 是否启用磁盘告警
	DiskThreshold float64 `json:"diskThreshold"` // 磁盘使用率阈值(0-100)
	DiskDuration  int     `json:"diskDuration"`  // 持续时间（秒）

	// 网络断开告警配置
	NetworkEnabled  bool `json:"networkEnabled"`  // 是否启用网络断开告警
	NetworkDuration int  `json:"networkDuration"` // 持续时间（秒）
}

// NotificationConfig 告警通知配置
type NotificationConfig struct {
	// 钉钉配置
	DingTalkEnabled bool   `json:"dingTalkEnabled"` // 是否启用钉钉通知
	DingTalkWebhook string `json:"dingTalkWebhook"` // 钉钉Webhook URL
	DingTalkSecret  string `json:"dingTalkSecret"`  // 钉钉加签密钥（可选）

	// 企业微信配置
	WeComEnabled bool   `json:"weComEnabled"` // 是否启用企业微信通知
	WeComWebhook string `json:"weComWebhook"` // 企业微信Webhook URL

	// 飞书配置
	FeishuEnabled bool   `json:"feishuEnabled"` // 是否启用飞书通知
	FeishuWebhook string `json:"feishuWebhook"` // 飞书Webhook URL

	// 邮件配置
	EmailEnabled      bool     `json:"emailEnabled"`                    // 是否启用邮件通知
	EmailAddresses    []string `gorm:"-" json:"emailAddresses"`         // 接收邮件地址列表
	EmailAddressesStr string   `json:"-" gorm:"column:email_addresses"` // 数据库存储（JSON序列化）

	// 自定义Webhook配置
	CustomWebhookEnabled bool   `json:"customWebhookEnabled"` // 是否启用自定义Webhook
	CustomWebhookURL     string `json:"customWebhookUrl"`     // 自定义Webhook URL
}

func (AlertConfig) TableName() string {
	return "alert_configs"
}

// AlertRecord 告警记录
type AlertRecord struct {
	ID          int64   `gorm:"primaryKey;autoIncrement" json:"id"`    // 记录ID
	AgentID     string  `gorm:"index" json:"agentId"`                  // 探针ID
	ConfigID    string  `gorm:"index" json:"configId"`                 // 告警配置ID
	ConfigName  string  `json:"configName"`                            // 告警配置名称
	AlertType   string  `json:"alertType"`                             // 告警类型: cpu, memory, disk, network
	Message     string  `json:"message"`                               // 告警消息
	Threshold   float64 `json:"threshold"`                             // 告警阈值
	ActualValue float64 `json:"actualValue"`                           // 实际值
	Level       string  `json:"level"`                                 // 告警级别: info, warning, critical
	Status      string  `json:"status"`                                // 状态: firing（告警中）, resolved（已恢复）
	FiredAt     int64   `gorm:"index" json:"firedAt"`                  // 触发时间（时间戳毫秒）
	ResolvedAt  int64   `json:"resolvedAt,omitempty"`                  // 恢复时间（时间戳毫秒）
	CreatedAt   int64   `json:"createdAt"`                             // 创建时间（时间戳毫秒）
	UpdatedAt   int64   `json:"updatedAt" gorm:"autoUpdateTime:milli"` // 更新时间（时间戳毫秒）
}

func (AlertRecord) TableName() string {
	return "alert_records"
}

// AlertState 告警状态（内存中保存，用于判断是否持续超过阈值）
type AlertState struct {
	AgentID       string  // 探针ID
	ConfigID      string  // 告警配置ID
	AlertType     string  // 告警类型
	Value         float64 // 当前值
	Threshold     float64 // 阈值
	StartTime     int64   // 开始超过阈值的时间
	Duration      int     // 需要持续的时间（秒）
	LastCheckTime int64   // 上次检查时间
	IsFiring      bool    // 是否正在告警
	LastRecordID  int64   // 最后一条告警记录ID
}
