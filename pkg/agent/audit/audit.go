package audit

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/dushixiang/pika/internal/protocol"
	"github.com/shirou/gopsutil/v4/host"
	gopsutilNet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

const (
	StatusPass string = "pass"
	StatusFail string = "fail"
	StatusWarn string = "warn"
	StatusSkip string = "skip"
)

// RunAudit 执行 VPS 安全审计
func RunAudit() (*protocol.VPSAuditResult, error) {
	startTime := time.Now().UnixMilli()

	// 检查操作系统
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("只支持 Linux 系统")
	}

	// 检查运行权限 - root权限可以执行更完整的审计
	if os.Geteuid() != 0 {
		return nil, fmt.Errorf("需要root权限运行完整审计")
	}

	// 获取系统信息
	systemInfo, err := getSystemInfo()
	if err != nil {
		return nil, fmt.Errorf("获取系统信息失败: %w", err)
	}

	// 执行安全检查(从应急响应角度)
	checks := []SecurityCheckFunc{
		checkRootkit,             // rootkit检测
		checkSuspiciousProcesses, // 可疑进程
		checkSSHSecurity,         // SSH安全
		checkListeningPorts,      // 异常端口
		checkCronJobs,            // 定时任务
		checkSuspiciousFiles,     // 可疑文件
		checkSystemAccounts,      // 系统账户
		checkNetworkConnections,  // 网络连接
		checkFileIntegrity,       // 系统文件完整性
		checkLoginHistory,        // 登录历史
		checkImmutableFiles,      // 不可变文件(优化版)
		checkSuspiciousEnvVars,   // 进程环境变量
	}

	var securityChecks []protocol.SecurityCheck
	for _, checkFunc := range checks {
		check := checkFunc()
		securityChecks = append(securityChecks, check)
	}

	endTime := time.Now().UnixMilli()

	// 计算风险评分
	riskScore, threatLevel, recommendations := calculateRiskScore(securityChecks)

	return &protocol.VPSAuditResult{
		SystemInfo:      *systemInfo,
		SecurityChecks:  securityChecks,
		StartTime:       startTime,
		EndTime:         endTime,
		RiskScore:       riskScore,
		ThreatLevel:     threatLevel,
		Recommendations: recommendations,
	}, nil
}

type SecurityCheckFunc func() protocol.SecurityCheck

// getSystemInfo 获取系统信息
func getSystemInfo() (*protocol.SystemInfo, error) {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}

	info, err := host.Info()
	if err != nil {
		return nil, err
	}

	osInfo := fmt.Sprintf("%s %s", info.Platform, info.PlatformVersion)
	if info.PlatformFamily != "" {
		osInfo = fmt.Sprintf("%s (%s)", osInfo, info.PlatformFamily)
	}

	return &protocol.SystemInfo{
		Hostname:      hostname,
		OS:            osInfo,
		KernelVersion: info.KernelVersion,
		Uptime:        info.Uptime,
	}, nil
}

// checkRootkit 检测rootkit和恶意内核模块
func checkRootkit() protocol.SecurityCheck {
	check := protocol.SecurityCheck{
		Category: "rootkit_detection",
		Status:   StatusPass,
		Message:  "Rootkit检测",
		Details:  []protocol.SecurityCheckSub{},
	}

	// 检查加载的内核模块
	suspiciousModules := checkKernelModules()
	if len(suspiciousModules) > 0 {
		check.Status = StatusWarn
		for _, mod := range suspiciousModules {
			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:    "suspicious_module",
				Status:  StatusWarn,
				Message: fmt.Sprintf("可疑内核模块: %s", mod),
			})
		}
	} else {
		check.Details = append(check.Details, protocol.SecurityCheckSub{
			Name:    "kernel_modules",
			Status:  StatusPass,
			Message: "未发现可疑内核模块",
		})
	}

	// 检查隐藏进程
	hiddenProcs := checkHiddenProcesses()
	if len(hiddenProcs) > 0 {
		check.Status = StatusFail
		for _, pid := range hiddenProcs {
			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:    "hidden_process",
				Status:  StatusFail,
				Message: fmt.Sprintf("检测到隐藏进程: PID %d", pid),
			})
		}
	} else {
		check.Details = append(check.Details, protocol.SecurityCheckSub{
			Name:    "hidden_processes",
			Status:  StatusPass,
			Message: "未发现隐藏进程",
		})
	}

	if check.Status == StatusPass {
		check.Message = "未检测到rootkit"
	} else if check.Status == StatusWarn {
		check.Message = "发现可疑rootkit特征"
	} else {
		check.Message = "发现rootkit感染迹象"
	}

	return check
}

// checkSuspiciousEnvVars 检查进程的可疑环境变量
func checkSuspiciousEnvVars() protocol.SecurityCheck {
	check := protocol.SecurityCheck{
		Category: "suspicious_env_vars",
		Status:   StatusPass,
		Message:  "进程环境变量检查",
		Details:  []protocol.SecurityCheckSub{},
	}

	procs, err := process.Processes()
	if err != nil {
		check.Status = StatusSkip
		check.Message = "无法获取进程列表"
		return check
	}

	suspiciousCount := 0
	for _, p := range procs {
		env, err := p.Environ()
		if err != nil {
			continue
		}

		name, _ := p.Name()
		exe, _ := p.Exe()

		for _, e := range env {
			if strings.HasPrefix(e, "LD_PRELOAD=") || strings.HasPrefix(e, "LD_LIBRARY_PATH=") {
				parts := strings.SplitN(e, "=", 2)
				if len(parts) != 2 {
					continue
				}
				value := parts[1]

				// 检查是否是合法的库路径
				if isLegitimateLibraryPath(value, name, exe) {
					continue
				}

				// 评估风险等级
				riskLevel := assessLibraryRisk(value)
				status := StatusWarn
				if riskLevel == "high" {
					status = StatusFail
					check.Status = StatusFail
				} else if check.Status != StatusFail {
					check.Status = StatusWarn
				}

				suspiciousCount++

				// 收集证据
				evidence := collectProcessEvidence(p, riskLevel)

				check.Details = append(check.Details, protocol.SecurityCheckSub{
					Name:     "suspicious_ld_env",
					Status:   status,
					Message:  fmt.Sprintf("进程 '%s'(PID: %d) 发现可疑环境变量: %s (风险等级: %s)", name, p.Pid, e, riskLevel),
					Evidence: evidence,
				})
				// 找到一个就够了
				break
			}
		}
	}

	if suspiciousCount == 0 {
		check.Message = "未发现可疑的进程环境变量"
	} else {
		check.Message = fmt.Sprintf("发现 %d 个进程存在可疑环境变量", suspiciousCount)
	}

	return check
}

// checkSuspiciousProcesses 检查可疑进程
func checkSuspiciousProcesses() protocol.SecurityCheck {
	check := protocol.SecurityCheck{
		Category: "suspicious_processes",
		Status:   StatusPass,
		Message:  "可疑进程检测",
		Details:  []protocol.SecurityCheckSub{},
	}

	procs, err := process.Processes()
	if err != nil {
		check.Status = StatusSkip
		check.Message = "无法获取进程列表"
		return check
	}

	suspiciousCount := 0

	for _, p := range procs {
		name, _ := p.Name()
		cmdline, _ := p.Cmdline()
		exe, _ := p.Exe()

		// 检查挖矿程序
		if isMinerProcess(name, cmdline) {
			suspiciousCount++
			check.Status = StatusFail

			// 收集证据
			evidence := collectProcessEvidence(p, "high")

			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:     "miner_process",
				Status:   StatusFail,
				Message:  fmt.Sprintf("检测到挖矿程序: %s (PID: %d)", name, p.Pid),
				Evidence: evidence,
			})
		}

		// 检查反弹shell
		if isReverseShell(cmdline) {
			suspiciousCount++
			check.Status = StatusFail

			// 收集证据
			evidence := collectProcessEvidence(p, "high")

			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:     "reverse_shell",
				Status:   StatusFail,
				Message:  fmt.Sprintf("检测到反弹shell: %s (PID: %d)", cmdline, p.Pid),
				Evidence: evidence,
			})
		}

		// 检查无文件进程(memfd_create等)
		if strings.Contains(exe, "memfd:") || strings.Contains(exe, "(deleted)") {
			suspiciousCount++
			check.Status = StatusWarn

			// 收集证据
			evidence := collectProcessEvidence(p, "high")

			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:     "fileless_process",
				Status:   StatusWarn,
				Message:  fmt.Sprintf("检测到无文件进程: %s (PID: %d, 路径: %s)", name, p.Pid, exe),
				Evidence: evidence,
			})
		}

		// 检查进程名称和可执行文件不匹配(进程伪装)
		// 改进的伪装检测逻辑
		if exe != "" && name != "" && !strings.Contains(exe, "(deleted)") {
			baseName := filepath.Base(exe)

			// 更精确的匹配逻辑
			nameMatch := (baseName == name) ||
				strings.HasPrefix(name, baseName) ||
				strings.Contains(name, ":") // PostgreSQL等进程有冒号后缀

			if !nameMatch && !strings.Contains(exe, name) {
				// 扩展白名单
				whitelistPatterns := []string{
					"python", "java", "node", "ruby", "perl", "php",
					"bash", "sh", "zsh", "ksh", "fish",
					"systemd", "containerd", "docker", "podman",
					"postgres", "mysqld", "redis", // 数据库worker进程
					"nginx", "httpd", "apache2", // Web服务器worker
				}

				isWhitelisted := false
				nameLower := strings.ToLower(name)
				baseNameLower := strings.ToLower(baseName)

				for _, pattern := range whitelistPatterns {
					if strings.Contains(nameLower, pattern) ||
						strings.Contains(baseNameLower, pattern) {
						isWhitelisted = true
						break
					}
				}

				// 检查是否是符号链接导致的不匹配(常见于busybox等)
				if !isWhitelisted {
					// 只有当路径看起来可疑时才报警
					if isSuspiciousPath(exe) {
						suspiciousCount++
						check.Status = StatusWarn

						evidence := collectProcessEvidence(p, "medium")

						check.Details = append(check.Details, protocol.SecurityCheckSub{
							Name:     "process_masquerading",
							Status:   StatusWarn,
							Message:  fmt.Sprintf("可疑的进程名称不匹配: 名称=%s, 路径=%s (PID: %d)", name, exe, p.Pid),
							Evidence: evidence,
						})
					}
				}
			}
		}
	}

	if suspiciousCount == 0 {
		check.Details = append(check.Details, protocol.SecurityCheckSub{
			Name:    "clean",
			Status:  StatusPass,
			Message: "未发现可疑进程",
		})
		check.Message = "进程正常"
	} else {
		check.Message = fmt.Sprintf("发现 %d 个可疑进程", suspiciousCount)
	}

	return check
}

