package audit

import (
	"fmt"
	"strings"
	"time"

	"github.com/dushixiang/pika/internal/protocol"
	"github.com/shirou/gopsutil/v4/process"
)

// ProcessChecker 进程检查器
type ProcessChecker struct {
	config   *Config
	cache    *ProcessCache
	evidence *EvidenceCollector
}

// NewProcessChecker 创建进程检查器
func NewProcessChecker(config *Config, cache *ProcessCache, evidence *EvidenceCollector) *ProcessChecker {
	return &ProcessChecker{
		config:   config,
		cache:    cache,
		evidence: evidence,
	}
}

// CheckSuspiciousProcesses 检查可疑进程
func (pc *ProcessChecker) CheckSuspiciousProcesses() protocol.SecurityCheck {
	check := protocol.SecurityCheck{
		Category: "suspicious_processes",
		Status:   StatusPass,
		Message:  "可疑进程检测",
		Details:  []protocol.SecurityCheckSub{},
	}

	procs, err := pc.cache.Get()
	if err != nil {
		check.Status = StatusSkip
		check.Message = "无法获取进程列表"
		globalLogger.Error("无法获取进程列表: %v", err)
		return check
	}

	suspiciousCount := 0

	for _, p := range procs {
		name, _ := p.Name()
		cmdline, _ := p.Cmdline()
		exe, _ := p.Exe()

		// 1. 检查挖矿
		if pc.isMinerProcess(name, cmdline) {
			suspiciousCount++
			check.Status = StatusFail

			cpuPercent, _ := p.CPUPercent()

			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:     "miner_process",
				Status:   StatusFail,
				Message:  fmt.Sprintf("检测到挖矿程序: %s (PID: %d, CPU: %.1f%%)", name, p.Pid, cpuPercent),
				Evidence: pc.evidence.CollectProcessEvidence(p, "high"),
			})
			continue
		}

		// 2. 检查高 CPU + 网络连接的可疑进程
		if pc.isHighCPUMinerSuspect(p, name, exe) {
			suspiciousCount++
			check.Status = StatusWarn

			cpuPercent, _ := p.CPUPercent()

			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:     "high_cpu_network_process",
				Status:   StatusWarn,
				Message:  fmt.Sprintf("高CPU进程持续网络连接: %s (PID: %d, CPU: %.1f%%) - 疑似挖矿", name, p.Pid, cpuPercent),
				Evidence: pc.evidence.CollectProcessEvidence(p, "medium"),
			})
			continue
		}

		// 3. 检查反弹 Shell
		if pc.isReverseShell(cmdline) {
			suspiciousCount++
			check.Status = StatusFail
			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:     "reverse_shell",
				Status:   StatusFail,
				Message:  fmt.Sprintf("检测到反弹shell: %s (PID: %d)", cmdline, p.Pid),
				Evidence: pc.evidence.CollectProcessEvidence(p, "high"),
			})
			continue
		}

		// 4. 检查无文件进程/被删除的二进制
		if strings.Contains(exe, "(deleted)") {
			// 检查白名单
			if pc.isInDeletedWhitelist(name) {
				continue
			}

			// 检查启动时间
			createTime, _ := p.CreateTime()
			uptime := time.Now().UnixMilli() - createTime
			if uptime < int64(pc.config.ProcessConfig.RecentStartupHours)*60*60*1000 {
				suspiciousCount++
				check.Status = StatusWarn
				check.Details = append(check.Details, protocol.SecurityCheckSub{
					Name:     "deleted_binary",
					Status:   StatusWarn,
					Message:  fmt.Sprintf("可疑的已删除进程: %s (PID: %d, 路径: %s) - 最近%dh内启动", name, p.Pid, exe, pc.config.ProcessConfig.RecentStartupHours),
					Evidence: pc.evidence.CollectProcessEvidence(p, "medium"),
				})
			}
		}

		// 5. 内存执行检测 (memfd)
		if strings.Contains(exe, "memfd:") {
			// 排除 runc / kubelet 等容器组件
			if !strings.Contains(cmdline, "runc") && !strings.Contains(name, "containerd") {
				suspiciousCount++
				check.Status = StatusFail
				check.Details = append(check.Details, protocol.SecurityCheckSub{
					Name:     "fileless_memfd",
					Status:   StatusFail,
					Message:  fmt.Sprintf("检测到无文件内存进程: %s (PID: %d)", name, p.Pid),
					Evidence: pc.evidence.CollectProcessEvidence(p, "high"),
				})
			}
		}
	}

	if suspiciousCount == 0 {
		check.Message = "进程行为正常"
	} else {
		check.Message = fmt.Sprintf("发现 %d 个可疑进程", suspiciousCount)
	}

	return check
}

