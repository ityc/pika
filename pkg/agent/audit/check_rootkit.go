package audit

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dushixiang/pika/internal/protocol"
)

// RootkitChecker rootkit 检测器
type RootkitChecker struct {
	config   *Config
	cache    *ProcessCache
	evidence *EvidenceCollector
}

// NewRootkitChecker 创建 rootkit 检测器
func NewRootkitChecker(config *Config, cache *ProcessCache, evidence *EvidenceCollector) *RootkitChecker {
	return &RootkitChecker{
		config:   config,
		cache:    cache,
		evidence: evidence,
	}
}

// Check 检测 rootkit 和恶意内核模块
func (rc *RootkitChecker) Check() protocol.SecurityCheck {
	check := protocol.SecurityCheck{
		Category: "rootkit_detection",
		Status:   StatusPass,
		Message:  "Rootkit检测",
		Details:  []protocol.SecurityCheckSub{},
	}

	// 检查加载的内核模块
	suspiciousModules := rc.checkKernelModules()
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
	hiddenProcs := rc.checkHiddenProcesses()
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

	// 更新总体消息
	switch check.Status {
	case StatusPass:
		check.Message = "未检测到rootkit"
	case StatusWarn:
		check.Message = "发现可疑rootkit特征"
	case StatusFail:
		check.Message = "发现rootkit感染迹象"
	}

	return check
}

// checkKernelModules 检查可疑内核模块
func (rc *RootkitChecker) checkKernelModules() []string {
	var suspicious []string

	content, err := os.ReadFile("/proc/modules")
	if err != nil {
		globalLogger.Warn("无法读取 /proc/modules: %v", err)
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
					globalLogger.Warn("发现可疑内核模块: %s", modName)
					break
				}
			}
		}
	}

	return suspicious
}

// checkHiddenProcesses 检查隐藏进程（优化版：二次确认机制）
func (rc *RootkitChecker) checkHiddenProcesses() []int32 {
	// 第一轮快照
	hiddenCandidates := rc.getHiddenCandidates()

	if len(hiddenCandidates) == 0 {
		return []int32{}
	}

	globalLogger.Debug("发现 %d 个隐藏进程候选，开始二次确认", len(hiddenCandidates))

	// 等待 100ms，消除进程刚好退出的时间差
	time.Sleep(100 * time.Millisecond)

	// 第二轮确认
	finalHidden := []int32{}

	// 重新获取当前的 gopsutil 列表
	procs, err := rc.cache.Get()
	if err != nil {
		globalLogger.Warn("二次确认时无法获取进程列表: %v", err)
		return []int32{}
	}

	currentApiPids := make(map[int32]bool)
	for _, p := range procs {
		currentApiPids[p.Pid] = true
	}

	for _, pid := range hiddenCandidates {
		// 检查 /proc/pid 是否依然存在
		if _, err := os.Stat(fmt.Sprintf("/proc/%d", pid)); err == nil {
			// /proc 下存在，但 API 列表里依然没有
			if !currentApiPids[pid] {
				finalHidden = append(finalHidden, pid)
				globalLogger.Warn("确认隐藏进程: PID %d", pid)
			}
		}
	}

	return finalHidden
}

// getHiddenCandidates 获取初步的隐藏进程候选
func (rc *RootkitChecker) getHiddenCandidates() []int32 {
	var hidden []int32

	// 1. 扫描 /proc
	procPids := make(map[int32]bool)
	entries, err := os.ReadDir("/proc")
	if err != nil {
		globalLogger.Warn("无法读取 /proc: %v", err)
		return hidden
	}

	for _, entry := range entries {
		if entry.IsDir() {
			pid, err := strconv.Atoi(entry.Name())
			if err == nil {
				procPids[int32(pid)] = true
			}
		}
	}

	// 2. 扫描 API (gopsutil)
	apiPids := make(map[int32]bool)
	procs, err := rc.cache.Get()
	if err != nil {
		globalLogger.Warn("无法获取进程列表: %v", err)
		return hidden
	}

	for _, p := range procs {
		apiPids[p.Pid] = true
	}

	// 3. 对比
	for pid := range procPids {
		if !apiPids[pid] {
			hidden = append(hidden, pid)
		}
	}

	return hidden
}
