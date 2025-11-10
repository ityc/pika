package repo

import (
	"context"

	"github.com/dushixiang/pika/internal/models"
	"gorm.io/gorm"
)

type MetricRepo struct {
	db *gorm.DB
}

func NewMetricRepo(db *gorm.DB) *MetricRepo {
	return &MetricRepo{
		db: db,
	}
}

// SaveCPUMetric 保存CPU指标
func (r *MetricRepo) SaveCPUMetric(ctx context.Context, metric *models.CPUMetric) error {
	return r.db.WithContext(ctx).Create(metric).Error
}

// SaveMemoryMetric 保存内存指标
func (r *MetricRepo) SaveMemoryMetric(ctx context.Context, metric *models.MemoryMetric) error {
	return r.db.WithContext(ctx).Create(metric).Error
}

// SaveDiskMetric 保存磁盘指标
func (r *MetricRepo) SaveDiskMetric(ctx context.Context, metric *models.DiskMetric) error {
	return r.db.WithContext(ctx).Create(metric).Error
}

// SaveNetworkMetric 保存网络指标
func (r *MetricRepo) SaveNetworkMetric(ctx context.Context, metric *models.NetworkMetric) error {
	return r.db.WithContext(ctx).Create(metric).Error
}

// SaveLoadMetric 保存负载指标
func (r *MetricRepo) SaveLoadMetric(ctx context.Context, metric *models.LoadMetric) error {
	return r.db.WithContext(ctx).Create(metric).Error
}

// SaveDiskIOMetric 保存磁盘IO指标
func (r *MetricRepo) SaveDiskIOMetric(ctx context.Context, metric *models.DiskIOMetric) error {
	return r.db.WithContext(ctx).Create(metric).Error
}

// SaveGPUMetric 保存GPU指标
func (r *MetricRepo) SaveGPUMetric(ctx context.Context, metric *models.GPUMetric) error {
	return r.db.WithContext(ctx).Create(metric).Error
}

// SaveTemperatureMetric 保存温度指标
func (r *MetricRepo) SaveTemperatureMetric(ctx context.Context, metric *models.TemperatureMetric) error {
	return r.db.WithContext(ctx).Create(metric).Error
}

// SaveHostMetric 保存主机信息指标（只保留最新的一条记录）
func (r *MetricRepo) SaveHostMetric(ctx context.Context, metric *models.HostMetric) error {
	// 使用事务确保原子性
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先删除该 agent 的所有旧记录
		if err := tx.Where("agent_id = ?", metric.AgentID).Delete(&models.HostMetric{}).Error; err != nil {
			return err
		}
		// 插入新记录
		return tx.Create(metric).Error
	})
}

// GetLatestHostMetric 获取最新的主机信息
func (r *MetricRepo) GetLatestHostMetric(ctx context.Context, agentID string) (*models.HostMetric, error) {
	var metric models.HostMetric
	err := r.db.WithContext(ctx).
		Where("agent_id = ?", agentID).
		Order("timestamp DESC").
		First(&metric).Error
	if err != nil {
		return nil, err
	}
	return &metric, nil
}

// DeleteOldMetrics 删除指定时间之前的所有指标数据
func (r *MetricRepo) DeleteOldMetrics(ctx context.Context, beforeTimestamp int64) error {
	// 批量大小
	batchSize := 1000

	// 定义要清理的表（Host 信息只保留最新的，不需要清理）
	tables := []interface{}{
		&models.CPUMetric{},
		&models.MemoryMetric{},
		&models.DiskMetric{},
		&models.NetworkMetric{},
		&models.LoadMetric{},
		&models.DiskIOMetric{},
		&models.GPUMetric{},
		&models.TemperatureMetric{},
	}

	// 对每个表进行分批删除
	for _, table := range tables {
		for {
			// 分批删除，避免长事务
			result := r.db.WithContext(ctx).
				Where("timestamp < ?", beforeTimestamp).
				Limit(batchSize).
				Delete(table)

			if result.Error != nil {
				return result.Error
			}

			// 如果删除的行数少于批量大小，说明已经删除完毕
			if result.RowsAffected < int64(batchSize) {
				break
			}
		}
	}

	return nil
}

// AggregatedCPUMetric CPU聚合指标
type AggregatedCPUMetric struct {
	Timestamp    int64   `json:"timestamp"`
	AvgUsage     float64 `json:"avgUsage"`
	MinUsage     float64 `json:"minUsage"`
	MaxUsage     float64 `json:"maxUsage"`
	LogicalCores int     `json:"logicalCores"`
}