// checkSSHSecurity 检查SSH后门和安全配置(增强版)
func checkSSHSecurity() protocol.SecurityCheck {
	check := protocol.SecurityCheck{
		Category: "ssh_security",
		Status:   StatusPass,
		Message:  "SSH安全检查",
		Details:  []protocol.SecurityCheckSub{},
	}

	// 检查SSH服务是否运行
	if !isProcessRunning("sshd") {
		check.Message = "SSH服务未运行"
		check.Details = append(check.Details, protocol.SecurityCheckSub{
			Name:    "ssh_status",
			Status:  StatusPass,
			Message: "SSH未运行",
		})
		return check
	}

	// 1. 检查authorized_keys文件内容(新增)
	suspiciousKeys := checkAuthorizedKeysContent()
	if len(suspiciousKeys) > 0 {
		check.Status = StatusWarn
		for _, key := range suspiciousKeys {
			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:    "suspicious_authorized_key",
				Status:  StatusWarn,
				Message: key,
			})
		}
	}

	// 2. 检查SSH配置
	sshConfig := readSSHConfig()

	// 检查是否允许root登录
	permitRoot := sshConfig["permitrootlogin"] // 转小写统一处理
	if permitRoot == "yes" {
		check.Status = StatusWarn
		check.Details = append(check.Details, protocol.SecurityCheckSub{
			Name:    "permit_root_login",
			Status:  StatusWarn,
			Message: "允许root SSH登录",
		})
	} else {
		check.Details = append(check.Details, protocol.SecurityCheckSub{
			Name:    "permit_root_login",
			Status:  StatusPass,
			Message: fmt.Sprintf("Root登录策略: %s", permitRoot),
		})
	}

	// 检查空密码登录
	permitEmpty := sshConfig["permitemptypasswords"]
	if permitEmpty == "yes" {
		check.Status = StatusFail
		check.Details = append(check.Details, protocol.SecurityCheckSub{
			Name:    "permit_empty_passwords",
			Status:  StatusFail,
			Message: "允许空密码登录",
		})
	}

	// 检查密码认证是否启用(建议使用密钥)
	passwordAuth := sshConfig["passwordauthentication"]
	if passwordAuth == "yes" {
		check.Status = StatusWarn
		check.Details = append(check.Details, protocol.SecurityCheckSub{
			Name:    "password_auth_enabled",
			Status:  StatusWarn,
			Message: "启用了密码认证(建议仅使用密钥认证)",
		})
	}

	// 3. 检查SSH后门进程
	suspiciousSshd := checkSuspiciousSSHD()
	if len(suspiciousSshd) > 0 {
		check.Status = StatusFail
		for _, sshd := range suspiciousSshd {
			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:    "backdoor_sshd",
				Status:  StatusFail,
				Message: sshd,
			})
		}
	}

	// 4. 检查SSH二进制文件完整性(新增)
	sshdBinaries := []string{"/usr/sbin/sshd", "/usr/bin/ssh"}
	for _, binPath := range sshdBinaries {
		info, err := os.Stat(binPath)
		if err != nil {
			continue
		}

		// 检查最近修改
		modTime := info.ModTime()
		age := time.Since(modTime)
		if age < 30*24*time.Hour { // 最近30天
			fileHash := calculateSHA256(binPath)
			check.Status = StatusWarn
			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:    "ssh_binary_modified",
				Status:  StatusWarn,
				Message: fmt.Sprintf("SSH二进制文件最近被修改: %s (%d天前)", binPath, int(age.Hours()/24)),
				Evidence: &protocol.Evidence{
					FilePath:  binPath,
					FileHash:  fileHash,
					Timestamp: modTime.UnixMilli(),
					RiskLevel: "high",
				},
			})
		}
	}

	// 5. 检查SSH配置文件的软链接(新增,检测配置劫持)
	sshConfigPaths := []string{"/etc/ssh/sshd_config"}
	for _, configPath := range sshConfigPaths {
		linkTarget, err := os.Readlink(configPath)
		if err == nil {
			// 是软链接,检查目标是否可疑
			if strings.Contains(linkTarget, "/tmp") || strings.Contains(linkTarget, "/dev/shm") {
				check.Status = StatusFail
				check.Details = append(check.Details, protocol.SecurityCheckSub{
					Name:    "ssh_config_symlink",
					Status:  StatusFail,
					Message: fmt.Sprintf("SSH配置文件是指向可疑位置的软链接: %s -> %s", configPath, linkTarget),
				})
			}
		}
	}

	if check.Status == StatusPass {
		check.Message = "SSH配置安全"
	} else if check.Status == StatusWarn {
		check.Message = "SSH配置存在风险"
	} else {
		check.Message = "检测到SSH后门"
	}

	return check
}

