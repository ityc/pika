package id

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// Manager 管理探针的唯一标识
type Manager struct {
	idFilePath string
}

// NewManager 创建 ID 管理器
func NewManager() *Manager {
	return &Manager{
		idFilePath: getIDFilePath(),
	}
}

// getIDFilePath 获取 ID 文件路径
func getIDFilePath() string {
	// 获取用户主目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// 如果无法获取主目录，使用当前目录
		homeDir = "."
	}

	// 统一使用 ~/.pika/agent.id
	return filepath.Join(homeDir, ".pika", "agent.id")
}

// Load 加载或生成探针 ID
// 如果 ID 文件存在，则读取；否则生成新的 UUID 并保存
func (m *Manager) Load() (string, error) {
	// 尝试读取现有 ID
	if id, err := m.read(); err == nil && id != "" {
		return id, nil
	}

	// 生成新 ID
	id := uuid.NewString()

	// 保存 ID
	if err := m.save(id); err != nil {
		return "", fmt.Errorf("保存 agent ID 失败: %w", err)
	}

	return id, nil
}

// read 读取 ID 文件
func (m *Manager) read() (string, error) {
	data, err := os.ReadFile(m.idFilePath)
	if err != nil {
		return "", err
	}

	id := strings.TrimSpace(string(data))
	if id == "" {
		return "", fmt.Errorf("ID 文件为空")
	}

	return id, nil
}

// save 保存 ID 到文件
func (m *Manager) save(id string) error {
	// 确保目录存在
	dir := filepath.Dir(m.idFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 写入 ID 文件
	if err := os.WriteFile(m.idFilePath, []byte(id), 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

// GetPath 获取 ID 文件路径
func (m *Manager) GetPath() string {
	return m.idFilePath
}

// Exists 检查 ID 文件是否存在
func (m *Manager) Exists() bool {
	_, err := os.Stat(m.idFilePath)
	return err == nil
}

// Delete 删除 ID 文件
func (m *Manager) Delete() error {
	return os.Remove(m.idFilePath)
}