// GetCPUMetrics 获取聚合后的CPU指标（始终返回聚合数据）
// interval: 聚合间隔，单位秒（如：60表示1分钟，3600表示1小时）
func (r *MetricRepo) GetCPUMetrics(ctx context.Context, agentID string, start, end int64, interval int) ([]AggregatedCPUMetric, error) {
	var metrics []AggregatedCPUMetric

	query := `
		SELECT
			CAST(FLOOR(timestamp / ?) * ? AS BIGINT) as timestamp,
			AVG(usage_percent) as avg_usage,
			MIN(usage_percent) as min_usage,
			MAX(usage_percent) as max_usage,
			MAX(logical_cores) as logical_cores
		FROM cpu_metrics
		WHERE agent_id = ? AND timestamp >= ? AND timestamp <= ?
		GROUP BY 1
		ORDER BY timestamp ASC
	`

	intervalMs := int64(interval * 1000)
	err := r.db.WithContext(ctx).
		Raw(query, intervalMs, intervalMs, agentID, start, end).
		Scan(&metrics).Error

	return metrics, err
}

// AggregatedMemoryMetric 内存聚合指标
type AggregatedMemoryMetric struct {
	Timestamp int64   `json:"timestamp"`
	AvgUsage  float64 `json:"avgUsage"`
	MinUsage  float64 `json:"minUsage"`
	MaxUsage  float64 `json:"maxUsage"`
	Total     uint64  `json:"total"`
}

// GetMemoryMetrics 获取聚合后的内存指标（始终返回聚合数据）
func (r *MetricRepo) GetMemoryMetrics(ctx context.Context, agentID string, start, end int64, interval int) ([]AggregatedMemoryMetric, error) {
	var metrics []AggregatedMemoryMetric

	query := `
		SELECT
			CAST(FLOOR(timestamp / ?) * ? AS BIGINT) as timestamp,
			AVG(usage_percent) as avg_usage,
			MIN(usage_percent) as min_usage,
			MAX(usage_percent) as max_usage,
			MAX(total) as total
		FROM memory_metrics
		WHERE agent_id = ? AND timestamp >= ? AND timestamp <= ?
		GROUP BY 1
		ORDER BY timestamp ASC
	`

	intervalMs := int64(interval * 1000)
	err := r.db.WithContext(ctx).
		Raw(query, intervalMs, intervalMs, agentID, start, end).
		Scan(&metrics).Error

	return metrics, err
}

// AggregatedDiskMetric 磁盘聚合指标
type AggregatedDiskMetric struct {
	Timestamp  int64   `json:"timestamp"`
	MountPoint string  `json:"mountPoint"`
	AvgUsage   float64 `json:"avgUsage"`
	MinUsage   float64 `json:"minUsage"`
	MaxUsage   float64 `json:"maxUsage"`
	Total      uint64  `json:"total"`
}

// GetDiskMetrics 获取聚合后的磁盘指标（始终返回聚合数据）
func (r *MetricRepo) GetDiskMetrics(ctx context.Context, agentID string, start, end int64, interval int) ([]AggregatedDiskMetric, error) {
	var metrics []AggregatedDiskMetric

	query := `
		SELECT
			CAST(FLOOR(timestamp / ?) * ? AS BIGINT) as timestamp,
			mount_point,
			AVG(usage_percent) as avg_usage,
			MIN(usage_percent) as min_usage,
			MAX(usage_percent) as max_usage,
			MAX(total) as total
		FROM disk_metrics
		WHERE agent_id = ? AND timestamp >= ? AND timestamp <= ?
		GROUP BY 1, mount_point
		ORDER BY timestamp ASC, mount_point
	`

	intervalMs := int64(interval * 1000)
	err := r.db.WithContext(ctx).
		Raw(query, intervalMs, intervalMs, agentID, start, end).
		Scan(&metrics).Error

	return metrics, err
}

// AggregatedNetworkMetric 网络聚合指标
type AggregatedNetworkMetric struct {
	Timestamp   int64   `json:"timestamp"`
	AvgSentRate float64 `json:"avgSentRate"`
	AvgRecvRate float64 `json:"avgRecvRate"`
	MaxSentRate uint64  `json:"maxSentRate"`
	MaxRecvRate uint64  `json:"maxRecvRate"`
}

// GetNetworkMetrics 获取聚合后的网络指标（合并所有网卡接口）
func (r *MetricRepo) GetNetworkMetrics(ctx context.Context, agentID string, start, end int64, interval int) ([]AggregatedNetworkMetric, error) {
	var metrics []AggregatedNetworkMetric

	query := `
		SELECT
			CAST(FLOOR(timestamp / ?) * ? AS BIGINT) as timestamp,
			AVG(bytes_sent_rate) as avg_sent_rate,
			AVG(bytes_recv_rate) as avg_recv_rate,
			MAX(bytes_sent_rate) as max_sent_rate,
			MAX(bytes_recv_rate) as max_recv_rate
		FROM network_metrics
		WHERE agent_id = ? AND timestamp >= ? AND timestamp <= ?
		GROUP BY 1
		ORDER BY timestamp ASC
	`

	intervalMs := int64(interval * 1000)
	err := r.db.WithContext(ctx).
		Raw(query, intervalMs, intervalMs, agentID, start, end).
		Scan(&metrics).Error

	return metrics, err
}

