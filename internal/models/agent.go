package models

// Agent 探针信息
type Agent struct {
	ID         string `gorm:"primaryKey" json:"id"`                  // 探针ID (UUID)
	Name       string `gorm:"index" json:"name"`                     // 探针名称
	Hostname   string `gorm:"index" json:"hostname,omitempty"`       // 主机名
	IP         string `gorm:"index" json:"ip,omitempty"`             // IP地址
	OS         string `json:"os"`                                    // 操作系统
	Arch       string `json:"arch"`                                  // 架构
	Version    string `json:"version"`                               // 探针版本
	Platform   string `json:"platform"`                              // 平台
	Location   string `json:"location"`                              // 位置
	ExpireTime int64  `json:"expireTime"`                            // 到期时间（时间戳毫秒）
	Status     int    `json:"status"`                                // 状态: 0-离线, 1-在线
	LastSeenAt int64  `gorm:"index" json:"lastSeenAt"`               // 最后上线时间（时间戳毫秒）
	CreatedAt  int64  `json:"createdAt"`                             // 创建时间（时间戳毫秒）
	UpdatedAt  int64  `json:"updatedAt" gorm:"autoUpdateTime:milli"` // 更新时间（时间戳毫秒）
}

func (Agent) TableName() string {
	return "agents"
}
