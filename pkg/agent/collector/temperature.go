package collector

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/dushixiang/pika/internal/protocol"
)

// TemperatureCollector 温度监控采集器
type TemperatureCollector struct{}

// NewTemperatureCollector 创建温度采集器
func NewTemperatureCollector() *TemperatureCollector {
	return &TemperatureCollector{}
}

// Collect 采集温度数据（某些系统可能不支持）
func (t *TemperatureCollector) Collect() ([]*protocol.TemperatureData, error) {
	switch runtime.GOOS {
	case "linux":
		return t.collectLinux()
	case "darwin":
		return t.collectDarwin()
	default:
		// 不支持的系统，返回空数组
		return []*protocol.TemperatureData{}, nil
	}
}

// collectLinux 在 Linux 系统上采集温度数据
func (t *TemperatureCollector) collectLinux() ([]*protocol.TemperatureData, error) {
	// 尝试使用 sensors 命令
	_, err := exec.LookPath("sensors")
	if err != nil {
		// sensors 命令不可用，返回空数组
		return []*protocol.TemperatureData{}, nil
	}

	cmd := exec.Command("sensors", "-A")
	output, err := cmd.Output()
	if err != nil {
		return []*protocol.TemperatureData{}, nil
	}

	var tempDataList []*protocol.TemperatureData
	lines := strings.Split(string(output), "\n")
	currentSensor := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 检测传感器名称
		if !strings.Contains(line, ":") {
			currentSensor = line
			continue
		}

		// 解析温度行
		if strings.Contains(line, "°C") {
			parts := strings.Split(line, ":")
			if len(parts) < 2 {
				continue
			}

			label := strings.TrimSpace(parts[0])
			valueStr := strings.TrimSpace(parts[1])

			// 提取温度值
			tempStr := strings.Split(valueStr, "°C")[0]
			tempStr = strings.TrimSpace(strings.TrimPrefix(tempStr, "+"))

			temp, err := strconv.ParseFloat(tempStr, 64)
			if err != nil {
				continue
			}

			sensorKey := currentSensor + "_" + label
			tempData := &protocol.TemperatureData{
				SensorKey:   sensorKey,
				Temperature: temp,
			}

			tempDataList = append(tempDataList, tempData)
		}
	}

	return tempDataList, nil
}

// collectDarwin 在 macOS 系统上采集温度数据
func (t *TemperatureCollector) collectDarwin() ([]*protocol.TemperatureData, error) {
	// macOS 的温度监控需要特殊工具，如 osx-cpu-temp
	_, err := exec.LookPath("osx-cpu-temp")
	if err != nil {
		// 工具不可用，返回空数组
		return []*protocol.TemperatureData{}, nil
	}

	cmd := exec.Command("osx-cpu-temp")
	output, err := cmd.Output()
	if err != nil {
		return []*protocol.TemperatureData{}, nil
	}

	// 解析输出，格式类似：60.0°C
	outputStr := strings.TrimSpace(string(output))
	tempStr := strings.Split(outputStr, "°C")[0]
	temp, err := strconv.ParseFloat(tempStr, 64)
	if err != nil {
		return []*protocol.TemperatureData{}, nil
	}

	tempData := &protocol.TemperatureData{
		SensorKey:   "CPU",
		Temperature: temp,
	}

	return []*protocol.TemperatureData{tempData}, nil
}