// checkListeningPorts 检查异常监听端口(增强版:进程-端口合理性分析)
func checkListeningPorts() protocol.SecurityCheck {
	check := protocol.SecurityCheck{
		Category: "listening_ports",
		Status:   StatusPass,
		Message:  "端口监听检查",
		Details:  []protocol.SecurityCheckSub{},
	}

	// 使用gopsutil获取监听端口及进程信息
	connections, err := gopsutilNet.Connections("all")
	if err != nil {
		check.Status = StatusSkip
		check.Message = "无法获取网络连接信息"
		return check
	}

	// 反弹shell/恶意工具常用端口(降低优先级,减少误报)
	suspiciousPorts := map[uint32]string{
		4444:  "Metasploit默认端口",
		1337:  "黑客常用端口",
		31337: "黑客常用端口",
	}

	// 常见服务的合法端口范围
	legitimatePortRanges := map[string][]uint32{
		"nginx":      {80, 443, 8080, 8443},
		"apache":     {80, 443, 8080, 8443},
		"httpd":      {80, 443, 8080, 8443},
		"mysql":      {3306, 33060},
		"mariadb":    {3306, 33060},
		"postgres":   {5432, 5433},
		"redis":      {6379, 6380},
		"mongodb":    {27017, 27018, 27019},
		"ssh":        {22, 2222},
		"sshd":       {22, 2222},
		"docker":     {2375, 2376, 2377},
		"containerd": {2375, 2376},
	}

	suspiciousCount := 0
	totalListening := 0
	publicListening := 0

	for _, conn := range connections {
		if conn.Status != "LISTEN" {
			continue
		}

		totalListening++
		port := conn.Laddr.Port
		listenAddr := conn.Laddr.IP
		pid := conn.Pid

		// 本地回环监听的服务一般是安全的,跳过
		if listenAddr == "127.0.0.1" || listenAddr == "::1" {
			continue
		}

		publicListening++

		// 获取进程信息
		if pid > 0 {
			proc, err := process.NewProcess(int32(pid))
			if err != nil {
				continue
			}

			name, _ := proc.Name()
			exe, _ := proc.Exe()
			cmdline, _ := proc.Cmdline()

			// 检查1: 高危端口
			if desc, exists := suspiciousPorts[port]; exists {
				suspiciousCount++
				check.Status = StatusWarn

				evidence := collectProcessEvidence(proc, "high")

				check.Details = append(check.Details, protocol.SecurityCheckSub{
					Name:     fmt.Sprintf("suspicious_port_%d", port),
					Status:   StatusWarn,
					Message:  fmt.Sprintf("高危端口监听: %s:%d (%s) - 进程: %s (PID: %d)", listenAddr, port, desc, name, pid),
					Evidence: evidence,
				})
				continue
			}

			// 检查2: 挖矿程序监听端口
			if isMinerProcess(name, cmdline) {
				suspiciousCount++
				check.Status = StatusFail

				evidence := collectProcessEvidence(proc, "high")

				check.Details = append(check.Details, protocol.SecurityCheckSub{
					Name:     fmt.Sprintf("miner_listener_%d", port),
					Status:   StatusFail,
					Message:  fmt.Sprintf("挖矿程序监听端口: %s 监听 %s:%d (PID: %d)", name, listenAddr, port, pid),
					Evidence: evidence,
				})
				continue
			}

			// 检查3: 可疑路径的程序监听端口
			if isSuspiciousPath(exe) {
				suspiciousCount++
				check.Status = StatusFail

				evidence := collectProcessEvidence(proc, "high")

				check.Details = append(check.Details, protocol.SecurityCheckSub{
					Name:     fmt.Sprintf("suspicious_exe_listener_%d", port),
					Status:   StatusFail,
					Message:  fmt.Sprintf("可疑路径程序监听端口: %s 监听 %s:%d (PID: %d, 路径: %s)", name, listenAddr, port, pid, exe),
					Evidence: evidence,
				})
				continue
			}

			// 检查4: 进程与端口的合理性(新增)
			// 例如: MySQL进程监听80端口是异常的
			nameLower := strings.ToLower(name)
			isReasonable := true

			// 检查是否是已知服务
			if validPorts, exists := legitimatePortRanges[nameLower]; exists {
				isReasonable = false
				for _, validPort := range validPorts {
					if port == validPort {
						isReasonable = true
						break
					}
				}

				// 如果不合理,报告
				if !isReasonable {
					suspiciousCount++
					check.Status = StatusWarn

					check.Details = append(check.Details, protocol.SecurityCheckSub{
						Name:    fmt.Sprintf("port_mismatch_%d", port),
						Status:  StatusWarn,
						Message: fmt.Sprintf("进程端口不匹配: %s 监听异常端口 %s:%d (预期端口: %v)", name, listenAddr, port, validPorts),
					})
					continue
				}
			}

			// 检查5: 0.0.0.0监听高危端口(新增)
			// 某些服务应该只监听localhost
			if listenAddr == "0.0.0.0" || listenAddr == "::" {
				dangerousPublicPorts := []uint32{3306, 5432, 6379, 27017, 9200, 5601, 11211}
				for _, dangerousPort := range dangerousPublicPorts {
					if port == dangerousPort {
						suspiciousCount++
						check.Status = StatusWarn

						check.Details = append(check.Details, protocol.SecurityCheckSub{
							Name:    fmt.Sprintf("public_db_port_%d", port),
							Status:  StatusWarn,
							Message: fmt.Sprintf("数据库服务公网监听: %s 在 0.0.0.0:%d 监听 (建议仅监听127.0.0.1)", name, port),
						})
						break
					}
				}
			}
		}
	}

	if suspiciousCount == 0 {
		check.Details = append(check.Details, protocol.SecurityCheckSub{
			Name:    "safe_ports",
			Status:  StatusPass,
			Message: fmt.Sprintf("监听%d个端口(公网%d个),未发现可疑监听", totalListening, publicListening),
		})
		check.Message = "端口监听正常"
	} else {
		check.Message = fmt.Sprintf("发现 %d 个可疑端口监听", suspiciousCount)
	}

	return check
}

// isSuspiciousPath 检查可执行文件路径是否可疑
func isSuspiciousPath(path string) bool {
	if path == "" {
		return false
	}

	suspiciousPaths := []string{
		"/tmp/",
		"/dev/shm/",
		"/var/tmp/",
		"memfd:",
		"(deleted)",
	}

	for _, sp := range suspiciousPaths {
		if strings.Contains(path, sp) {
			return true
		}
	}

	return false
}

// checkCronJobs 检查定时任务(包括systemd timer)
func checkCronJobs() protocol.SecurityCheck {
	check := protocol.SecurityCheck{
		Category: "cron_jobs",
		Status:   StatusPass,
		Message:  "定时任务检查",
		Details:  []protocol.SecurityCheckSub{},
	}

	suspiciousCount := 0

	// 1. 检查传统crontab
	systemCrons := []string{
		"/etc/crontab",
		"/etc/cron.d/",
		"/etc/cron.hourly/",
		"/etc/cron.daily/",
		"/etc/cron.weekly/",
		"/etc/cron.monthly/",
		"/var/spool/cron/",
	}

	for _, cronPath := range systemCrons {
		suspicious := checkCronPath(cronPath)
		suspiciousCount += len(suspicious)
		for _, cron := range suspicious {
			check.Status = StatusWarn
			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:    "suspicious_cron",
				Status:  StatusWarn,
				Message: cron,
			})
		}
	}

	// 2. 检查systemd timers(现代Linux更常用)
	suspiciousTimers := checkSystemdTimers()
	if len(suspiciousTimers) > 0 {
		suspiciousCount += len(suspiciousTimers)
		check.Status = StatusWarn
		for _, timer := range suspiciousTimers {
			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:    "suspicious_systemd_timer",
				Status:  StatusWarn,
				Message: timer,
			})
		}
	}

	if suspiciousCount == 0 {
		check.Details = append(check.Details, protocol.SecurityCheckSub{
			Name:    "cron_clean",
			Status:  StatusPass,
			Message: "定时任务正常",
		})
		check.Message = "定时任务正常"
	} else {
		check.Message = fmt.Sprintf("发现 %d 个可疑定时任务", suspiciousCount)
	}

	return check
}

// checkSystemdTimers 检查systemd定时器
func checkSystemdTimers() []string {
	var suspicious []string
	// 获取所有timer单元的详细信息
	timerListOutput, err := execCommand("systemctl", "list-unit-files", "--type=timer", "--no-pager")
	if err != nil {
		return suspicious
	}

	// 解析timer列表
	lines := strings.Split(timerListOutput, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 1 || !strings.HasSuffix(fields[0], ".timer") {
			continue
		}

		timerName := fields[0]

		// 获取timer对应的service内容
		serviceName := strings.TrimSuffix(timerName, ".timer") + ".service"
		serviceContent, err := execCommand("systemctl", "cat", serviceName)
		if err != nil {
			continue
		}

		// 检查service内容是否可疑
		if isSuspiciousCron(serviceContent) {
			suspicious = append(suspicious, fmt.Sprintf("可疑systemd timer: %s -> %s", timerName, serviceName))
		}
	}

	return suspicious
}

// checkSuspiciousFiles 检查可疑文件(增强版)
func checkSuspiciousFiles() protocol.SecurityCheck {
	check := protocol.SecurityCheck{
		Category: "suspicious_files",
		Status:   StatusPass,
		Message:  "可疑文件检查",
		Details:  []protocol.SecurityCheckSub{},
	}

	suspiciousCount := 0

	// 1. 检查临时目录下的可执行文件
	tmpDirs := []string{"/tmp", "/dev/shm", "/var/tmp"}
	for _, dir := range tmpDirs {
		files := findSuspiciousExecutables(dir)
		if len(files) > 0 {
			check.Status = StatusWarn
			for _, file := range files {
				suspiciousCount++
				check.Details = append(check.Details, protocol.SecurityCheckSub{
					Name:    "tmp_executable",
					Status:  StatusWarn,
					Message: file,
				})
			}
		}
	}

	// 2. 检查隐藏的可执行文件(以.开头)
	hiddenExecs := findHiddenExecutables()
	if len(hiddenExecs) > 0 {
		check.Status = StatusWarn
		for _, file := range hiddenExecs {
			suspiciousCount++
			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:    "hidden_executable",
				Status:  StatusWarn,
				Message: fmt.Sprintf("隐藏可执行文件: %s", file),
			})
		}
	}

	// 3. 检查可疑的SUID/SGID文件
	suspiciousSUIDs := findSuspiciousSUIDFiles()
	if len(suspiciousSUIDs) > 0 {
		check.Status = StatusFail
		for _, file := range suspiciousSUIDs {
			suspiciousCount++
			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:    "suspicious_suid",
				Status:  StatusFail,
				Message: file,
			})
		}
	}

	if suspiciousCount == 0 {
		check.Details = append(check.Details, protocol.SecurityCheckSub{
			Name:    "files_clean",
			Status:  StatusPass,
			Message: "未发现可疑文件",
		})
		check.Message = "文件系统正常"
	} else {
		check.Message = fmt.Sprintf("发现 %d 个可疑文件", suspiciousCount)
	}

	return check
}

