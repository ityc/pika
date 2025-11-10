package config

// AppConfig 应用配置
type AppConfig struct {
	JWT JWTConfig `mapstructure:"JWT"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret       string `mapstructure:"Secret"`
	ExpiresHours int    `mapstructure:"ExpiresHours"`
}
