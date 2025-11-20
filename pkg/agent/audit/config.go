package audit

import "time"

// Config 审计配置
type Config struct {
	// 进程相关
	ProcessConfig ProcessConfig

	// SSH 相关
	SSHConfig SSHConfig

	// 网络相关
	NetworkConfig NetworkConfig

	// 文件相关
	FileConfig FileConfig

	// 定时任务相关
	CronConfig CronConfig

	// 登录相关
	LoginConfig LoginConfig

	// 风险评分配置
	ScoringConfig ScoringConfig

	// 性能相关
	PerformanceConfig PerformanceConfig
}

// ProcessConfig 进程检查配置
type ProcessConfig struct {
	// 挖矿关键词
	MinerKeywords []string

	// 高 CPU 阈值 (百分比)
	HighCPUThreshold float64

	// 高 CPU 白名单
	HighCPUWhitelist []string

	// Deleted 进程白名单
	DeletedWhitelist []string

	// 最近启动时间阈值 (小时)
	RecentStartupHours int
}

// SSHConfig SSH 检查配置
type SSHConfig struct {
	// SSH 二进制文件路径
	BinaryPaths []string

	// SSH 配置文件路径
	ConfigPaths []string

	// 最近修改时间阈值 (天)
	RecentModifyDays int

	// authorized_keys 最大文件大小 (字节)
	MaxAuthorizedKeysSize int64

	// authorized_keys 最近修改时间阈值 (天)
	AuthKeysRecentModifyDays int

	// 最大公钥数量
	MaxKeysCount int
}

// NetworkConfig 网络检查配置
type NetworkConfig struct {
	// 可疑端口映射
	SuspiciousPorts map[uint32]string

	// 挖矿池端口
	MinerPorts []uint32

	// 挖矿池域名关键词
	MinerPoolKeywords []string
}

// FileConfig 文件检查配置
type FileConfig struct {
	// 临时目录列表
	TempDirs []string

	// 可疑路径列表
	SuspiciousPaths []string

	// 关键系统二进制文件
	CriticalBinaries []string

	// 不可变文件检查列表
	ImmutableCheckFiles []string

	// 最近可执行文件阈值 (小时)
	RecentExecutableHours int

	// 大型可执行文件阈值 (MB)
	LargeExecutableMB int64
}

// CronConfig 定时任务配置
type CronConfig struct {
	// 系统 cron 路径
	SystemCronPaths []string
}

// LoginConfig 登录历史配置
type LoginConfig struct {
	// 最近登录记录数量
	RecentLoginCount int

	// 失败登录记录数量
	FailedLoginCount int

	// 高频 IP 阈值
	HighFrequencyIPThreshold int

	// 同一 IP 登录阈值
	SameIPLoginThreshold int

	// Root 不同 IP 阈值
	RootDifferentIPThreshold int
}

// ScoringConfig 风险评分配置
type ScoringConfig struct {
	// 各检查项的权重
	Weights map[string]CheckWeight

	// 威胁等级阈值
	CriticalThreshold int
	HighThreshold     int
	MediumThreshold   int

	// 最大建议数量
	MaxRecommendations int
}

// CheckWeight 检查项权重
type CheckWeight struct {
	Category  string
	FailScore int
	WarnScore int
}

