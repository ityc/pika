package audit

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	// 验证关键配置
	if config.ProcessConfig.HighCPUThreshold != 70.0 {
		t.Errorf("Expected HighCPUThreshold to be 70.0, got %f", config.ProcessConfig.HighCPUThreshold)
	}

	if len(config.ProcessConfig.MinerKeywords) == 0 {
		t.Error("MinerKeywords should not be empty")
	}

	if config.ScoringConfig.CriticalThreshold != 80 {
		t.Errorf("Expected CriticalThreshold to be 80, got %d", config.ScoringConfig.CriticalThreshold)
	}
}

func TestProcessCache(t *testing.T) {
	cache := NewProcessCache(100 * time.Millisecond)

	// 第一次获取
	procs1, err := cache.Get()
	if err != nil {
		t.Fatalf("Failed to get processes: %v", err)
	}

	if len(procs1) == 0 {
		t.Error("Expected processes, got empty list")
	}

	// 第二次获取（应该从缓存）
	procs2, err := cache.Get()
	if err != nil {
		t.Fatalf("Failed to get cached processes: %v", err)
	}

	// 应该是相同的引用
	if len(procs1) != len(procs2) {
		t.Error("Cache should return same process list")
	}

	// 等待缓存过期
	time.Sleep(150 * time.Millisecond)

	// 获取新的进程列表
	procs3, err := cache.Get()
	if err != nil {
		t.Fatalf("Failed to get refreshed processes: %v", err)
	}

	if len(procs3) == 0 {
		t.Error("Expected processes after cache expiry")
	}
}

func TestCommandExecutor(t *testing.T) {
	executor := NewCommandExecutor(5 * time.Second)

	// 测试成功的命令
	output, err := executor.Execute("echo", "hello")
	if err != nil {
		t.Fatalf("Failed to execute command: %v", err)
	}

	if output != "hello\n" {
		t.Errorf("Expected 'hello\\n', got '%s'", output)
	}

	// 测试超时
	executor2 := NewCommandExecutor(100 * time.Millisecond)
	_, err = executor2.Execute("sleep", "1")
	if err == nil {
		t.Error("Expected timeout error")
	}
}

func TestFileHashCache(t *testing.T) {
	cache := NewFileHashCache()

	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "test-hash-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("test content")
	tmpFile.Close()

	// 第一次计算
	hash1 := cache.GetSHA256(tmpFile.Name())
	if hash1 == "" {
		t.Error("Expected hash, got empty string")
	}

	// 第二次应该从缓存
	hash2 := cache.GetSHA256(tmpFile.Name())
	if hash1 != hash2 {
		t.Error("Cache should return same hash")
	}

	// 修改文件
	os.WriteFile(tmpFile.Name(), []byte("modified content"), 0644)

	// 应该计算新的哈希
	hash3 := cache.GetSHA256(tmpFile.Name())
	if hash3 == hash1 {
		t.Error("Hash should be different after file modification")
	}
}

func TestStringUtils(t *testing.T) {
	su := &StringUtils{}

	// 测试 Truncate
	if result := su.Truncate("hello", 10); result != "hello" {
		t.Errorf("Expected 'hello', got '%s'", result)
	}

	if result := su.Truncate("hello world", 5); result != "hello" {
		t.Errorf("Expected 'hello', got '%s'", result)
	}

	// 测试 ContainsAny
	keywords := []string{"foo", "bar", "baz"}

	if !su.ContainsAny("hello foobar", keywords) {
		t.Error("Expected true for 'hello foobar'")
	}

	if su.ContainsAny("hello world", keywords) {
		t.Error("Expected false for 'hello world'")
	}

	// 测试大小写不敏感
	if !su.ContainsAny("HELLO FOO", keywords) {
		t.Error("Expected true for case-insensitive match")
	}
}

func TestIsLocalIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"127.0.0.1", true},
		{"::1", true},
		{"localhost", true},
		{"192.168.1.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
	}

	for _, tt := range tests {
		result := IsLocalIP(tt.ip)
		if result != tt.expected {
			t.Errorf("IsLocalIP(%s) = %v, expected %v", tt.ip, result, tt.expected)
		}
	}
}

func TestBatchProcessor(t *testing.T) {
	bp := NewBatchProcessor(3)

	items := []string{"a", "b", "c", "d", "e", "f", "g"}
	processed := []string{}

	err := bp.Process(items, func(batch []string) error {
		processed = append(processed, batch...)
		return nil
	})

	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if len(processed) != len(items) {
		t.Errorf("Expected %d items, got %d", len(items), len(processed))
	}

	// 验证顺序
	for i, item := range items {
		if processed[i] != item {
			t.Errorf("Order mismatch at index %d: expected %s, got %s", i, item, processed[i])
		}
	}
}

func TestWarningCollector(t *testing.T) {
	wc := NewWarningCollector()

	wc.Add("warning 1")
	wc.Add("warning 2")
	wc.Add("warning 3")

	warnings := wc.GetAll()

	if len(warnings) != 3 {
		t.Errorf("Expected 3 warnings, got %d", len(warnings))
	}

	if warnings[0] != "warning 1" {
		t.Errorf("Expected 'warning 1', got '%s'", warnings[0])
	}
}

// 集成测试（需要 root 权限和 Linux 环境）
func TestAuditorIntegration(t *testing.T) {
	// 跳过如果不是 Linux 或没有 root 权限
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=true to run")
	}

	auditor := NewAuditor(nil)

	result, err := auditor.RunAudit()
	if err != nil {
		t.Fatalf("RunAudit failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if len(result.SecurityChecks) == 0 {
		t.Error("Expected security checks, got none")
	}

	if result.RiskScore < 0 || result.RiskScore > 100 {
		t.Errorf("Invalid risk score: %d", result.RiskScore)
	}

	if result.ThreatLevel == "" {
		t.Error("Threat level should not be empty")
	}

	t.Logf("Audit completed in %dms", result.EndTime-result.StartTime)
	t.Logf("Risk Score: %d, Threat Level: %s", result.RiskScore, result.ThreatLevel)
	t.Logf("Security Checks: %d", len(result.SecurityChecks))
}

// 性能基准测试
func BenchmarkProcessCache(b *testing.B) {
	cache := NewProcessCache(1 * time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get()
	}
}

func BenchmarkFileHashCache(b *testing.B) {
	cache := NewFileHashCache()

	// 创建临时文件
	tmpFile, _ := os.CreateTemp("", "bench-*.txt")
	tmpFile.WriteString("test content for benchmarking")
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.GetSHA256(tmpFile.Name())
	}
}