// findSuspiciousExecutables 查找可疑的可执行文件(带时间和大小分析)
func findSuspiciousExecutables(dir string) []string {
	suspicious := []string{}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return suspicious
	}

	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fullPath := filepath.Join(dir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// 检查是否可执行
		if info.Mode()&0111 != 0 {
			// 最近24小时创建的可执行文件更可疑
			age := now.Sub(info.ModTime())
			sizeKB := info.Size() / 1024

			if age < 24*time.Hour {
				suspicious = append(suspicious, fmt.Sprintf("%s 下的最近可执行文件: %s (大小: %dKB, 创建时间: %s)",
					dir, fullPath, sizeKB, info.ModTime().Format("2006-01-02 15:04:05")))
			} else if sizeKB > 1024 { // 大于1MB的可执行文件
				suspicious = append(suspicious, fmt.Sprintf("%s 下的大型可执行文件: %s (大小: %dKB)",
					dir, fullPath, sizeKB))
			} else {
				// 其他可执行文件只在特定条件下报告
				if strings.HasPrefix(entry.Name(), ".") {
					suspicious = append(suspicious, fmt.Sprintf("%s 下的隐藏可执行文件: %s", dir, fullPath))
				}
			}
		}
	}

	return suspicious
}

// findSuspiciousSUIDFiles 查找可疑的SUID/SGID文件
func findSuspiciousSUIDFiles() []string {
	suspicious := []string{}

	// 已知的合法SUID文件列表(常见系统工具)
	legitimateSUIDs := map[string]bool{
		"/usr/bin/sudo":                true,
		"/usr/bin/su":                  true,
		"/usr/bin/passwd":              true,
		"/usr/bin/chsh":                true,
		"/usr/bin/chfn":                true,
		"/usr/bin/gpasswd":             true,
		"/usr/bin/newgrp":              true,
		"/bin/su":                      true,
		"/bin/mount":                   true,
		"/bin/umount":                  true,
		"/bin/ping":                    true,
		"/usr/bin/pkexec":              true,
		"/usr/lib/openssh/ssh-keysign": true,
		"/usr/lib/dbus-1.0/dbus-daemon-launch-helper": true,
		"/usr/sbin/unix_chkpwd":                       true,
		"/sbin/unix_chkpwd":                           true,
	}

	// 使用find命令查找SUID/SGID文件(更高效)
	output, err := execCommand("find", "/usr", "/bin", "/sbin", "-type", "f", "(",
		"-perm", "-4000", "-o", "-perm", "-2000", ")", "-ls")
	if err != nil {
		return suspicious
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		// 解析find -ls输出,提取文件路径
		fields := strings.Fields(line)
		if len(fields) < 11 {
			continue
		}

		filePath := fields[len(fields)-1]

		// 跳过合法的SUID文件
		if legitimateSUIDs[filePath] {
			continue
		}

		// 检查文件是否在可疑路径
		if strings.HasPrefix(filePath, "/tmp/") || strings.HasPrefix(filePath, "/dev/shm/") ||
			strings.HasPrefix(filePath, "/var/tmp/") {
			suspicious = append(suspicious, fmt.Sprintf("可疑位置的SUID文件: %s (高危)", filePath))
			continue
		}

		// 报告所有非标准的SUID文件
		suspicious = append(suspicious, fmt.Sprintf("非标准SUID文件: %s", filePath))
	}

	// 限制报告数量
	if len(suspicious) > 15 {
		suspicious = suspicious[:15]
	}

	return suspicious
}

// checkSystemAccounts 检查系统账户异常
func checkSystemAccounts() protocol.SecurityCheck {
	check := protocol.SecurityCheck{
		Category: "system_accounts",
		Status:   StatusPass,
		Message:  "系统账户检查",
		Details:  []protocol.SecurityCheckSub{},
	}

	// 检查可登录的系统账户
	suspiciousAccounts := findSuspiciousAccounts()
	if len(suspiciousAccounts) > 0 {
		check.Status = StatusWarn
		for _, acc := range suspiciousAccounts {
			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:    "suspicious_account",
				Status:  StatusWarn,
				Message: acc,
			})
		}
	}

	// 检查无密码账户
	noPasswordAccounts := findNoPasswordAccounts()
	if len(noPasswordAccounts) > 0 {
		check.Status = StatusFail
		for _, acc := range noPasswordAccounts {
			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:    "no_password_account",
				Status:  StatusFail,
				Message: fmt.Sprintf("无密码账户: %s", acc),
			})
		}
	}

	// 检查UID为0的非root账户
	rootUIDAccounts := findRootUIDAccounts()
	if len(rootUIDAccounts) > 0 {
		check.Status = StatusFail
		for _, acc := range rootUIDAccounts {
			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:    "uid0_account",
				Status:  StatusFail,
				Message: fmt.Sprintf("UID为0的非root账户: %s", acc),
			})
		}
	}

	if check.Status == StatusPass {
		check.Details = append(check.Details, protocol.SecurityCheckSub{
			Name:    "accounts_clean",
			Status:  StatusPass,
			Message: "系统账户正常",
		})
		check.Message = "账户配置正常"
	} else if check.Status == StatusWarn {
		check.Message = "发现可疑账户"
	} else {
		check.Message = "发现严重账户安全问题"
	}

	return check
}

// checkNetworkConnections 检查异常网络连接(增强版:关联进程)
func checkNetworkConnections() protocol.SecurityCheck {
	check := protocol.SecurityCheck{
		Category: "network_connections",
		Status:   StatusPass,
		Message:  "网络连接检查",
		Details:  []protocol.SecurityCheckSub{},
	}

	connections, err := gopsutilNet.Connections("all")
	if err != nil {
		check.Status = StatusSkip
		check.Message = "无法获取网络连接"
		return check
	}

	suspiciousCount := 0
	establishedCount := 0

	// 检查可疑的外部连接
	for _, conn := range connections {
		if conn.Status != "ESTABLISHED" {
			continue
		}

		establishedCount++

		// 获取进程信息
		var processName string
		var processPath string
		if conn.Pid > 0 {
			proc, err := process.NewProcess(int32(conn.Pid))
			if err == nil {
				processName, _ = proc.Name()
				processPath, _ = proc.Exe()
			}
		}

		// 检查1: 连接到挖矿池
		if isMiningPool(conn.Raddr.IP, conn.Raddr.Port) {
			suspiciousCount++
			check.Status = StatusFail

			msg := fmt.Sprintf("连接到挖矿池: %s:%d", conn.Raddr.IP, conn.Raddr.Port)
			if processName != "" {
				msg += fmt.Sprintf(" (进程: %s, PID: %d)", processName, conn.Pid)
			}

			var evidence *protocol.Evidence
			if conn.Pid > 0 {
				proc, _ := process.NewProcess(int32(conn.Pid))
				if proc != nil {
					evidence = collectProcessEvidence(proc, "high")
				}
			}

			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:     "mining_connection",
				Status:   StatusFail,
				Message:  msg,
				Evidence: evidence,
			})
		}

		// 检查2: 系统进程的异常外连
		if processName != "" && isSystemProcess(processName) && !isLocalIP(conn.Raddr.IP) {
			// 系统进程不应该连接外部IP
			suspiciousCount++
			check.Status = StatusWarn

			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:    "system_process_external_conn",
				Status:  StatusWarn,
				Message: fmt.Sprintf("系统进程异常外连: %s (PID:%d) -> %s:%d", processName, conn.Pid, conn.Raddr.IP, conn.Raddr.Port),
			})
		}

		// 检查3: 可疑路径的进程外连
		if processPath != "" && isSuspiciousPath(processPath) {
			suspiciousCount++
			check.Status = StatusFail

			var evidence *protocol.Evidence
			if conn.Pid > 0 {
				proc, _ := process.NewProcess(int32(conn.Pid))
				if proc != nil {
					evidence = collectProcessEvidence(proc, "high")
				}
			}

			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:     "suspicious_path_connection",
				Status:   StatusFail,
				Message:  fmt.Sprintf("可疑路径进程外连: %s (PID:%d) -> %s:%d", processPath, conn.Pid, conn.Raddr.IP, conn.Raddr.Port),
				Evidence: evidence,
			})
		}
	}

	if suspiciousCount == 0 {
		check.Details = append(check.Details, protocol.SecurityCheckSub{
			Name:    "connections_clean",
			Status:  StatusPass,
			Message: fmt.Sprintf("共 %d 个活动连接,未发现异常", establishedCount),
		})
		check.Message = "网络连接正常"
	} else {
		check.Message = fmt.Sprintf("发现 %d 个可疑连接", suspiciousCount)
	}

	return check
}

// isSystemProcess 判断是否是系统进程
func isSystemProcess(name string) bool {
	systemProcs := []string{
		"systemd", "init", "sshd", "cron", "rsyslogd",
		"dbus-daemon", "systemd-udevd", "systemd-logind",
	}

	nameLower := strings.ToLower(name)
	for _, sp := range systemProcs {
		if nameLower == sp {
			return true
		}
	}
	return false
}

// isLocalIP 判断是否是本地IP
func isLocalIP(ip string) bool {
	if ip == "127.0.0.1" || ip == "::1" || ip == "localhost" {
		return true
	}
	// 检查私有IP段
	if strings.HasPrefix(ip, "192.168.") || strings.HasPrefix(ip, "10.") ||
		strings.HasPrefix(ip, "172.16.") || strings.HasPrefix(ip, "172.17.") ||
		strings.HasPrefix(ip, "172.18.") || strings.HasPrefix(ip, "172.19.") ||
		strings.HasPrefix(ip, "172.2") || strings.HasPrefix(ip, "172.30.") ||
		strings.HasPrefix(ip, "172.31.") {
		return true
	}
	return false
}

