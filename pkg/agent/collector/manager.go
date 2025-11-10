package collector

import (
	"encoding/json"

	"github.com/dushixiang/pika/internal/protocol"
	"github.com/dushixiang/pika/pkg/agent/config"
)

// WebSocketWriter 定义 WebSocket 写入接口
type WebSocketWriter interface {
	WriteJSON(v interface{}) error
}

// Manager 采集器管理器
type Manager struct {
	cpuCollector         *CPUCollector
	memoryCollector      *MemoryCollector
	diskCollector        *DiskCollector
	diskIOCollector      *DiskIOCollector
	networkCollector     *NetworkCollector
	loadCollector        *LoadCollector
	hostCollector        *HostCollector
	temperatureCollector *TemperatureCollector
	gpuCollector         *GPUCollector
}

// NewManager 创建采集器管理器
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		cpuCollector:         NewCPUCollector(),
		memoryCollector:      NewMemoryCollector(),
		diskCollector:        NewDiskCollector(),
		diskIOCollector:      NewDiskIOCollector(),
		networkCollector:     NewNetworkCollector(cfg),
		loadCollector:        NewLoadCollector(),
		hostCollector:        NewHostCollector(),
		temperatureCollector: NewTemperatureCollector(),
		gpuCollector:         NewGPUCollector(),
	}
}

// CollectAndSendCPU 采集并发送 CPU 指标
func (m *Manager) CollectAndSendCPU(conn WebSocketWriter) error {
	cpuData, err := m.cpuCollector.Collect()
	if err != nil {
		return err
	}

	return m.sendMetrics(conn, protocol.MetricTypeCPU, cpuData)
}

// CollectAndSendMemory 采集并发送内存指标
func (m *Manager) CollectAndSendMemory(conn WebSocketWriter) error {
	memData, err := m.memoryCollector.Collect()
	if err != nil {
		return err
	}

	return m.sendMetrics(conn, protocol.MetricTypeMemory, memData)
}

// CollectAndSendDisk 采集并发送磁盘指标
func (m *Manager) CollectAndSendDisk(conn WebSocketWriter) error {
	diskDataList, err := m.diskCollector.Collect()
	if err != nil {
		return err
	}
	return m.sendMetrics(conn, protocol.MetricTypeDisk, diskDataList)
}

// CollectAndSendDiskIO 采集并发送磁盘 IO 指标
func (m *Manager) CollectAndSendDiskIO(conn WebSocketWriter) error {
	diskIODataList, err := m.diskIOCollector.Collect()
	if err != nil {
		return err
	}
	return m.sendMetrics(conn, protocol.MetricTypeDiskIO, diskIODataList)
}

// CollectAndSendNetwork 采集并发送网络指标
func (m *Manager) CollectAndSendNetwork(conn WebSocketWriter) error {
	networkDataList, err := m.networkCollector.Collect()
	if err != nil {
		return err
	}
	return m.sendMetrics(conn, protocol.MetricTypeNetwork, networkDataList)
}

// CollectAndSendLoad 采集并发送系统负载指标
func (m *Manager) CollectAndSendLoad(conn WebSocketWriter) error {
	loadData, err := m.loadCollector.Collect()
	if err != nil {
		return err
	}
	return m.sendMetrics(conn, protocol.MetricTypeLoad, loadData)
}

// CollectAndSendHost 采集并发送主机信息
func (m *Manager) CollectAndSendHost(conn WebSocketWriter) error {
	hostData, err := m.hostCollector.Collect()
	if err != nil {
		return err
	}

	return m.sendMetrics(conn, protocol.MetricTypeHost, hostData)
}

// CollectAndSendGPU 采集并发送 GPU 指标
func (m *Manager) CollectAndSendGPU(conn WebSocketWriter) error {
	gpuDataList, err := m.gpuCollector.Collect()
	if err != nil || len(gpuDataList) == 0 {
		// GPU 监控不是必须的,失败或无数据时直接返回
		return nil
	}

	return m.sendMetrics(conn, protocol.MetricTypeGPU, gpuDataList)
}

// CollectAndSendTemperature 采集并发送温度信息
func (m *Manager) CollectAndSendTemperature(conn WebSocketWriter) error {
	tempDataList, err := m.temperatureCollector.Collect()
	if err != nil || len(tempDataList) == 0 {
		// 温度监控不是必须的,失败或无数据时直接返回
		return nil
	}

	return m.sendMetrics(conn, protocol.MetricTypeTemperature, tempDataList)
}

// sendMetrics 发送指标数据
func (m *Manager) sendMetrics(conn WebSocketWriter, metricType protocol.MetricType, data interface{}) error {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	metrics := protocol.MetricsWrapper{
		Type: metricType,
		Data: json.RawMessage(dataBytes),
	}

	metricsData, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	msg := protocol.Message{
		Type: protocol.MessageTypeMetrics,
		Data: metricsData,
	}

	return conn.WriteJSON(msg)
}
