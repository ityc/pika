package audit

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/dushixiang/pika/internal/protocol"
)

// 状态常量
const (
	StatusPass = "pass"
	StatusFail = "fail"
	StatusWarn = "warn"
	StatusSkip = "skip"
)

// Auditor VPS 安全审计器
type Auditor struct {
	config   *Config
	cache    *ProcessCache
	executor *CommandExecutor
	evidence *EvidenceCollector

	// 各个检查器
	rootkitChecker *RootkitChecker
	processChecker *ProcessChecker
	sshChecker     *SSHChecker
	networkChecker *NetworkChecker
	fileChecker    *FileChecker
	accountChecker *AccountChecker
}

// NewAuditor 创建审计器
func NewAuditor(config *Config) *Auditor {
	if config == nil {
		config = DefaultConfig()
	}

	// 初始化共享组件
	cache := NewProcessCache(config.PerformanceConfig.ProcessCacheDuration)
	executor := NewCommandExecutor(config.PerformanceConfig.CommandTimeout)
	evidence := NewEvidenceCollector()

	// 初始化各个检查器
	return &Auditor{
		config:   config,
		cache:    cache,
		executor: executor,
		evidence: evidence,

		rootkitChecker: NewRootkitChecker(config, cache, evidence),
		processChecker: NewProcessChecker(config, cache, evidence),
		sshChecker:     NewSSHChecker(config, cache, evidence, executor),
		networkChecker: NewNetworkChecker(config, cache, evidence),
		fileChecker:    NewFileChecker(config, evidence, executor),
		accountChecker: NewAccountChecker(config),
	}
}

// RunAudit 执行 VPS 安全审计
func (a *Auditor) RunAudit() (*protocol.VPSAuditResult, error) {
	startTime := time.Now().UnixMilli()

	// 检查操作系统
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("只支持 Linux 系统")
	}

	// 检查运行权限
	if os.Geteuid() != 0 {
		return nil, fmt.Errorf("需要root权限运行完整审计")
	}

	globalLogger.Info("开始安全审计...")

	// 获取系统信息
	sysInfoCollector := NewSystemInfoCollector(a.executor)
	systemInfo, err := sysInfoCollector.Collect()
	if err != nil {
		return nil, fmt.Errorf("获取系统信息失败: %w", err)
	}

	// 收集警告
	warningCollector := NewWarningCollector()

	// 执行所有安全检查
	var securityChecks []protocol.SecurityCheck

	checks := []struct {
		name string
		fn   func() protocol.SecurityCheck
	}{
		{"Rootkit检测", a.rootkitChecker.Check},
		{"可疑进程检测", a.processChecker.CheckSuspiciousProcesses},
		{"进程环境变量检查", a.processChecker.CheckSuspiciousEnvVars},
		{"SSH安全检查", a.sshChecker.Check},
		{"端口监听检查", a.networkChecker.CheckListeningPorts},
		{"网络连接检查", a.networkChecker.CheckNetworkConnections},
		{"可疑文件检查", a.fileChecker.CheckSuspiciousFiles},
		{"文件完整性检查", a.fileChecker.CheckFileIntegrity},
		{"不可变文件检查", a.fileChecker.CheckImmutableFiles},
		{"系统账户检查", a.accountChecker.CheckSystemAccounts},
	}

	for _, check := range checks {
		globalLogger.Debug("执行检查: %s", check.name)
		result := check.fn()
		securityChecks = append(securityChecks, result)

		// 收集警告
		for _, detail := range result.Details {
			if detail.Status == StatusSkip && detail.Message != "" {
				warningCollector.Add(fmt.Sprintf("[%s] %s", result.Category, detail.Message))
			}
		}
	}

	endTime := time.Now().UnixMilli()

	// 计算风险评分（使用改进的算法）
	riskScore, threatLevel, recommendations := a.calculateRiskScore(securityChecks)

	globalLogger.Info("审计完成，耗时 %dms, 风险评分: %d, 威胁等级: %s",
		endTime-startTime, riskScore, threatLevel)

	return &protocol.VPSAuditResult{
		SystemInfo:      *systemInfo,
		SecurityChecks:  securityChecks,
		StartTime:       startTime,
		EndTime:         endTime,
		RiskScore:       riskScore,
		ThreatLevel:     threatLevel,
		Recommendations: recommendations,
		AuditWarnings:   warningCollector.GetAll(),
	}, nil
}

// calculateRiskScore 计算综合风险评分（改进版）
func (a *Auditor) calculateRiskScore(checks []protocol.SecurityCheck) (int, string, []string) {
	score := 0
	var recommendations []string

	// 使用配置的权重
	for _, check := range checks {
		weight, exists := a.config.ScoringConfig.Weights[check.Category]
		if !exists {
			// 默认权重
			weight = CheckWeight{
				Category:  check.Category,
				FailScore: 20,
				WarnScore: 5,
			}
			globalLogger.Warn("检查类别 '%s' 没有配置权重，使用默认值", check.Category)
		}

		for _, detail := range check.Details {
			switch detail.Status {
			case StatusFail:
				score += weight.FailScore
				recommendations = append(recommendations,
					fmt.Sprintf("【紧急】%s: %s", check.Category, detail.Message))
			case StatusWarn:
				score += weight.WarnScore
				recommendations = append(recommendations,
					fmt.Sprintf("【警告】%s: %s", check.Category, detail.Message))
			}
		}
	}

	// 限制最大分数为 100
	if score > 100 {
		score = 100
	}

	// 确定威胁等级（使用配置的阈值）
	var level string
	switch {
	case score >= a.config.ScoringConfig.CriticalThreshold:
		level = "critical"
	case score >= a.config.ScoringConfig.HighThreshold:
		level = "high"
	case score >= a.config.ScoringConfig.MediumThreshold:
		level = "medium"
	default:
		level = "low"
	}

	// 限制建议数量
	if len(recommendations) > a.config.ScoringConfig.MaxRecommendations {
		recommendations = recommendations[:a.config.ScoringConfig.MaxRecommendations]
	}

	return score, level, recommendations
}

// RunAuditWithConfig 使用自定义配置执行审计
func RunAuditWithConfig(config *Config) (*protocol.VPSAuditResult, error) {
	auditor := NewAuditor(config)
	return auditor.RunAudit()
}

// RunAudit 使用默认配置执行审计（保持向后兼容）
func RunAudit() (*protocol.VPSAuditResult, error) {
	return RunAuditWithConfig(nil)
}
