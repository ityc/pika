package audit

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dushixiang/pika/internal/protocol"
)

// 这个文件包含文件检查、账户检查、Cron 检查和登录历史检查
// 为简化重构，将这些相对独立的检查整合在一起

// FileChecker 文件检查器
type FileChecker struct {
	config   *Config
	evidence *EvidenceCollector
	executor *CommandExecutor
}

// NewFileChecker 创建文件检查器
func NewFileChecker(config *Config, evidence *EvidenceCollector, executor *CommandExecutor) *FileChecker {
	return &FileChecker{
		config:   config,
		evidence: evidence,
		executor: executor,
	}
}

// CheckSuspiciousFiles 检查可疑文件
func (fc *FileChecker) CheckSuspiciousFiles() protocol.SecurityCheck {
	check := protocol.SecurityCheck{
		Category: "suspicious_files",
		Status:   StatusPass,
		Message:  "可疑文件检查",
		Details:  []protocol.SecurityCheckSub{},
	}

	suspiciousCount := 0

	// 检查临时目录下的可执行文件
	for _, dir := range fc.config.FileConfig.TempDirs {
		files := fc.findSuspiciousExecutables(dir)
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

// CheckFileIntegrity 检查系统文件完整性（修复bug版本）
func (fc *FileChecker) CheckFileIntegrity() protocol.SecurityCheck {
	check := protocol.SecurityCheck{
		Category: "file_integrity",
		Status:   StatusPass,
		Message:  "系统文件完整性检查",
		Details:  []protocol.SecurityCheckSub{},
	}

	suspiciousCount := 0

	// 1. 检查 /etc/ld.so.preload
	if content, err := os.ReadFile("/etc/ld.so.preload"); err == nil {
		contentStr := strings.TrimSpace(string(content))
		if contentStr != "" {
			suspiciousCount++
			check.Status = StatusFail
			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:    "ld_preload_exists",
				Status:  StatusFail,
				Message: fmt.Sprintf("发现动态链接库劫持配置 (/etc/ld.so.preload): %s", contentStr),
				Evidence: &protocol.Evidence{
					FilePath:  "/etc/ld.so.preload",
					RiskLevel: "critical",
				},
			})
		}
	}

	// 2. 检查系统文件最近修改时间
	recentModifiedCount := 0
	for _, binPath := range fc.config.FileConfig.CriticalBinaries {
		if _, err := os.Stat(binPath); err != nil {
			continue
		}

		isModified, verifyOutput := fc.verifyFileByModTime(binPath)
		if isModified {
			recentModifiedCount++
			check.Status = StatusWarn
			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:    "recently_modified_binary",
				Status:  StatusWarn,
				Message: verifyOutput,
			})
		}
	}

	if suspiciousCount > 0 || recentModifiedCount > 0 {
		check.Message = fmt.Sprintf("发现 %d 个文件完整性异常", suspiciousCount+recentModifiedCount)
	} else {
		check.Message = "核心文件完整性正常"
	}

	return check
}

// CheckImmutableFiles 检查不可变文件
func (fc *FileChecker) CheckImmutableFiles() protocol.SecurityCheck {
	check := protocol.SecurityCheck{
		Category: "immutable_files",
		Status:   StatusPass,
		Message:  "不可变文件检查",
		Details:  []protocol.SecurityCheckSub{},
	}

	// 检查 lsattr 是否可用
	if _, err := exec.LookPath("lsattr"); err != nil {
		check.Status = StatusSkip
		check.Message = "lsattr 命令不可用，无法检查不可变属性"
		check.Details = append(check.Details, protocol.SecurityCheckSub{
			Name:    "lsattr_unavailable",
			Status:  StatusSkip,
			Message: "系统未安装 lsattr 工具（需要 e2fsprogs 包）",
		})
		return check
	}

	suspiciousCount := 0

	for _, file := range fc.config.FileConfig.ImmutableCheckFiles {
		if _, err := os.Stat(file); err != nil {
			continue
		}

		cmd := exec.Command("lsattr", "-d", file)
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		outStr := string(output)
		if strings.Contains(outStr, "-i-") || strings.Contains(outStr, "----i") {
			suspiciousCount++
			check.Status = StatusFail

			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:    "immutable_file_found",
				Status:  StatusFail,
				Message: fmt.Sprintf("关键文件被锁定(不可变): %s (可能是Rootkit保护)", file),
				Evidence: &protocol.Evidence{
					FilePath:  file,
					RiskLevel: "high",
				},
			})
		}
	}

	if suspiciousCount > 0 {
		check.Message = fmt.Sprintf("发现 %d 个异常锁定的系统文件", suspiciousCount)
	} else {
		check.Message = "未发现被恶意锁定的文件"
	}

	return check
}

