package models

// User 用户信息
type User struct {
	ID        string `gorm:"primaryKey" json:"id"`                  // 用户ID (UUID)
	Username  string `gorm:"uniqueIndex" json:"username"`           // 用户名
	Password  string `json:"-"`                                     // 密码（加密存储，不返回给前端）
	Nickname  string `json:"nickname"`                              // 昵称
	CreatedAt int64  `json:"createdAt"`                             // 创建时间（时间戳毫秒）
	UpdatedAt int64  `json:"updatedAt" gorm:"autoUpdateTime:milli"` // 更新时间（时间戳毫秒）
}

func (User) TableName() string {
	return "users"
}
