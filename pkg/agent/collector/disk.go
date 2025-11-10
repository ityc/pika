package collector

import (
	"runtime"
	"strings"

	"github.com/dushixiang/pika/internal/protocol"
	"github.com/shirou/gopsutil/v4/disk"
)

// DiskCollector 磁盘监控采集器
type DiskCollector struct {
}

// NewDiskCollector 创建磁盘采集器
func NewDiskCollector() *DiskCollector {
	return &DiskCollector{}
}

// shouldIgnorePartition 判断是否应该忽略该分区
func shouldIgnorePartition(partition disk.PartitionStat) bool {
	// 在 macOS 上过滤掉特殊文件系统
	if runtime.GOOS == "darwin" {
		// 忽略 devfs、com.apple.TimeMachine 等虚拟文件系统
		ignoredFsTypes := []string{
			"devfs",             // 设备文件系统
			"autofs",            // 自动挂载文件系统
			"mtmfs",             // Mobile Time Machine
			"com.apple.osxfuse", // FUSE 文件系统
		}

		for _, ignoredType := range ignoredFsTypes {
			if partition.Fstype == ignoredType {
				return true
			}
		}

		// 忽略特定的挂载点
		ignoredMountPoints := []string{
			"/dev",
			"/System/Volumes/VM",         // 虚拟内存
			"/System/Volumes/Preboot",    // Preboot 分区
			"/System/Volumes/Update",     // 更新分区
			"/System/Volumes/Hardware",   // 硬件分区
			"/System/Volumes/xarts",      // xART 分区
			"/System/Volumes/iSCPreboot", // iSC Preboot
			"/System/Volumes/Data",       // 这是系统数据卷，通常会和主卷重复计数
			"/private/var/vm",            // 虚拟内存
		}

		for _, ignoredMount := range ignoredMountPoints {
			if strings.HasPrefix(partition.Mountpoint, ignoredMount) {
				return true
			}
		}
	}

	// 在 Linux 上过滤掉特殊文件系统
	if runtime.GOOS == "linux" {
		ignoredFsTypes := []string{
			"tmpfs",
			"devtmpfs",
			"devfs",
			"proc",
			"sysfs",
			"cgroup",
			"cgroup2",
			"nsfs",
			"overlay",
			"squashfs",
			"iso9660",
		}

		for _, ignoredType := range ignoredFsTypes {
			if partition.Fstype == ignoredType {
				return true
			}
		}
	}

	// 过滤掉只读的 CD-ROM 等
	for _, opt := range partition.Opts {
		if strings.Contains(strings.ToLower(opt), "cdrom") {
			return true
		}
	}

	return false
}

// Collect 采集磁盘数据(合并静态和动态数据)
func (d *DiskCollector) Collect() ([]protocol.DiskData, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, err
	}
	var diskDataList []protocol.DiskData
	for _, partition := range partitions {
		// 跳过应该忽略的分区
		if shouldIgnorePartition(partition) {
			continue
		}

		// 获取动态使用情况
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			continue // 跳过无法访问的分区
		}

		// 跳过容量为 0 的分区（可能是虚拟文件系统）
		if usage.Total == 0 {
			continue
		}

		diskData := protocol.DiskData{
			MountPoint:   partition.Mountpoint,
			Device:       partition.Device,
			Fstype:       partition.Fstype,
			Total:        usage.Total,
			Used:         usage.Used,
			Free:         usage.Free,
			UsagePercent: usage.UsedPercent,
		}

		diskDataList = append(diskDataList, diskData)
	}

	return diskDataList, nil
}

// DiskIOCollector 磁盘 IO 监控采集器
type DiskIOCollector struct{}

// NewDiskIOCollector 创建磁盘 IO 采集器
func NewDiskIOCollector() *DiskIOCollector {
	return &DiskIOCollector{}
}

// Collect 采集磁盘 IO 数据(全部是动态数据,无需缓存)
func (d *DiskIOCollector) Collect() ([]*protocol.DiskIOData, error) {
	ioCounters, err := disk.IOCounters()
	if err != nil {
		return nil, err
	}

	var diskIODataList []*protocol.DiskIOData
	for device, counter := range ioCounters {
		diskIOData := &protocol.DiskIOData{
			Device:         device,
			ReadCount:      counter.ReadCount,
			WriteCount:     counter.WriteCount,
			ReadBytes:      counter.ReadBytes,
			WriteBytes:     counter.WriteBytes,
			ReadTime:       counter.ReadTime,
			WriteTime:      counter.WriteTime,
			IoTime:         counter.IoTime,
			IopsInProgress: counter.IopsInProgress,
		}

		diskIODataList = append(diskIODataList, diskIOData)
	}

	return diskIODataList, nil
}
