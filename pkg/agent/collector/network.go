package collector

import (
	"time"

	"github.com/dushixiang/pika/internal/protocol"
	"github.com/dushixiang/pika/pkg/agent/config"
	"github.com/shirou/gopsutil/v4/net"
)

// NetworkCollector 网络监控采集器
type NetworkCollector struct {
	config        *config.Config                // 配置信息
	lastStats     map[string]net.IOCountersStat // 上次采集的统计数据
	lastCollectAt time.Time                     // 上次采集时间
}

// NewNetworkCollector 创建网络采集器
func NewNetworkCollector(cfg *config.Config) *NetworkCollector {
	return &NetworkCollector{
		config:    cfg,
		lastStats: make(map[string]net.IOCountersStat),
	}
}

// Collect 采集网络数据(计算自上次采集以来的增量)
func (n *NetworkCollector) Collect() ([]protocol.NetworkData, error) {
	now := time.Now()

	// 计算距离上次采集的时间间隔(秒)
	var intervalSeconds float64
	if !n.lastCollectAt.IsZero() {
		intervalSeconds = now.Sub(n.lastCollectAt).Seconds()
	}

	// 获取网络接口信息
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	// 创建接口信息映射
	interfaceMap := make(map[string]*protocol.NetworkData)
	for _, iface := range interfaces {
		// 使用配置中的排除规则过滤网卡
		if n.config.ShouldExcludeNetworkInterface(iface.Name) {
			continue
		}

		// 获取 IP 地址列表
		var addrs []string
		for _, addr := range iface.Addrs {
			addrs = append(addrs, addr.Addr)
		}

		interfaceMap[iface.Name] = &protocol.NetworkData{
			Interface:  iface.Name,
			MacAddress: iface.HardwareAddr,
			Addrs:      addrs,
		}
	}

	// 获取网络 IO 统计
	ioCounters, err := net.IOCounters(true)
	if err != nil {
		return nil, err
	}

	var networkDataList []protocol.NetworkData
	for _, counter := range ioCounters {
		// 使用配置中的排除规则过滤网卡
		if n.config.ShouldExcludeNetworkInterface(counter.Name) {
			continue
		}

		// 如果已有接口信息,则更新;否则创建新的
		netData := interfaceMap[counter.Name]
		if netData == nil {
			netData = &protocol.NetworkData{
				Interface: counter.Name,
			}
		}

		// 计算增量(如果是第一次采集,则使用当前值)
		lastStat, exists := n.lastStats[counter.Name]
		if exists && intervalSeconds > 0 {
			// 计算增量并转换为每秒速率
			bytesSentDelta := counter.BytesSent - lastStat.BytesSent
			bytesRecvDelta := counter.BytesRecv - lastStat.BytesRecv

			// 存储每秒的速率(转为整数)
			netData.BytesSentRate = uint64(float64(bytesSentDelta) / intervalSeconds)
			netData.BytesRecvRate = uint64(float64(bytesRecvDelta) / intervalSeconds)
			netData.BytesSentTotal = counter.BytesSent
			netData.BytesRecvTotal = counter.BytesRecv
		} else {
			// 第一次采集,使用0值(避免返回巨大的累计值)
			netData.BytesSentRate = 0
			netData.BytesRecvRate = 0
			netData.BytesSentTotal = counter.BytesSent
			netData.BytesRecvTotal = counter.BytesRecv
		}

		// 保存当前统计数据用于下次计算增量
		n.lastStats[counter.Name] = counter

		networkDataList = append(networkDataList, *netData)
	}

	// 更新采集时间
	n.lastCollectAt = now

	return networkDataList, nil
}
