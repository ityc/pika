package collector

import (
	"github.com/dushixiang/pika/internal/protocol"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
)

// LoadCollector 系统负载监控采集器
type LoadCollector struct{}

// NewLoadCollector 创建负载采集器
func NewLoadCollector() *LoadCollector {
	return &LoadCollector{}
}

// Collect 采集系统负载数据
func (l *LoadCollector) Collect() (*protocol.LoadData, error) {
	avgStat, err := load.Avg()
	if err != nil {
		return nil, err
	}

	return &protocol.LoadData{
		Load1:  avgStat.Load1,
		Load5:  avgStat.Load5,
		Load15: avgStat.Load15,
	}, nil
}

// HostCollector 主机信息采集器
type HostCollector struct{}

// NewHostCollector 创建主机信息采集器
func NewHostCollector() *HostCollector {
	return &HostCollector{}
}

// Collect 采集主机信息（定期采集以检测主机名等变化）
func (h *HostCollector) Collect() (*protocol.HostInfoData, error) {
	hostInfo, err := host.Info()
	if err != nil {
		return nil, err
	}

	hostData := &protocol.HostInfoData{
		Hostname:        hostInfo.Hostname,
		Uptime:          hostInfo.Uptime,
		BootTime:        hostInfo.BootTime,
		Procs:           hostInfo.Procs,
		OS:              hostInfo.OS,
		Platform:        hostInfo.Platform,
		PlatformFamily:  hostInfo.PlatformFamily,
		PlatformVersion: hostInfo.PlatformVersion,
		KernelVersion:   hostInfo.KernelVersion,
		KernelArch:      hostInfo.KernelArch,
	}

	// 尝试获取虚拟化信息
	if hostInfo.VirtualizationSystem != "" {
		hostData.VirtualizationSystem = hostInfo.VirtualizationSystem
		hostData.VirtualizationRole = hostInfo.VirtualizationRole
	}

	return hostData, nil
}