// findSuspiciousExecutables 查找可疑的可执行文件
func (fc *FileChecker) findSuspiciousExecutables(dir string) []string {
	var suspicious []string

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
			age := now.Sub(info.ModTime())
			sizeKB := info.Size() / 1024
			sizeMB := info.Size() / 1024 / 1024

			if age < time.Duration(fc.config.FileConfig.RecentExecutableHours)*time.Hour {
				suspicious = append(suspicious, fmt.Sprintf("%s 下的最近可执行文件: %s (大小: %dKB, 创建时间: %s)",
					dir, fullPath, sizeKB, info.ModTime().Format("2006-01-02 15:04:05")))
			} else if sizeMB > fc.config.FileConfig.LargeExecutableMB {
				suspicious = append(suspicious, fmt.Sprintf("%s 下的大型可执行文件: %s (大小: %dMB)",
					dir, fullPath, sizeMB))
			}
		}
	}

	return suspicious
}

// verifyFileByModTime 通过修改时间检查文件
func (fc *FileChecker) verifyFileByModTime(filePath string) (bool, string) {
	info, err := os.Stat(filePath)
	if err != nil {
		return false, ""
	}

	modTime := info.ModTime()
	age := time.Since(modTime)

	// 如果在最近 24 小时内被修改，标记为可疑
	if age < 24*time.Hour {
		return true, fmt.Sprintf("系统命令最近24小时内有更新: %s (请确认是否为正常运维)", filePath)
	}

	return false, ""
}

// AccountChecker 账户检查器
type AccountChecker struct {
	config *Config
}

// NewAccountChecker 创建账户检查器
func NewAccountChecker(config *Config) *AccountChecker {
	return &AccountChecker{config: config}
}

// CheckSystemAccounts 检查系统账户异常
func (ac *AccountChecker) CheckSystemAccounts() protocol.SecurityCheck {
	check := protocol.SecurityCheck{
		Category: "system_accounts",
		Status:   StatusPass,
		Message:  "系统账户检查",
		Details:  []protocol.SecurityCheckSub{},
	}

	// 1. 检查可疑账户
	suspiciousAccounts := ac.findSuspiciousAccounts()
	for _, acc := range suspiciousAccounts {
		check.Status = StatusWarn
		check.Details = append(check.Details, protocol.SecurityCheckSub{
			Name:    "suspicious_account",
			Status:  StatusWarn,
			Message: acc,
		})
	}

	// 2. 检查无密码账户
	noPasswordAccounts := ac.findNoPasswordAccounts()
	for _, acc := range noPasswordAccounts {
		check.Status = StatusFail
		check.Details = append(check.Details, protocol.SecurityCheckSub{
			Name:    "no_password_account",
			Status:  StatusFail,
			Message: fmt.Sprintf("无密码账户: %s", acc),
		})
	}

	// 3. 检查 UID 为 0 的非 root 账户
	rootUIDAccounts := ac.findRootUIDAccounts()
	for _, acc := range rootUIDAccounts {
		check.Status = StatusFail
		check.Details = append(check.Details, protocol.SecurityCheckSub{
			Name:    "uid0_account",
			Status:  StatusFail,
			Message: fmt.Sprintf("UID为0的非root账户: %s", acc),
		})
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

// findSuspiciousAccounts 查找可疑账户
func (ac *AccountChecker) findSuspiciousAccounts() []string {
	var suspicious []string

	file, err := os.Open("/etc/passwd")
	if err != nil {
		return suspicious
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) < 7 {
			continue
		}

		username := parts[0]
		uidStr := parts[2]
		shell := parts[6]

		uid, _ := strconv.Atoi(uidStr)

		// 系统账户(UID<1000)但有可登录 shell
		if uid < 1000 && username != "root" {
			legitimateShells := []string{
				"/sbin/nologin", "/bin/false", "/usr/sbin/nologin",
				"/bin/sync", "/sbin/shutdown", "/sbin/halt",
			}

			isLegitimate := false
			for _, legitShell := range legitimateShells {
				if shell == legitShell {
					isLegitimate = true
					break
				}
			}

			if !isLegitimate {
				suspicious = append(suspicious, fmt.Sprintf("系统账户有可登录shell: %s (UID: %s, Shell: %s)", username, uidStr, shell))
			}
		}
	}

	return suspicious
}

// findNoPasswordAccounts 查找无密码账户
func (ac *AccountChecker) findNoPasswordAccounts() []string {
	var noPassword []string

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

			if password == "" {
				noPassword = append(noPassword, username)
			}
		}
	}

	return noPassword
}

// findRootUIDAccounts 查找 UID 为 0 的非 root 账户
func (ac *AccountChecker) findRootUIDAccounts() []string {
	var rootUID []string

	file, err := os.Open("/etc/passwd")
	if err != nil {
		return rootUID
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) >= 3 {
			username := parts[0]
			uid := parts[2]

			if uid == "0" && username != "root" {
				rootUID = append(rootUID, username)
			}
		}
	}

	return rootUID
}