// PerformanceConfig 性能配置
type PerformanceConfig struct {
	// 进程缓存时间
	ProcessCacheDuration time.Duration

	// 命令执行超时时间
	CommandTimeout time.Duration

	// authorized_keys 读取限制 (KB)
	AuthKeysReadLimitKB int64

	// 文件完整性检查批量大小
	IntegrityCheckBatchSize int
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		ProcessConfig: ProcessConfig{
			MinerKeywords: []string{
				"xmrig", "minergate", "cpuminer", "ccminer",
				"ethminer", "claymore", "phoenix", "t-rex",
				"minerd", "stratum", "nicehash", "cryptonight",
			},
			HighCPUThreshold: 70.0, // 降低到 70%
			HighCPUWhitelist: []string{
				// 数据库
				"postgres", "mysqld", "mariadb", "mongodb", "redis",
				// 编译工具
				"gcc", "g++", "clang", "rustc", "go", "javac",
				// 视频/图像处理
				"ffmpeg", "ffprobe", "convert", "imagemagick",
				// 科学计算
				"python", "R", "matlab", "octave", "julia",
				// 虚拟化
				"qemu", "kvm", "virtualbox", "vmware",
				// 压缩解压
				"tar", "gzip", "bzip2", "xz", "7z",
				// CI/CD
				"jenkins", "gitlab-runner", "docker", "containerd",
			},
			DeletedWhitelist: []string{
				"nginx", "docker", "containerd", "code-server",
				"mysqld", "node", "k3s", "kubelet",
			},
			RecentStartupHours: 24,
		},
		SSHConfig: SSHConfig{
			BinaryPaths: []string{
				"/usr/sbin/sshd",
				"/usr/bin/ssh",
			},
			ConfigPaths: []string{
				"/etc/ssh/sshd_config",
			},
			RecentModifyDays:         90,
			MaxAuthorizedKeysSize:    1024 * 1024, // 1MB
			AuthKeysRecentModifyDays: 30,
			MaxKeysCount:             10,
		},
		NetworkConfig: NetworkConfig{
			SuspiciousPorts: map[uint32]string{
				4444:  "Metasploit默认端口",
				1337:  "黑客常用端口",
				31337: "黑客常用端口",
			},
			MinerPorts: []uint32{
				3333, 4444, 5555, 7777, 8888, 9999,
				14444, 45560,
			},
			MinerPoolKeywords: []string{
				"pool", "stratum", "mine", "mining",
				"xmr", "monero", "eth", "btc",
			},
		},
		FileConfig: FileConfig{
			TempDirs: []string{
				"/tmp",
				"/dev/shm",
				"/var/tmp",
			},
			SuspiciousPaths: []string{
				"/tmp/",
				"/dev/shm/",
				"/var/tmp/",
				"memfd:",
				"(deleted)",
			},
			CriticalBinaries: []string{
				"/bin/bash", "/bin/sh", "/bin/ls", "/bin/ps", "/bin/netstat",
				"/usr/bin/ssh", "/usr/bin/sudo", "/usr/sbin/sshd", "/usr/sbin/lsof",
				"/usr/bin/passwd", "/usr/bin/chsh", "/usr/bin/curl", "/usr/bin/wget",
			},
			ImmutableCheckFiles: []string{
				"/etc/passwd", "/etc/shadow", "/etc/group",
				"/etc/sudoers", "/etc/hosts", "/etc/resolv.conf",
				"/etc/ld.so.preload", "/etc/crontab",
				"/bin/ls", "/bin/ps", "/bin/netstat",
				"/usr/sbin/sshd",
				"/root/.ssh/authorized_keys",
			},
			RecentExecutableHours: 2,
			LargeExecutableMB:     10,
		},
		CronConfig: CronConfig{
			SystemCronPaths: []string{
				"/etc/crontab",
				"/etc/cron.d/",
				"/etc/cron.hourly/",
				"/etc/cron.daily/",
				"/etc/cron.weekly/",
				"/etc/cron.monthly/",
				"/var/spool/cron/",
			},
		},
		LoginConfig: LoginConfig{
			RecentLoginCount:         50,
			FailedLoginCount:         100,
			HighFrequencyIPThreshold: 10,
			SameIPLoginThreshold:     30, // 降低到 30
			RootDifferentIPThreshold: 3,
		},
		ScoringConfig: ScoringConfig{
			Weights: map[string]CheckWeight{
				"rootkit_detection":    {FailScore: 50, WarnScore: 20},
				"suspicious_processes": {FailScore: 40, WarnScore: 15},
				"suspicious_env_vars":  {FailScore: 35, WarnScore: 15},
				"ssh_security":         {FailScore: 15, WarnScore: 5},
				"listening_ports":      {FailScore: 30, WarnScore: 10},
				"cron_jobs":            {FailScore: 25, WarnScore: 10},
				"suspicious_files":     {FailScore: 30, WarnScore: 10},
				"system_accounts":      {FailScore: 40, WarnScore: 15},
				"network_connections":  {FailScore: 35, WarnScore: 15},
				"file_integrity":       {FailScore: 30, WarnScore: 10},
				"immutable_files":      {FailScore: 45, WarnScore: 15},
				"login_history":        {FailScore: 20, WarnScore: 8},
			},
			CriticalThreshold:  80,
			HighThreshold:      50,
			MediumThreshold:    20,
			MaxRecommendations: 20,
		},
		PerformanceConfig: PerformanceConfig{
			ProcessCacheDuration:    1 * time.Second,
			CommandTimeout:          15 * time.Second,
			AuthKeysReadLimitKB:     512,
			IntegrityCheckBatchSize: 10,
		},
	}
}
