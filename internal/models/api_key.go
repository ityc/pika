package models

// ApiKey API密钥信息
type ApiKey struct {
	ID        string `gorm:"primaryKey" json:"id"`                  // 密钥ID (UUID)
	Name      string `gorm:"index" json:"name"`                     // 密钥名称/备注
	Key       string `gorm:"uniqueIndex" json:"key"`                // API密钥
	Enabled   bool   `gorm:"index;default:true" json:"enabled"`     // 是否启用
	CreatedBy string `gorm:"index" json:"createdBy"`                // 创建人ID
	CreatedAt int64  `json:"createdAt"`                             // 创建时间（时间戳毫秒）
	UpdatedAt int64  `json:"updatedAt" gorm:"autoUpdateTime:milli"` // 更新时间（时间戳毫秒）
}

func (ApiKey) TableName() string {
	return "api_keys"
}
