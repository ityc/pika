package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/dushixiang/pika/pkg/agent/config"
)

// VersionInfo 版本信息
type VersionInfo struct {
	Version string `json:"version"`
}

// Updater 自动更新器
type Updater struct {
	cfg            *config.Config
	currentVer     string
	httpClient     *http.Client
	executablePath string
}

// New 创建更新器
func New(cfg *config.Config, currentVer string) (*Updater, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("获取可执行文件路径失败: %w", err)
	}

	return &Updater{
		cfg:        cfg,
		currentVer: currentVer,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		executablePath: execPath,
	}, nil
}

// Start 启动自动更新检查
func (u *Updater) Start(ctx context.Context) {
	if !u.cfg.AutoUpdate.Enabled {
		log.Println("自动更新已禁用")
		return
	}

	log.Printf("自动更新已启用，检查间隔: %v", u.cfg.GetUpdateCheckInterval())

	// 立即检查一次
	u.checkAndUpdate()

	// 定时检查
	ticker := time.NewTicker(u.cfg.GetUpdateCheckInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			u.checkAndUpdate()
		case <-ctx.Done():
			log.Println("停止自动更新检查")
			return
		}
	}
}

// checkAndUpdate 检查并更新
func (u *Updater) checkAndUpdate() {
	log.Println("🔍 检查更新...")

	// 获取最新版本信息
	versionInfo, err := u.fetchLatestVersion()
	if err != nil {
		log.Printf("⚠️  获取版本信息失败: %v", err)
		return
	}

	// 比较版本
	if versionInfo.Version == u.currentVer {
		log.Printf("✅ 当前已是最新版本: %s", u.currentVer)
		return
	}

	log.Printf("🆕 发现新版本: %s (当前版本: %s)", versionInfo.Version, u.currentVer)

	// 下载新版本
	if err := u.downloadAndUpdate(versionInfo); err != nil {
		log.Printf("❌ 更新失败: %v", err)
		return
	}

	log.Println("✅ 更新成功，将在下次重启时生效")
}

// fetchLatestVersion 获取最新版本信息
func (u *Updater) fetchLatestVersion() (*VersionInfo, error) {
	updateURL := u.cfg.GetUpdateURL()
	return CheckUpdate(updateURL, u.currentVer)
}

// downloadAndUpdate 下载并更新
func (u *Updater) downloadAndUpdate(versionInfo *VersionInfo) error {
	log.Printf("📥 下载新版本: %s", versionInfo.Version)

	downloadURL := u.cfg.GetDownloadURL()
	if err := Update(downloadURL); err != nil {
		return err
	}

	log.Printf("✅ 新版本已安装到: %s", u.executablePath)
	return nil
}

// CheckUpdate 手动检查更新（用于命令行）
func CheckUpdate(updateURL, currentVer string) (*VersionInfo, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	url := fmt.Sprintf("%s?os=%s&arch=%s", updateURL, runtime.GOOS, runtime.GOARCH)

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP 状态码: %d", resp.StatusCode)
	}

	var versionInfo VersionInfo
	if err := json.NewDecoder(resp.Body).Decode(&versionInfo); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &versionInfo, nil
}

// Update 手动更新（用于命令行）
func Update(downloadURL string) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %w", err)
	}

	// 解析实际路径（处理符号链接）
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("解析可执行文件路径失败: %w", err)
	}

	client := &http.Client{
		Timeout: 300 * time.Second,
	}

	// 下载文件
	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP 状态码: %d", resp.StatusCode)
	}

	// 创建临时文件
	tmpFile := execPath + ".new"
	out, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer out.Close()
	defer os.Remove(tmpFile)

	// 写入文件
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	log.Printf("下载完成，文件大小: %d 字节", written)

	// 根据操作系统选择不同的更新策略
	if runtime.GOOS == "windows" {
		// Windows: 使用批处理脚本延迟替换
		return updateOnWindows(execPath, tmpFile)
	}

	// Unix-like: 直接替换
	return updateOnUnix(execPath, tmpFile)
}

// updateOnUnix Unix-like 系统的更新逻辑
func updateOnUnix(execPath, tmpFile string) error {
	// 备份旧版本
	backupPath := execPath + ".bak"
	if err := os.Rename(execPath, backupPath); err != nil {
		return fmt.Errorf("备份旧版本失败: %w", err)
	}

	// 替换为新版本
	if err := os.Rename(tmpFile, execPath); err != nil {
		// 恢复备份
		os.Rename(backupPath, execPath)
		return fmt.Errorf("替换新版本失败: %w", err)
	}

	// 删除备份
	os.Remove(backupPath)

	log.Println("✅ 更新完成，进程即将退出，等待系统服务重启...")

	// 退出当前进程，让系统服务管理器（systemd/supervisor等）自动重启
	// 注意：这要求服务配置了自动重启（如 systemd 的 Restart=always）
	os.Exit(0)

	return nil
}

// updateOnWindows Windows 系统的更新逻辑
func updateOnWindows(execPath, tmpFile string) error {
	// 在 Windows 上，无法直接替换正在运行的可执行文件
	// 策略: 创建一个批处理脚本来延迟替换和重启

	batScript := execPath + ".update.bat"
	batContent := fmt.Sprintf(`@echo off
timeout /t 2 /nobreak >nul
move /y "%s" "%s.bak"
move /y "%s" "%s"
del "%s.bak"
start "" "%s"
del "%%~f0"
`, execPath, execPath, tmpFile, execPath, execPath, execPath)

	if err := os.WriteFile(batScript, []byte(batContent), 0755); err != nil {
		return fmt.Errorf("创建更新脚本失败: %w", err)
	}

	log.Println("✅ 更新完成，准备重启进程...")

	// 启动批处理脚本并退出当前进程
	cmd := exec.Command("cmd.exe", "/C", batScript)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		os.Remove(batScript)
		return fmt.Errorf("启动更新脚本失败: %w", err)
	}

	// 让系统服务管理器来重启（当前进程退出后）
	log.Println("进程即将退出，等待系统服务重启...")
	os.Exit(0)

	return nil
}