// AggregatedLoadMetric 负载聚合指标
type AggregatedLoadMetric struct {
	Timestamp int64   `json:"timestamp"`
	AvgLoad1  float64 `json:"avgLoad1"`
	AvgLoad5  float64 `json:"avgLoad5"`
	AvgLoad15 float64 `json:"avgLoad15"`
	MaxLoad1  float64 `json:"maxLoad1"`
	MaxLoad5  float64 `json:"maxLoad5"`
	MaxLoad15 float64 `json:"maxLoad15"`
}

// GetLoadMetrics 获取聚合后的负载指标（始终返回聚合数据）
func (r *MetricRepo) GetLoadMetrics(ctx context.Context, agentID string, start, end int64, interval int) ([]AggregatedLoadMetric, error) {
	var metrics []AggregatedLoadMetric

	query := `
		SELECT
			CAST(FLOOR(timestamp / ?) * ? AS BIGINT) as timestamp,
			AVG(load1) as avg_load1,
			AVG(load5) as avg_load5,
			AVG(load15) as avg_load15,
			MAX(load1) as max_load1,
			MAX(load5) as max_load5,
			MAX(load15) as max_load15
		FROM load_metrics
		WHERE agent_id = ? AND timestamp >= ? AND timestamp <= ?
		GROUP BY 1
		ORDER BY timestamp ASC
	`

	intervalMs := int64(interval * 1000)
	err := r.db.WithContext(ctx).
		Raw(query, intervalMs, intervalMs, agentID, start, end).
		Scan(&metrics).Error

	return metrics, err
}

// GetLatestCPUMetric 获取最新的CPU指标
func (r *MetricRepo) GetLatestCPUMetric(ctx context.Context, agentID string) (*models.CPUMetric, error) {
	var metric models.CPUMetric
	err := r.db.WithContext(ctx).
		Where("agent_id = ?", agentID).
		Order("timestamp DESC").
		First(&metric).Error
	if err != nil {
		return nil, err
	}
	return &metric, nil
}

// GetLatestMemoryMetric 获取最新的内存指标
func (r *MetricRepo) GetLatestMemoryMetric(ctx context.Context, agentID string) (*models.MemoryMetric, error) {
	var metric models.MemoryMetric
	err := r.db.WithContext(ctx).
		Where("agent_id = ?", agentID).
		Order("timestamp DESC").
		First(&metric).Error
	if err != nil {
		return nil, err
	}
	return &metric, nil
}

// GetLatestDiskMetrics 获取最新的磁盘指标（所有挂载点）
func (r *MetricRepo) GetLatestDiskMetrics(ctx context.Context, agentID string) ([]models.DiskMetric, error) {
	// 先获取最新时间戳
	var latestTimestamp int64
	err := r.db.WithContext(ctx).
		Model(&models.DiskMetric{}).
		Where("agent_id = ?", agentID).
		Select("MAX(timestamp)").
		Scan(&latestTimestamp).Error

	if err != nil {
		return nil, err
	}

	// 获取该时间戳的所有磁盘数据
	var metrics []models.DiskMetric
	err = r.db.WithContext(ctx).
		Where("agent_id = ? AND timestamp = ?", agentID, latestTimestamp).
		Find(&metrics).Error

	return metrics, err
}

// GetLatestNetworkMetrics 获取最新的网络指标（所有网卡）
func (r *MetricRepo) GetLatestNetworkMetrics(ctx context.Context, agentID string) ([]models.NetworkMetric, error) {
	// 先获取最新时间戳
	var latestTimestamp int64
	err := r.db.WithContext(ctx).
		Model(&models.NetworkMetric{}).
		Where("agent_id = ?", agentID).
		Select("MAX(timestamp)").
		Scan(&latestTimestamp).Error

	if err != nil {
		return nil, err
	}

	// 获取该时间戳的所有网络数据
	var metrics []models.NetworkMetric
	err = r.db.WithContext(ctx).
		Where("agent_id = ? AND timestamp = ?", agentID, latestTimestamp).
		Find(&metrics).Error

	return metrics, err
}

// GetLatestLoadMetric 获取最新的负载指标
func (r *MetricRepo) GetLatestLoadMetric(ctx context.Context, agentID string) (*models.LoadMetric, error) {
	var metric models.LoadMetric
	err := r.db.WithContext(ctx).
		Where("agent_id = ?", agentID).
		Order("timestamp DESC").
		First(&metric).Error
	if err != nil {
		return nil, err
	}
	return &metric, nil
}