// checkFileIntegrity 检查关键系统文件完整性(增强版)
func checkFileIntegrity() protocol.SecurityCheck {
	check := protocol.SecurityCheck{
		Category: "file_integrity",
		Status:   StatusPass,
		Message:  "系统文件完整性检查",
		Details:  []protocol.SecurityCheckSub{},
	}

	suspiciousCount := 0

	// 1. 检查/etc/ld.so.preload (这个文件通常不应该存在)
	if content, err := os.ReadFile("/etc/ld.so.preload"); err == nil {
		contentStr := strings.TrimSpace(string(content))
		if contentStr != "" {
			suspiciousCount++
			check.Status = StatusWarn

			fileHash := calculateSHA256("/etc/ld.so.preload")
			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:    "ld_preload_exists",
				Status:  StatusWarn,
				Message: fmt.Sprintf("/etc/ld.so.preload 存在: %s", contentStr),
				Evidence: &protocol.Evidence{
					FilePath:  "/etc/ld.so.preload",
					FileHash:  fileHash,
					RiskLevel: "high",
				},
			})
		}
	}

	// 2. 检查/etc/ld.so.conf.d/下的可疑配置
	ldConfDir := "/etc/ld.so.conf.d"
	if entries, err := os.ReadDir(ldConfDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			fullPath := filepath.Join(ldConfDir, entry.Name())
			content, err := os.ReadFile(fullPath)
			if err != nil {
				continue
			}
			contentStr := string(content)
			if strings.Contains(contentStr, "/tmp") || strings.Contains(contentStr, "/dev/shm") {
				suspiciousCount++
				check.Status = StatusWarn

				fileHash := calculateSHA256(fullPath)
				check.Details = append(check.Details, protocol.SecurityCheckSub{
					Name:    "suspicious_ld_conf",
					Status:  StatusWarn,
					Message: fmt.Sprintf("可疑的库配置: %s 包含临时目录路径", fullPath),
					Evidence: &protocol.Evidence{
						FilePath:  fullPath,
						FileHash:  fileHash,
						RiskLevel: "medium",
					},
				})
			}
		}
	}

	// 3. 检查关键系统二进制文件的最近修改(新增)
	criticalBinaries := []string{
		"/bin/bash", "/bin/sh", "/bin/ls", "/bin/ps",
		"/usr/bin/ssh", "/usr/bin/sudo", "/usr/sbin/sshd",
		"/usr/bin/passwd", "/usr/bin/chsh",
	}

	now := time.Now()
	for _, binPath := range criticalBinaries {
		info, err := os.Stat(binPath)
		if err != nil {
			continue // 文件不存在,跳过
		}

		// 检查最近7天内被修改的关键二进制文件(可能被替换)
		modTime := info.ModTime()
		age := now.Sub(modTime)
		if age < 7*24*time.Hour {
			suspiciousCount++
			check.Status = StatusWarn

			fileHash := calculateSHA256(binPath)
			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:    "recently_modified_binary",
				Status:  StatusWarn,
				Message: fmt.Sprintf("关键系统文件最近被修改: %s (修改时间: %s, %d天前)", binPath, modTime.Format("2006-01-02 15:04:05"), int(age.Hours()/24)),
				Evidence: &protocol.Evidence{
					FilePath:  binPath,
					FileHash:  fileHash,
					Timestamp: modTime.UnixMilli(),
					RiskLevel: "medium",
				},
			})
		}
	}

	// 4. 检查启动脚本的最近修改(新增)
	startupScripts := []string{
		"/etc/rc.local",
		"/etc/profile",
		"/etc/bash.bashrc",
		"/etc/environment",
	}

	for _, scriptPath := range startupScripts {
		info, err := os.Stat(scriptPath)
		if err != nil {
			continue
		}

		modTime := info.ModTime()
		age := now.Sub(modTime)
		if age < 7*24*time.Hour {
			suspiciousCount++
			check.Status = StatusWarn

			fileHash := calculateSHA256(scriptPath)
			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:    "recently_modified_startup",
				Status:  StatusWarn,
				Message: fmt.Sprintf("启动脚本最近被修改: %s (修改时间: %s, %d天前)", scriptPath, modTime.Format("2006-01-02 15:04:05"), int(age.Hours()/24)),
				Evidence: &protocol.Evidence{
					FilePath:  scriptPath,
					FileHash:  fileHash,
					Timestamp: modTime.UnixMilli(),
					RiskLevel: "medium",
				},
			})
		}
	}

	// 5. 检查PAM配置(新增,用于检测SSH后门)
	pamSshdPath := "/etc/pam.d/sshd"
	if content, err := os.ReadFile(pamSshdPath); err == nil {
		contentStr := string(content)
		// 检查是否包含可疑的pam模块
		suspiciousPamModules := []string{"pam_backdoor", "pam_unix_auth"}
		for _, module := range suspiciousPamModules {
			if strings.Contains(contentStr, module) {
				suspiciousCount++
				check.Status = StatusFail

				fileHash := calculateSHA256(pamSshdPath)
				check.Details = append(check.Details, protocol.SecurityCheckSub{
					Name:    "suspicious_pam_module",
					Status:  StatusFail,
					Message: fmt.Sprintf("PAM配置包含可疑模块: %s in %s", module, pamSshdPath),
					Evidence: &protocol.Evidence{
						FilePath:  pamSshdPath,
						FileHash:  fileHash,
						RiskLevel: "high",
					},
				})
			}
		}
	}

	if suspiciousCount == 0 {
		check.Details = append(check.Details, protocol.SecurityCheckSub{
			Name:    "integrity_ok",
			Status:  StatusPass,
			Message: "系统文件完整性正常",
		})
		check.Message = "文件完整性正常"
	} else {
		check.Message = fmt.Sprintf("发现 %d 个文件异常", suspiciousCount)
	}

	return check
}

// checkImmutableFiles 检查不可变文件(优化版:只检查关键文件和批量执行)
func checkImmutableFiles() protocol.SecurityCheck {
	check := protocol.SecurityCheck{
		Category: "immutable_files",
		Status:   StatusPass,
		Message:  "不可变文件检查",
		Details:  []protocol.SecurityCheckSub{},
	}

	// 策略1: 只检查最关键的文件(通常被rootkit/后门保护)
	criticalFiles := []string{
		"/etc/ld.so.preload",
		"/etc/crontab",
		"/etc/rc.local",
		"/etc/passwd",
		"/etc/shadow",
		"/etc/sudoers",
		"/etc/hosts",
		"/etc/ssh/sshd_config",
		"/root/.ssh/authorized_keys",
		"/etc/systemd/system",
		"/lib/systemd/system",
	}

	suspiciousCount := 0

	// 先检查关键文件
	for _, file := range criticalFiles {
		if _, err := os.Stat(file); err != nil {
			continue // 文件不存在,跳过
		}

		// 执行lsattr检查
		lsattrCmd := exec.Command("lsattr", "-d", file)
		lsattrOutput, err := lsattrCmd.Output()
		if err != nil {
			continue
		}

		output := string(lsattrOutput)
		// 检查是否有i标志(immutable)
		if strings.Contains(output, "----i") || strings.Contains(output, "---i-") {
			suspiciousCount++
			check.Status = StatusWarn

			// 计算文件哈希
			fileHash := calculateSHA256(file)

			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:    "immutable_critical_file",
				Status:  StatusWarn,
				Message: fmt.Sprintf("关键文件被设置为不可变: %s", file),
				Evidence: &protocol.Evidence{
					FilePath:  file,
					FileHash:  fileHash,
					RiskLevel: "high", // 关键文件不可变是高危信号
				},
			})
		}
	}

	// 策略2: 批量检查关键目录(性能优化)
	criticalDirs := []string{"/etc", "/bin", "/sbin"}

	for _, dir := range criticalDirs {
		// 使用lsattr -R批量递归检查(比逐个文件快得多)
		lsattrCmd := exec.Command("lsattr", "-Ra", dir)
		lsattrOutput, err := lsattrCmd.Output()
		if err != nil {
			continue // 可能是权限问题或目录不存在
		}

		// 解析输出
		lines := strings.Split(string(lsattrOutput), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// lsattr输出格式: "----i-------e------ /path/to/file"
			if strings.Contains(line, "----i") || strings.Contains(line, "---i-") {
				// 提取文件路径
				parts := strings.Fields(line)
				if len(parts) < 2 {
					continue
				}
				filePath := parts[len(parts)-1]

				// 跳过已经检查过的关键文件(避免重复报告)
				alreadyReported := false
				for _, cf := range criticalFiles {
					if filePath == cf {
						alreadyReported = true
						break
					}
				}
				if alreadyReported {
					continue
				}

				suspiciousCount++
				check.Status = StatusWarn

				fileHash := calculateSHA256(filePath)

				check.Details = append(check.Details, protocol.SecurityCheckSub{
					Name:    "immutable_file",
					Status:  StatusWarn,
					Message: fmt.Sprintf("发现不可变文件: %s", filePath),
					Evidence: &protocol.Evidence{
						FilePath:  filePath,
						FileHash:  fileHash,
						RiskLevel: "medium",
					},
				})

				// 限制报告数量,避免过多
				if suspiciousCount >= 20 {
					break
				}
			}
		}

		if suspiciousCount >= 20 {
			break
		}
	}

	if suspiciousCount > 0 {
		if suspiciousCount >= 20 {
			check.Message = fmt.Sprintf("发现大量不可变文件(>=20个),可能存在rootkit")
		} else {
			check.Message = fmt.Sprintf("发现 %d 个不可变文件", suspiciousCount)
		}
	} else {
		check.Message = "未发现可疑的不可变文件"
		check.Details = append(check.Details, protocol.SecurityCheckSub{
			Name:    "immutable_files_clean",
			Status:  StatusPass,
			Message: "未发现可疑的不可变文件",
		})
	}

	return check
}