// CheckSuspiciousEnvVars 检查进程的可疑环境变量
func (pc *ProcessChecker) CheckSuspiciousEnvVars() protocol.SecurityCheck {
	check := protocol.SecurityCheck{
		Category: "suspicious_env_vars",
		Status:   StatusPass,
		Message:  "进程环境变量检查",
		Details:  []protocol.SecurityCheckSub{},
	}

	procs, err := pc.cache.Get()
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
				if pc.isLegitimateLibraryPath(value, name, exe) {
					continue
				}

				// 评估风险等级
				riskLevel := pc.assessLibraryRisk(value)
				status := StatusWarn
				if riskLevel == "high" {
					status = StatusFail
					check.Status = StatusFail
				} else if check.Status != StatusFail {
					check.Status = StatusWarn
				}

				suspiciousCount++

				check.Details = append(check.Details, protocol.SecurityCheckSub{
					Name:     "suspicious_ld_env",
					Status:   status,
					Message:  fmt.Sprintf("进程 '%s'(PID: %d) 发现可疑环境变量: %s (风险等级: %s)", name, p.Pid, e, riskLevel),
					Evidence: pc.evidence.CollectProcessEvidence(p, riskLevel),
				})
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

// isMinerProcess 检查是否是挖矿进程
func (pc *ProcessChecker) isMinerProcess(name, cmdline string) bool {
	checkStr := strings.ToLower(name + " " + cmdline)

	// 检查关键词
	for _, keyword := range pc.config.ProcessConfig.MinerKeywords {
		if strings.Contains(checkStr, keyword) {
			return true
		}
	}

	// 检查挖矿命令特征
	if strings.Contains(cmdline, "--donate-level") ||
		strings.Contains(cmdline, "-o stratum+tcp") ||
		(strings.Contains(cmdline, "--pool") && strings.Contains(cmdline, "--user")) {
		return true
	}

	return false
}

// isHighCPUMinerSuspect 检查是否是高 CPU 挖矿嫌疑进程
func (pc *ProcessChecker) isHighCPUMinerSuspect(p *process.Process, name, exe string) bool {
	cpuPercent, err := p.CPUPercent()
	if err != nil || cpuPercent < pc.config.ProcessConfig.HighCPUThreshold {
		return false
	}

	// 检查白名单
	nameLower := strings.ToLower(name)
	exeLower := strings.ToLower(exe)

	for _, whitelisted := range pc.config.ProcessConfig.HighCPUWhitelist {
		if strings.Contains(nameLower, whitelisted) || strings.Contains(exeLower, whitelisted) {
			return false
		}
	}

	// 检查是否有网络连接
	connections, err := p.Connections()
	if err != nil || len(connections) == 0 {
		return false
	}

	// 检查是否有 ESTABLISHED 状态的外部连接
	for _, conn := range connections {
		if conn.Status == "ESTABLISHED" && !IsLocalIP(conn.Raddr.IP) {
			return true
		}
	}

	return false
}

// isReverseShell 检查是否是反弹 shell
func (pc *ProcessChecker) isReverseShell(cmdline string) bool {
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

// isInDeletedWhitelist 检查是否在 deleted 进程白名单中
func (pc *ProcessChecker) isInDeletedWhitelist(name string) bool {
	for _, safeName := range pc.config.ProcessConfig.DeletedWhitelist {
		if strings.Contains(strings.ToLower(name), safeName) {
			return true
		}
	}
	return false
}

// isLegitimateLibraryPath 检查是否是合法的库路径
func (pc *ProcessChecker) isLegitimateLibraryPath(path, processName, processExe string) bool {
	// 开发工具(IDE)
	devTools := []string{
		"jetbrains", "vscode", "idea", "pycharm", "goland", "webstorm",
		"clion", "rider", "datagrip", "androidstudio",
	}
	for _, tool := range devTools {
		if strings.Contains(strings.ToLower(processExe), tool) {
			return true
		}
	}

	// 数据库
	databases := []string{"postgres", "mysql", "mariadb", "mongodb", "redis", "elasticsearch", "clickhouse"}
	for _, db := range databases {
		if strings.Contains(strings.ToLower(processName), db) {
			return true
		}
	}

	// 容器相关
	containers := []string{"docker", "containerd", "kubelet", "podman", "cri-o", "k3s", "kubernetes"}
	for _, container := range containers {
		if strings.Contains(strings.ToLower(processName), container) {
			return true
		}
	}

	// Web 服务器
	webServers := []string{"nginx", "apache", "httpd", "caddy", "traefik"}
	for _, ws := range webServers {
		if strings.Contains(strings.ToLower(processName), ws) {
			return true
		}
	}

	// 科学计算和虚拟化
	scientificAndVirt := []string{
		"matlab", "mathematica", "octave", "julia", // 科学计算
		"virtualbox", "qemu", "vmware", "kvm", "libvirt", // 虚拟化
		"datadog", "newrelic", "prometheus", "grafana", // 监控工具
	}
	for _, tool := range scientificAndVirt {
		if strings.Contains(strings.ToLower(processName), tool) ||
			strings.Contains(strings.ToLower(processExe), tool) {
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
func (pc *ProcessChecker) assessLibraryRisk(path string) string {
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
