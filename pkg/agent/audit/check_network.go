package audit

import (
	"fmt"
	"strings"

	"github.com/dushixiang/pika/internal/protocol"
	gopsutilNet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

// NetworkChecker 网络检查器
type NetworkChecker struct {
	config   *Config
	cache    *ProcessCache
	evidence *EvidenceCollector
	strUtil  *StringUtils
}

// NewNetworkChecker 创建网络检查器
func NewNetworkChecker(config *Config, cache *ProcessCache, evidence *EvidenceCollector) *NetworkChecker {
	return &NetworkChecker{
		config:   config,
		cache:    cache,
		evidence: evidence,
		strUtil:  &StringUtils{},
	}
}

// CheckListeningPorts 检查异常监听端口
func (nc *NetworkChecker) CheckListeningPorts() protocol.SecurityCheck {
	check := protocol.SecurityCheck{
		Category: "listening_ports",
		Status:   StatusPass,
		Message:  "端口监听检查",
		Details:  []protocol.SecurityCheckSub{},
	}

	connections, err := gopsutilNet.Connections("all")
	if err != nil {
		check.Status = StatusSkip
		check.Message = "无法获取网络连接信息"
		globalLogger.Error("无法获取网络连接信息: %v", err)
		return check
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

		// 本地回环监听的服务一般是安全的，跳过
		if listenAddr == "127.0.0.1" || listenAddr == "::1" {
			continue
		}

		publicListening++

		// 获取进程信息
		if pid <= 0 {
			continue
		}

		proc, err := process.NewProcess(int32(pid))
		if err != nil {
			continue
		}

		name, _ := proc.Name()
		exe, _ := proc.Exe()
		cmdline, _ := proc.Cmdline()

		// 检查 1: 高危端口
		if desc, exists := nc.config.NetworkConfig.SuspiciousPorts[port]; exists {
			suspiciousCount++
			check.Status = StatusWarn

			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:     fmt.Sprintf("suspicious_port_%d", port),
				Status:   StatusWarn,
				Message:  fmt.Sprintf("高危端口监听: %s:%d (%s) - 进程: %s (PID: %d)", listenAddr, port, desc, name, pid),
				Evidence: nc.evidence.CollectProcessEvidence(proc, "high"),
			})
			continue
		}

		// 检查 2: 挖矿程序监听端口
		if nc.isMinerProcess(name, cmdline) {
			suspiciousCount++
			check.Status = StatusFail

			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:     fmt.Sprintf("miner_listener_%d", port),
				Status:   StatusFail,
				Message:  fmt.Sprintf("挖矿程序监听端口: %s 监听 %s:%d (PID: %d)", name, listenAddr, port, pid),
				Evidence: nc.evidence.CollectProcessEvidence(proc, "high"),
			})
			continue
		}

		// 检查 3: 可疑路径的程序监听端口
		if nc.isSuspiciousPath(exe) {
			suspiciousCount++
			check.Status = StatusFail

			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:     fmt.Sprintf("suspicious_exe_listener_%d", port),
				Status:   StatusFail,
				Message:  fmt.Sprintf("可疑路径程序监听端口: %s 监听 %s:%d (PID: %d, 路径: %s)", name, listenAddr, port, pid, exe),
				Evidence: nc.evidence.CollectProcessEvidence(proc, "high"),
			})
			continue
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

// CheckNetworkConnections 检查异常网络连接
func (nc *NetworkChecker) CheckNetworkConnections() protocol.SecurityCheck {
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
		globalLogger.Error("无法获取网络连接: %v", err)
		return check
	}

	suspiciousCount := 0
	establishedCount := 0

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

		// 检查 1: 连接到挖矿池
		if nc.isMiningPool(conn.Raddr.IP, conn.Raddr.Port) {
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
					evidence = nc.evidence.CollectProcessEvidence(proc, "high")
				}
			}

			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:     "mining_connection",
				Status:   StatusFail,
				Message:  msg,
				Evidence: evidence,
			})
		}

		// 检查 2: 可疑路径的进程外连
		if processPath != "" && nc.isSuspiciousPath(processPath) {
			suspiciousCount++
			check.Status = StatusFail

			var evidence *protocol.Evidence
			if conn.Pid > 0 {
				proc, _ := process.NewProcess(int32(conn.Pid))
				if proc != nil {
					evidence = nc.evidence.CollectProcessEvidence(proc, "high")
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

// isSuspiciousPath 检查可执行文件路径是否可疑
func (nc *NetworkChecker) isSuspiciousPath(path string) bool {
	if path == "" {
		return false
	}

	for _, sp := range nc.config.FileConfig.SuspiciousPaths {
		if strings.Contains(path, sp) {
			return true
		}
	}

	return false
}

// isMinerProcess 检查是否是挖矿进程
func (nc *NetworkChecker) isMinerProcess(name, cmdline string) bool {
	return nc.strUtil.ContainsAny(name+" "+cmdline, nc.config.ProcessConfig.MinerKeywords)
}

// isMiningPool 检查是否连接到挖矿池
func (nc *NetworkChecker) isMiningPool(ip string, port uint32) bool {
	// 检查端口
	for _, p := range nc.config.NetworkConfig.MinerPorts {
		if port == p {
			return true
		}
	}

	// TODO: 可以添加域名检查（需要反向 DNS）
	return false
}