// checkCommandHistory 检查命令历史

// checkLoginHistory 检查登录历史(智能分析版)
func checkLoginHistory() protocol.SecurityCheck {
	check := protocol.SecurityCheck{
		Category: "login_history",
		Status:   StatusPass,
		Message:  "登录历史检查",
		Details:  []protocol.SecurityCheckSub{},
	}

	// 分析最近的成功登录
	lastOutput, err := execCommand("last", "-n", "50")
	if err == nil {
		suspiciousLogins := analyzeSuccessfulLogins(lastOutput)
		if len(suspiciousLogins) > 0 {
			check.Status = StatusWarn
			for _, login := range suspiciousLogins {
				check.Details = append(check.Details, protocol.SecurityCheckSub{
					Name:    "suspicious_login",
					Status:  StatusWarn,
					Message: login,
				})
			}
		}
	}

	// 分析失败的登录攻击
	lastbOutput, err := execCommand("lastb", "-n", "100")
	if err == nil {
		attackAnalysis := analyzeFailedLogins(lastbOutput)
		if attackAnalysis != nil {
			// 只有高频攻击或成功攻击才报告
			if attackAnalysis.HighFrequencyIPs > 0 {
				check.Status = StatusWarn
				check.Details = append(check.Details, protocol.SecurityCheckSub{
					Name:    "brute_force_attack",
					Status:  StatusWarn,
					Message: fmt.Sprintf("检测到暴力破解攻击: %d个IP高频尝试登录(>10次)", attackAnalysis.HighFrequencyIPs),
				})

				// 列出前5个最活跃的攻击IP
				for i, ipStat := range attackAnalysis.TopAttackers {
					if i >= 5 {
						break
					}
					check.Details = append(check.Details, protocol.SecurityCheckSub{
						Name:    fmt.Sprintf("top_attacker_%d", i+1),
						Status:  StatusWarn,
						Message: fmt.Sprintf("攻击来源 #%d: %s (失败%d次, 尝试用户: %s)", i+1, ipStat.IP, ipStat.Count, strings.Join(ipStat.Users, ", ")),
					})
				}
			}

			// 统计信息(仅作参考,不算异常)
			if attackAnalysis.TotalFailed > 0 {
				check.Details = append(check.Details, protocol.SecurityCheckSub{
					Name:    "failed_login_summary",
					Status:  StatusPass,
					Message: fmt.Sprintf("失败登录统计: 共%d次失败尝试, 来自%d个不同IP, 尝试了%d个不同用户名", attackAnalysis.TotalFailed, attackAnalysis.UniqueIPs, attackAnalysis.UniqueUsers),
				})
			}
		} else {
			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:    "failed_logins",
				Status:  StatusPass,
				Message: "暂无失败登录记录",
			})
		}
	} else {
		check.Details = append(check.Details, protocol.SecurityCheckSub{
			Name:    "failed_logins_check",
			Status:  StatusSkip,
			Message: "无法检查失败登录记录(可能需要root权限)",
		})
	}

	if check.Status == StatusPass {
		check.Message = "未发现可疑登录活动"
	} else {
		check.Message = "发现可疑登录活动"
	}

	return check
}

// analyzeSuccessfulLogins 分析成功登录,检测异常模式
func analyzeSuccessfulLogins(output string) []string {
	suspicious := []string{}
	lines := strings.Split(output, "\n")

	ipCount := make(map[string]int)
	var logins []loginRecord

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.Contains(trimmed, "wtmp begins") || strings.Contains(trimmed, "wtmp ends") {
			continue
		}

		// 解析登录记录: root pts/0 222.212.179.149 Mon Nov 10 13:45 still logged in
		fields := strings.Fields(trimmed)
		if len(fields) >= 3 {
			user := fields[0]
			ip := fields[2]

			// 过滤本地登录和系统重启记录
			if ip == "0.0.0.0" || ip == ":0" || ip == ":0.0" || user == "reboot" || user == "shutdown" {
				continue
			}

			logins = append(logins, loginRecord{user: user, ip: ip})
			ipCount[ip]++
		}
	}

	// 检测1: 短时间内来自同一IP的多次成功登录(可能是暴力破解成功后的批量操作)
	for ip, count := range ipCount {
		if count >= 5 {
			suspicious = append(suspicious, fmt.Sprintf("检测到IP %s 在最近50条记录中成功登录%d次(可能异常)", ip, count))
		}
	}

	// 检测2: root用户从多个不同IP登录(可能账户被多人使用)
	rootIPs := make(map[string]bool)
	for _, login := range logins {
		if login.user == "root" {
			rootIPs[login.ip] = true
		}
	}
	if len(rootIPs) >= 3 {
		ips := make([]string, 0, len(rootIPs))
		for ip := range rootIPs {
			ips = append(ips, ip)
		}
		suspicious = append(suspicious, fmt.Sprintf("检测到root用户从%d个不同IP登录: %s", len(rootIPs), strings.Join(ips, ", ")))
	}

	return suspicious
}

// loginRecord 登录记录
type loginRecord struct {
	user string
	ip   string
}

// AttackAnalysis 攻击分析结果
type AttackAnalysis struct {
	TotalFailed      int
	UniqueIPs        int
	UniqueUsers      int
	HighFrequencyIPs int
	TopAttackers     []IPAttackStat
}

// IPAttackStat IP攻击统计
type IPAttackStat struct {
	IP    string
	Count int
	Users []string
}

// analyzeFailedLogins 分析失败登录,识别暴力破解攻击
func analyzeFailedLogins(output string) *AttackAnalysis {
	lines := strings.Split(output, "\n")

	ipStats := make(map[string]*IPAttackStat)
	userSet := make(map[string]bool)
	totalFailed := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.Contains(trimmed, "btmp begins") || strings.Contains(trimmed, "btmp ends") {
			continue
		}

		// 解析失败登录: admin ssh:notty 167.71.230.163 Sat Nov 8 06:12 - 06:12 (00:00)
		fields := strings.Fields(trimmed)
		if len(fields) >= 3 {
			user := fields[0]
			ip := fields[2]

			totalFailed++
			userSet[user] = true

			if stat, exists := ipStats[ip]; exists {
				stat.Count++
				// 记录尝试的用户(去重)
				userExists := false
				for _, u := range stat.Users {
					if u == user {
						userExists = true
						break
					}
				}
				if !userExists && len(stat.Users) < 5 {
					stat.Users = append(stat.Users, user)
				}
			} else {
				ipStats[ip] = &IPAttackStat{
					IP:    ip,
					Count: 1,
					Users: []string{user},
				}
			}
		}
	}

	if totalFailed == 0 {
		return nil
	}

	// 统计高频攻击IP(失败次数>10次)
	highFreqCount := 0
	var topAttackers []IPAttackStat
	for _, stat := range ipStats {
		if stat.Count > 10 {
			highFreqCount++
			topAttackers = append(topAttackers, *stat)
		}
	}

	// 按攻击次数排序
	for i := 0; i < len(topAttackers); i++ {
		for j := i + 1; j < len(topAttackers); j++ {
			if topAttackers[j].Count > topAttackers[i].Count {
				topAttackers[i], topAttackers[j] = topAttackers[j], topAttackers[i]
			}
		}
	}

	return &AttackAnalysis{
		TotalFailed:      totalFailed,
		UniqueIPs:        len(ipStats),
		UniqueUsers:      len(userSet),
		HighFrequencyIPs: highFreqCount,
		TopAttackers:     topAttackers,
	}
}

// ========== 辅助检测函数 ==========

// checkKernelModules 检查可疑内核模块
func checkKernelModules() []string {
	suspicious := []string{}

	content, err := os.ReadFile("/proc/modules")
	if err != nil {
		return suspicious
	}

	// 已知的恶意模块名称特征
	maliciousPatterns := []string{
		"rootkit", "backdoor", "hidden", "stealth",
		"diamorphine", "suterusu", "reptile",
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			modName := fields[0]
			for _, pattern := range maliciousPatterns {
				if strings.Contains(strings.ToLower(modName), pattern) {
					suspicious = append(suspicious, modName)
					break
				}
			}
		}
	}

	return suspicious
}

