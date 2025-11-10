package version

// Version 版本号（通过 -ldflags 注入）
var Version = "dev"

// GetVersion 获取版本号
func GetVersion() string {
	if Version == "" {
		return "dev"
	}
	return Version
}