// checkHiddenProcesses 检查隐藏进程
func checkHiddenProcesses() []int32 {
	hidden := []int32{}

	// 读取/proc目录
	procDirs, err := os.ReadDir("/proc")
	if err != nil {
		return hidden
	}

	procPIDs := make(map[int32]bool)
	for _, entry := range procDirs {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		procPIDs[int32(pid)] = true
	}

	// 使用gopsutil获取进程列表
	procs, err := process.Processes()
	if err != nil {
		return hidden
	}

	gopsutilPIDs := make(map[int32]bool)
	for _, p := range procs {
		gopsutilPIDs[p.Pid] = true
	}

	// 对比差异(简单检测)
	for pid := range procPIDs {
		if !gopsutilPIDs[pid] {
			hidden = append(hidden, pid)
		}
	}

	return hidden
}

// isMinerProcess 检查是否是挖矿进程
func isMinerProcess(name, cmdline string) bool {
	minerKeywords := []string{
		"xmrig", "minergate", "cpuminer", "ccminer",
		"ethminer", "claymore", "phoenix", "t-rex",
		"minerd", "stratum", "nicehash", "cryptonight",
	}

	checkStr := strings.ToLower(name + " " + cmdline)
	for _, keyword := range minerKeywords {
		if strings.Contains(checkStr, keyword) {
			return true
		}
	}

	// 检查挖矿命令特征
	if strings.Contains(cmdline, "--donate-level") ||
		strings.Contains(cmdline, "-o stratum+tcp") ||
		strings.Contains(cmdline, "--pool") && strings.Contains(cmdline, "--user") {
		return true
	}

	return false
}

// isReverseShell 检查是否是反弹shell
func isReverseShell(cmdline string) bool {
	patterns := []string{
		"bash -i >& /dev/tcp",
		"nc -e /bin/bash",
		"nc -e /bin/sh",
		"python -c 'import socket",
		"perl -e 'use Socket",
		"/bin/sh -i",
		"/bin/bash -i",
	}

	for _, pattern := range patterns {
		if strings.Contains(cmdline, pattern) {
			return true
		}
	}

	return false
}

// isProcessRunning 检查进程是否运行
func isProcessRunning(name string) bool {
	procs, err := process.Processes()
	if err != nil {
		return false
	}

	for _, p := range procs {
		pName, err := p.Name()
		if err == nil && strings.Contains(pName, name) {
			return true
		}
	}

	return false
}

// checkAuthorizedKeysContent 检查authorized_keys文件内容(增强版)
func checkAuthorizedKeysContent() []string {
	suspicious := []string{}
	users := getAllUsers()

	for _, user := range users {
		keyPath := filepath.Join(user.Home, ".ssh", "authorized_keys")

		// 检查文件是否存在
		info, err := os.Stat(keyPath)
		if err != nil {
			continue
		}

		// 1. 检查最近7天内被修改(从24小时改为7天,减少误报)
		if time.Since(info.ModTime()) < 7*24*time.Hour {
			suspicious = append(suspicious, fmt.Sprintf("用户 '%s' 的 authorized_keys 最近被修改: %s (%d天前)",
				user.Username, info.ModTime().Format("2006-01-02 15:04:05"), int(time.Since(info.ModTime()).Hours()/24)))
		}

		// 2. 检查文件内容
		content, err := os.ReadFile(keyPath)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")
		for lineNum, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			// 检查可疑的公钥选项
			if strings.Contains(line, "command=") {
				// 有command选项可能是强制命令,需要review
				suspicious = append(suspicious, fmt.Sprintf("用户 '%s' 的公钥包含command选项(行%d): %s...",
					user.Username, lineNum+1, truncateString(line, 60)))
			}

			if strings.Contains(line, "from=") {
				// 检查from字段是否包含*通配符
				if strings.Contains(line, "from=\"*\"") || strings.Contains(line, "from=*") {
					suspicious = append(suspicious, fmt.Sprintf("用户 '%s' 的公钥允许任意来源(from=*): 行%d", user.Username, lineNum+1))
				}
			}

			// 检查是否有多个公钥(可能被植入后门公钥)
			// 简单统计:超过5个公钥就提示
		}

		keyCount := 0
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") &&
				(strings.HasPrefix(line, "ssh-rsa") || strings.HasPrefix(line, "ssh-ed25519") || strings.HasPrefix(line, "ecdsa-")) {
				keyCount++
			}
		}

		if keyCount > 5 {
			suspicious = append(suspicious, fmt.Sprintf("用户 '%s' 拥有大量公钥(%d个), 建议审查", user.Username, keyCount))
		}

		// 3. 检查文件权限(authorized_keys应该是600或644)
		mode := info.Mode()
		if mode.Perm() != 0600 && mode.Perm() != 0644 {
			suspicious = append(suspicious, fmt.Sprintf("用户 '%s' 的 authorized_keys 权限异常: %o (应为600或644)", user.Username, mode.Perm()))
		}
	}

	return suspicious
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// readSSHConfig 读取SSH配置(改进版:统一转小写)
func readSSHConfig() map[string]string {
	config := make(map[string]string)

	// 优先使用sshd -T
	output, err := execCommand("sshd", "-T")
	if err == nil {
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				// 键名统一转小写
				config[strings.ToLower(parts[0])] = parts[1]
			}
		}
		return config
	}

	// 备用:读配置文件
	configPaths := []string{"/etc/ssh/sshd_config", "/etc/sshd_config"}
	for _, path := range configPaths {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			parts := strings.Fields(line)
			if len(parts) >= 2 {
				// 键名统一转小写
				config[strings.ToLower(parts[0])] = parts[1]
			}
		}
		break
	}

	return config
}

// checkSuspiciousSSHD 检查可疑的sshd进程
func checkSuspiciousSSHD() []string {
	suspicious := []string{}

	procs, err := process.Processes()
	if err != nil {
		return suspicious
	}

	sshdCount := 0
	for _, p := range procs {
		name, _ := p.Name()
		if strings.Contains(name, "sshd") {
			sshdCount++
			exe, _ := p.Exe()

			// 检查sshd路径是否正常
			if exe != "" && !strings.HasPrefix(exe, "/usr/sbin/sshd") {
				suspicious = append(suspicious, fmt.Sprintf("异常sshd路径: %s (PID: %d)", exe, p.Pid))
			}
		}
	}

	// 如果有多个sshd守护进程(不是子进程),可能异常
	if sshdCount > 10 {
		suspicious = append(suspicious, fmt.Sprintf("sshd进程数异常: %d", sshdCount))
	}

	return suspicious
}

// checkCronPath 检查cron目录
func checkCronPath(cronPath string) []string {
	suspicious := []string{}

	info, err := os.Stat(cronPath)
	if err != nil {
		return suspicious
	}

	if info.IsDir() {
		entries, err := os.ReadDir(cronPath)
		if err != nil {
			return suspicious
		}

		for _, entry := range entries {
			fullPath := filepath.Join(cronPath, entry.Name())
			content, err := os.ReadFile(fullPath)
			if err != nil {
				continue
			}

			// 检查可疑命令
			contentStr := string(content)
			if isSuspiciousCron(contentStr) {
				suspicious = append(suspicious, fmt.Sprintf("%s: %s", fullPath, contentStr[:100]))
			}
		}
	} else {
		content, err := os.ReadFile(cronPath)
		if err != nil {
			return suspicious
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			if isSuspiciousCron(line) {
				suspicious = append(suspicious, fmt.Sprintf("%s: %s", cronPath, line))
			}
		}
	}

	return suspicious
}

// isSuspiciousCron 判断是否是可疑的cron任务(改进版:减少误报)
func isSuspiciousCron(content string) bool {
	lowerContent := strings.ToLower(content)

	// 高危模式:下载后立即执行
	highRiskPatterns := []string{
		"curl.*|.*sh",         // curl xxx | sh
		"wget.*|.*sh",         // wget xxx | sh
		"curl.*|.*bash",       // curl xxx | bash
		"wget.*|.*bash",       // wget xxx | bash
		"curl.*&&.*chmod.*&&", // 下载->赋权->执行
		"wget.*&&.*chmod.*&&", // 下载->赋权->执行
		"base64 -d.*|.*sh",    // Base64解码后执行
		"base64 -d.*|.*bash",  // Base64解码后执行
		"bash -i",             // 交互式shell
		"sh -i",               // 交互式shell
		"/dev/tcp/",           // Bash反弹shell
		"nc.*-e",              // Netcat反弹shell
		"ncat.*-e",            // Ncat反弹shell
		"socat.*exec",         // Socat执行命令
		"python.*socket",      // Python反弹shell
		"perl.*socket",        // Perl反弹shell
	}

	for _, pattern := range highRiskPatterns {
		if strings.Contains(lowerContent, pattern) {
			return true
		}
	}

	// 中危模式:可疑路径+执行
	if (strings.Contains(lowerContent, "/tmp/") || strings.Contains(lowerContent, "/dev/shm/")) &&
		(strings.Contains(lowerContent, "chmod +x") || strings.Contains(lowerContent, "bash") || strings.Contains(lowerContent, "sh ")) {
		return true
	}

	// 单独的curl/wget不算可疑(合法备份脚本常用)
	return false
}

// findExecutablesInDir 查找目录下的可执行文件
func findExecutablesInDir(dir string) []string {
	executables := []string{}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return executables
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fullPath := filepath.Join(dir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// 检查是否可执行
		if info.Mode()&0111 != 0 {
			executables = append(executables, fullPath)
		}
	}

	return executables
}

// findHiddenExecutables 查找隐藏的可执行文件
func findHiddenExecutables() []string {
	hidden := []string{}

	searchDirs := []string{"/tmp", "/dev/shm", "/var/tmp"}

	for _, dir := range searchDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			name := entry.Name()
			if !strings.HasPrefix(name, ".") {
				continue
			}

			if entry.IsDir() {
				continue
			}

			fullPath := filepath.Join(dir, name)
			info, err := entry.Info()
			if err != nil {
				continue
			}

			if info.Mode()&0111 != 0 {
				hidden = append(hidden, fullPath)
			}
		}
	}

	return hidden
}

// UserInfo 用户信息
type UserInfo struct {
	Username string
	UID      string
	Home     string
	Shell    string
}

// getAllUsers 获取所有用户
func getAllUsers() []UserInfo {
	users := []UserInfo{}

	file, err := os.Open("/etc/passwd")
	if err != nil {
		return users
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) >= 7 {
			users = append(users, UserInfo{
				Username: parts[0],
				UID:      parts[2],
				Home:     parts[5],
				Shell:    parts[6],
			})
		}
	}

	return users
}

// findSuspiciousAccounts 查找可疑账户
func findSuspiciousAccounts() []string {
	suspicious := []string{}

	users := getAllUsers()
	for _, user := range users {
		uid, _ := strconv.Atoi(user.UID)

		// 系统账户(UID<1000)但有可登录shell
		if uid < 1000 && user.Username != "root" {
			// 合法的系统shell列表
			legitimateSystemShells := []string{
				"/sbin/nologin",
				"/bin/false",
				"/usr/sbin/nologin",
				"/bin/sync",      // sync账户使用
				"/sbin/shutdown", // shutdown账户使用
				"/sbin/halt",     // halt账户使用
			}

			isLegitimate := false
			for _, legitShell := range legitimateSystemShells {
				if user.Shell == legitShell {
					isLegitimate = true
					break
				}
			}

			// 只有非合法shell才报告为可疑
			if !isLegitimate {
				suspicious = append(suspicious, fmt.Sprintf("系统账户有可登录shell: %s (UID: %s, Shell: %s)", user.Username, user.UID, user.Shell))
			}
		}
	}

	return suspicious
}

// findNoPasswordAccounts 查找无密码账户
func findNoPasswordAccounts() []string {
	noPassword := []string{}

	file, err := os.Open("/etc/shadow")
	if err != nil {
		return noPassword
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) >= 2 {
			username := parts[0]
			password := parts[1]

			// 只有密码字段为空才是真正的无密码账户
			// "!" 和 "!!" 表示账户被锁定,这是安全的
			// "*" 表示无法使用密码登录(通常用于系统账户)
			if password == "" {
				noPassword = append(noPassword, username)
			}
		}
	}

	return noPassword
}

// findRootUIDAccounts 查找UID为0的非root账户
func findRootUIDAccounts() []string {
	rootUID := []string{}

	users := getAllUsers()
	for _, user := range users {
		if user.UID == "0" && user.Username != "root" {
			rootUID = append(rootUID, user.Username)
		}
	}

	return rootUID
}

// isMiningPool 检查是否连接到挖矿池
func isMiningPool(ip string, port uint32) bool {
	// 常见挖矿池端口
	minerPorts := []uint32{3333, 4444, 5555, 7777, 8888, 14444, 45560}
	for _, p := range minerPorts {
		if port == p {
			return true
		}
	}

	return false
}

// isTorConnection 检查是否是TOR连接
func isTorConnection(port uint32) bool {
	torPorts := []uint32{9001, 9030, 9050, 9051, 9150}
	for _, p := range torPorts {
		if port == p {
			return true
		}
	}
	return false
}

// execCommand 执行命令
func execCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return stdout.String(), nil
}

// ========== 新增的辅助函数 ==========

// calculateRiskScore 计算综合风险评分
func calculateRiskScore(checks []protocol.SecurityCheck) (int, string, []string) {
	score := 0
	var recommendations []string

	for _, check := range checks {
		for _, detail := range check.Details {
			switch detail.Status {
			case StatusFail:
				score += 20
				recommendations = append(recommendations,
					fmt.Sprintf("【紧急】%s: %s", check.Category, detail.Message))
			case StatusWarn:
				score += 5
				recommendations = append(recommendations,
					fmt.Sprintf("【警告】%s: %s", check.Category, detail.Message))
			}
		}
	}

	// 限制最大分数为100
	if score > 100 {
		score = 100
	}

	// 确定威胁等级
	var level string
	switch {
	case score >= 80:
		level = "critical"
	case score >= 50:
		level = "high"
	case score >= 20:
		level = "medium"
	default:
		level = "low"
	}

	// 限制建议数量,优先显示紧急的
	if len(recommendations) > 20 {
		recommendations = recommendations[:20]
	}

	return score, level, recommendations
}

// calculateSHA256 计算文件SHA256哈希
func calculateSHA256(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return ""
	}

	return hex.EncodeToString(hash.Sum(nil))
}

// buildProcessTree 构建进程树
func buildProcessTree(p *process.Process) []string {
	var tree []string
	current := p

	// 向上追溯父进程(最多5层)
	for i := 0; i < 5; i++ {
		if current == nil {
			break
		}

		name, _ := current.Name()
		exe, _ := current.Exe()
		cmdline, _ := current.Cmdline()

		info := fmt.Sprintf("PID:%d Name:%s", current.Pid, name)
		if exe != "" && exe != name {
			info += fmt.Sprintf(" Exe:%s", exe)
		}
		if cmdline != "" && len(cmdline) < 100 {
			info += fmt.Sprintf(" Cmd:%s", cmdline)
		}

		tree = append([]string{info}, tree...) // 前置插入

		ppid, err := current.Ppid()
		if err != nil || ppid == 0 {
			break
		}

		parent, err := process.NewProcess(ppid)
		if err != nil {
			break
		}
		current = parent
	}

	return tree
}

// collectProcessEvidence 收集进程证据
func collectProcessEvidence(p *process.Process, riskLevel string) *protocol.Evidence {
	exe, _ := p.Exe()
	createTime, _ := p.CreateTime()

	var fileHash string
	if exe != "" && !strings.Contains(exe, "deleted") && !strings.Contains(exe, "memfd:") {
		fileHash = calculateSHA256(exe)
	}

	processTree := buildProcessTree(p)

	return &protocol.Evidence{
		FileHash:    fileHash,
		ProcessTree: processTree,
		FilePath:    exe,
		Timestamp:   createTime,
		RiskLevel:   riskLevel,
	}
}

// isLegitimateLibraryPath 检查是否是合法的库路径
func isLegitimateLibraryPath(path, processName, processExe string) bool {
	// 开发工具
	devTools := []string{"jetbrains", "vscode", "idea", "pycharm", "goland"}
	for _, tool := range devTools {
		if strings.Contains(strings.ToLower(processExe), tool) {
			return true
		}
	}

	// 数据库
	databases := []string{"postgres", "mysql", "mariadb", "mongodb", "redis"}
	for _, db := range databases {
		if strings.Contains(strings.ToLower(processName), db) {
			return true
		}
	}

	// 容器相关
	containers := []string{"docker", "containerd", "kubelet", "podman", "cri-o"}
	for _, container := range containers {
		if strings.Contains(strings.ToLower(processName), container) {
			return true
		}
	}

	// Web服务器
	webServers := []string{"nginx", "apache", "httpd", "caddy"}
	for _, ws := range webServers {
		if strings.Contains(strings.ToLower(processName), ws) {
			return true
		}
	}

	// 系统库路径
	systemLibPaths := []string{
		"/usr/lib", "/lib", "/usr/local/lib",
		"/opt/", "/usr/share",
	}
	for _, sysPath := range systemLibPaths {
		if strings.HasPrefix(path, sysPath) {
			return true
		}
	}

	return false
}

// assessLibraryRisk 评估库路径的风险等级
func assessLibraryRisk(path string) string {
	// 高危路径
	highRiskPaths := []string{"/tmp/", "/dev/shm/", "/var/tmp/"}
	for _, hrp := range highRiskPaths {
		if strings.Contains(path, hrp) {
			return "high"
		}
	}

	// 隐藏文件
	if strings.Contains(path, "/.") {
		return "medium"
	}

	return "low"
}

// execCommandSafe 安全地执行命令(不使用shell)
func execCommandSafe(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	// 不捕获stderr,避免污染输出

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return stdout.String(), nil
}
