import {type ReactNode, useEffect, useMemo, useState} from 'react';
import {useNavigate, useParams} from 'react-router-dom';
import {ArrowLeft, Cpu, Database, HardDrive, MemoryStick, Network, Server, Thermometer, Zap} from 'lucide-react';
import {
    Area,
    AreaChart,
    CartesianGrid,
    Legend,
    Line,
    LineChart,
    ResponsiveContainer,
    Tooltip,
    XAxis,
    YAxis,
} from 'recharts';
import {getAvailableNetworkInterfaces} from '@/api/agent.ts';
import {cn} from '@/lib/utils';
// 工具函数
import {formatBytes, formatDateTime, formatPercentValue, formatUptime} from '@/utils/util';
// Hooks
import {useAgentOverview, useAggregatedMetrics} from '@/hooks/server';
// 常量
import {INTERFACE_COLORS, TEMPERATURE_COLORS} from '@/constants/server';
import {SERVER_TIME_RANGE_OPTIONS} from '@/constants/time';
// 公共组件
import {
    Card,
    ChartPlaceholder,
    CustomTooltip,
    EmptyState,
    LoadingSpinner,
    TimeRangeSelector
} from '@/components/common';
// 服务器组件
import {InfoGrid, type SnapshotCardData, SnapshotSection} from '@/components/server';
import LittleStatCard from "@/components/common/LittleStatCard.tsx";


type AccentVariant = 'blue' | 'emerald' | 'purple' | 'amber';


const ServerDetail = () => {
    const {id} = useParams<{ id: string }>();
    const navigate = useNavigate();

    const [timeRange, setTimeRange] = useState<string>('15m');
    const [selectedInterface, setSelectedInterface] = useState<string>('all');
    const [selectedTempType, setSelectedTempType] = useState<string>('all');
    const {agent, latestMetrics, loading} = useAgentOverview(id);
    const metricsData = useAggregatedMetrics(id, timeRange, selectedInterface);

    const cpuChartData = useMemo(() => {
        // CPU 数据：取第一个系列（usage）
        const cpuSeries = metricsData.cpu.find(s => s.name === 'usage');
        if (!cpuSeries) return [];

        return cpuSeries.data.map((point) => ({
            time: new Date(point.timestamp).toLocaleTimeString('zh-CN', {
                hour: '2-digit',
                minute: '2-digit',
            }),
            usage: Number(point.value.toFixed(2)),
            timestamp: point.timestamp,
        }));
    }, [metricsData.cpu]);

    const memoryChartData = useMemo(() => {
        // Memory 数据：取第一个系列（usage）
        const memorySeries = metricsData.memory.find(s => s.name === 'usage');
        if (!memorySeries) return [];

        return memorySeries.data.map((point) => ({
            time: new Date(point.timestamp).toLocaleTimeString('zh-CN', {
                hour: '2-digit',
                minute: '2-digit',
            }),
            usage: Number(point.value.toFixed(2)),
            timestamp: point.timestamp,
        }));
    }, [metricsData.memory]);

    // 获取所有可用的网卡列表（从后端接口获取）
    const [availableInterfaces, setAvailableInterfaces] = useState<string[]>([]);

    useEffect(() => {
        if (!id) {
            setAvailableInterfaces([]);
            return;
        }

        let cancelled = false;

        const fetchInterfaces = async () => {
            try {
                const response = await getAvailableNetworkInterfaces(id);
                if (!cancelled) {
                    setAvailableInterfaces(response.data.interfaces || []);
                }
            } catch (error) {
                console.error('Failed to load network interfaces:', error);
                if (!cancelled) {
                    setAvailableInterfaces([]);
                }
            }
        };

        fetchInterfaces();

        return () => {
            cancelled = true;
        };
    }, [id]);

    // 当网卡列表变化时，如果当前选中的网卡不在列表中，重置为 'all'
    useEffect(() => {
        if (selectedInterface !== 'all' && availableInterfaces.length > 0) {
            if (!availableInterfaces.includes(selectedInterface)) {
                setSelectedInterface('all');
            }
        }
    }, [availableInterfaces, selectedInterface]);

    const networkChartData = useMemo(() => {
        if (metricsData.network.length === 0) return [];

        // 找到上行和下行系列
        const uploadSeries = metricsData.network.find(s => s.name === 'upload');
        const downloadSeries = metricsData.network.find(s => s.name === 'download');

        if (!uploadSeries || !downloadSeries) return [];

        // 按时间戳对齐数据
        const timeMap = new Map<number, any>();

        uploadSeries.data.forEach(point => {
            const time = new Date(point.timestamp).toLocaleTimeString('zh-CN', {
                hour: '2-digit',
                minute: '2-digit',
            });
            timeMap.set(point.timestamp, {
                time,
                timestamp: point.timestamp,
                upload: Number((point.value / 1024 / 1024).toFixed(2)), // 转换为 MB/s
            });
        });

        downloadSeries.data.forEach(point => {
            const existing = timeMap.get(point.timestamp);
            if (existing) {
                existing.download = Number((point.value / 1024 / 1024).toFixed(2));
            }
        });

        return Array.from(timeMap.values());
    }, [metricsData.network]);

    // Disk I/O 图表数据
    const diskIOChartData = useMemo(() => {
        if (metricsData.diskIO.length === 0) return [];

        // 找到读和写系列
        const readSeries = metricsData.diskIO.find(s => s.name === 'read');
        const writeSeries = metricsData.diskIO.find(s => s.name === 'write');

        if (!readSeries || !writeSeries) return [];

        // 按时间戳对齐数据
        const timeMap = new Map<number, any>();

        readSeries.data.forEach(point => {
            const time = new Date(point.timestamp).toLocaleTimeString('zh-CN', {
                hour: '2-digit',
                minute: '2-digit',
            });
            timeMap.set(point.timestamp, {
                time,
                timestamp: point.timestamp,
                read: Number((point.value / 1024 / 1024).toFixed(2)), // 转换为 MB/s
            });
        });

        writeSeries.data.forEach(point => {
            const existing = timeMap.get(point.timestamp);
            if (existing) {
                existing.write = Number((point.value / 1024 / 1024).toFixed(2));
            }
        });

        return Array.from(timeMap.values());
    }, [metricsData.diskIO]);

    // GPU 图表数据（暂不支持，需要后端返回温度系列）
    const gpuChartData = useMemo(() => {
        if (metricsData.gpu.length === 0) return [];

        // 按时间戳聚合利用率和温度系列
        const timeMap = new Map<number, any>();

        const utilizationSeries = metricsData.gpu.find(s => s.name === 'utilization');
        const temperatureSeries = metricsData.gpu.find(s => s.name === 'temperature');

        // 添加利用率数据
        utilizationSeries?.data.forEach(point => {
            const time = new Date(point.timestamp).toLocaleTimeString('zh-CN', {
                hour: '2-digit',
                minute: '2-digit',
            });
            timeMap.set(point.timestamp, {
                time,
                timestamp: point.timestamp,
                utilization: Number(point.value.toFixed(2)),
            });
        });

        // 添加温度数据
        temperatureSeries?.data.forEach(point => {
            const existing = timeMap.get(point.timestamp);
            if (existing) {
                existing.temperature = Number(point.value.toFixed(2));
            }
        });

        return Array.from(timeMap.values());
    }, [metricsData.gpu]);

    // Temperature 图表数据（按类型分组显示各类型的温度）
    const temperatureChartData = useMemo(() => {
        if (metricsData.temperature.length === 0) return [];

        // 按时间戳聚合所有温度系列
        const timeMap = new Map<number, any>();

        metricsData.temperature.forEach(series => {
            const sensorName = series.name; // 使用系列名称作为传感器标识
            series.data.forEach(point => {
                const time = new Date(point.timestamp).toLocaleTimeString('zh-CN', {
                    hour: '2-digit',
                    minute: '2-digit',
                });

                if (!timeMap.has(point.timestamp)) {
                    timeMap.set(point.timestamp, {time, timestamp: point.timestamp});
                }

                const existing = timeMap.get(point.timestamp)!;
                existing[sensorName] = Number(point.value.toFixed(2));
            });
        });

        return Array.from(timeMap.values());
    }, [metricsData.temperature]);

    // 提取所有唯一的温度类型（用于图表 Line 渲染和下拉选择器）
    const temperatureTypes = useMemo(() => {
        return metricsData.temperature.map(s => s.name).sort();
    }, [metricsData.temperature]);

    // 网络连接图表数据
    const networkConnectionChartData = useMemo(() => {
        if (metricsData.networkConnection.length === 0) return [];

        // 按时间戳聚合所有连接状态系列
        const timeMap = new Map<number, any>();

        metricsData.networkConnection.forEach(series => {
            const stateName = series.name; // established, time_wait, close_wait, listen
            series.data.forEach(point => {
                const time = new Date(point.timestamp).toLocaleTimeString('zh-CN', {
                    hour: '2-digit',
                    minute: '2-digit',
                });

                if (!timeMap.has(point.timestamp)) {
                    timeMap.set(point.timestamp, {time, timestamp: point.timestamp});
                }

                const existing = timeMap.get(point.timestamp)!;
                // 转换为驼峰命名以匹配图表的 dataKey
                const camelCaseName = stateName.replace(/_([a-z])/g, (_, letter) => letter.toUpperCase());
                existing[camelCaseName] = Number(point.value.toFixed(0));
            });
        });

        return Array.from(timeMap.values());
    }, [metricsData.networkConnection]);

    // 根据选中的类型过滤温度数据
    const filteredTemperatureTypes = useMemo(() => {
        if (selectedTempType === 'all') {
            return temperatureTypes;
        }
        return temperatureTypes.filter(type => type === selectedTempType);
    }, [temperatureTypes, selectedTempType]);

    // 当温度类型列表变化时，如果当前选中的类型不在列表中，重置为 'all'
    useEffect(() => {
        if (selectedTempType !== 'all' && temperatureTypes.length > 0) {
            if (!temperatureTypes.includes(selectedTempType)) {
                setSelectedTempType('all');
            }
        }
    }, [temperatureTypes, selectedTempType]);

    const snapshotCards: SnapshotCardData[] = useMemo(() => {
        if (!latestMetrics) {
            return [] as Array<{
                key: string;
                icon: typeof Cpu;
                title: string;
                usagePercent: string;
                accent: AccentVariant;
                metrics: Array<{ label: string; value: ReactNode }>;
            }>;
        }

        const cards: Array<{
            key: string;
            icon: typeof Cpu;
            title: string;
            usagePercent: string;
            accent: AccentVariant;
            metrics: Array<{ label: string; value: ReactNode }>;
        }> = [];

        cards.push({
            key: 'cpu',
            icon: Cpu,
            title: 'CPU 使用',
            usagePercent: `${formatPercentValue(latestMetrics.cpu?.usagePercent)}%`,
            accent: 'blue',
            metrics: [
                {label: '当前使用', value: `${formatPercentValue(latestMetrics.cpu?.usagePercent)}%`},
                {
                    label: '采样时间',
                    value: latestMetrics.cpu ? formatDateTime(latestMetrics.cpu.timestamp) : '-',
                },
            ],
        });

        cards.push({
            key: 'memory',
            icon: MemoryStick,
            title: '内存使用',
            usagePercent: `${formatPercentValue(latestMetrics.memory?.usagePercent)}%`,
            accent: 'emerald',
            metrics: [
                {
                    label: '已用 / 总量',
                    value: `${formatBytes(latestMetrics.memory?.used)} / ${formatBytes(latestMetrics.memory?.total)}`
                },
                {
                    label: 'Swap 已用',
                    value: `${formatBytes(latestMetrics.memory?.swapUsed)} / ${formatBytes(latestMetrics.memory?.swapTotal)}`
                },
            ],
        });

        cards.push({
            key: 'disk',
            icon: HardDrive,
            title: '磁盘使用',
            usagePercent: latestMetrics.disk
                ? `${formatPercentValue(latestMetrics.disk.usagePercent)}%`
                : '—',
            accent: 'purple',
            metrics: [
                {
                    label: '已用 / 总量',
                    value: `${formatBytes(latestMetrics.disk?.used)} / ${formatBytes(latestMetrics.disk?.total)}`
                },
                {label: '磁盘数量', value: latestMetrics.disk?.totalDisks ?? '-'},
            ],
        });

        cards.push({
            key: 'network',
            icon: Network,
            title: '网络流量',
            usagePercent: latestMetrics.network
                ? `${formatBytes(latestMetrics.network.totalBytesSentRate)}/s`
                : '—',
            accent: 'amber',
            metrics: [
                {
                    label: '上行 / 下行',
                    value: `${formatBytes(latestMetrics.network?.totalBytesSentRate)}/s ↑ / ${formatBytes(
                        latestMetrics.network?.totalBytesRecvRate,
                    )}/s ↓`,
                },
                {
                    label: '网络累计',
                    value: `${formatBytes(latestMetrics.network?.totalBytesSentTotal)} ↑ / ${formatBytes(
                        latestMetrics.network?.totalBytesRecvTotal,
                    )} ↓`,
                },
            ],
        });

        // 如果配置了流量限额，添加流量统计卡片
        if (agent && agent.trafficLimit && agent.trafficLimit > 0) {
            const trafficUsedPercent = ((agent.trafficUsed || 0) / agent.trafficLimit) * 100;
            const trafficRemaining = agent.trafficLimit - (agent.trafficUsed || 0);

            cards.push({
                key: 'traffic',
                icon: Database,
                title: '流量统计',
                usagePercent: `${formatPercentValue(trafficUsedPercent)}%`,
                accent: 'blue',
                metrics: [
                    {
                        label: '已用 / 总量',
                        value: `${formatBytes(agent.trafficUsed || 0)} / ${formatBytes(agent.trafficLimit)}`,
                    },
                    {
                        label: '剩余流量',
                        value: `${formatBytes(trafficRemaining > 0 ? trafficRemaining : 0)}`,
                    },
                    ...(agent.trafficResetDay && agent.trafficResetDay > 0
                        ? [{
                            label: '重置日期',
                            value: `每月${agent.trafficResetDay}号`,
                        }]
                        : []),
                ],
            });
        }

        return cards;
    }, [latestMetrics, agent]);

    const platformDisplay = latestMetrics?.host?.platform
        ? `${latestMetrics.host.platform} ${latestMetrics.host.platformVersion || ''}`.trim()
        : agent?.os || '-';
    const architectureDisplay = latestMetrics?.host?.kernelArch || agent?.arch || '-';
    const uptimeDisplay = formatUptime(latestMetrics?.host?.uptime);
    const bootTimeDisplay = latestMetrics?.host?.bootTime
        ? formatDateTime(latestMetrics.host.bootTime * 1000)
        : '-';
    const lastSeenDisplay = agent ? formatDateTime(agent.lastSeenAt) : '-';
    const displayName = agent?.name?.trim() ? agent.name : '未命名探针';
    const isOnline = agent?.status === 1;
    const statusDotStyles = isOnline ? 'bg-emerald-500' : 'bg-rose-500';
    const statusText = isOnline ? '在线' : '离线';

    const networkSummary = latestMetrics?.network
        ? `${formatBytes(latestMetrics.network.totalBytesSentTotal)} ↑ / ${formatBytes(
            latestMetrics.network.totalBytesRecvTotal,
        )} ↓`
        : '—';

    const environmentInfo = [
        {label: '操作系统', value: platformDisplay || '-'},
        {label: '内核版本', value: latestMetrics?.host?.kernelVersion || '-'},
        {label: '硬件架构', value: architectureDisplay || '-'},
        {label: 'CPU 型号', value: latestMetrics?.cpu?.modelName || '-'},
        {label: '逻辑核心', value: latestMetrics?.cpu?.logicalCores ?? '-'},
        {label: '物理核心', value: latestMetrics?.cpu?.physicalCores ?? '-'},
    ];

    const statusInfo = [
        {label: '启动时间', value: bootTimeDisplay},
        {label: '运行时间', value: uptimeDisplay},
        {label: '最近心跳', value: lastSeenDisplay},
        {label: '进程数', value: latestMetrics?.host?.procs ?? '-'},
        {label: '网络累计', value: networkSummary},
    ];

    const heroStats = [
        {label: '运行系统', value: platformDisplay || '-'},
        {label: '硬件架构', value: architectureDisplay || '-'},
        {label: '最近心跳', value: lastSeenDisplay},
        {label: '运行时长', value: uptimeDisplay},
    ];

    if (loading) {
        return <LoadingSpinner variant="dark"/>;
    }

    if (!agent) {
        return <EmptyState variant="dark"/>;
    }

    return (
        <div className="bg-[#05050a] min-h-screen">
            <div className="mx-auto flex max-w-7xl flex-col px-4 pb-10 pt-6 sm:px-6 lg:px-8">
                <section
                    className="rounded-2xl border border-cyan-900/50 bg-[#0a0b10]/90 p-6 shadow-2xl backdrop-blur-sm">
                    <div className="flex flex-col gap-6">
                        <div className="flex flex-col gap-6 lg:flex-row lg:items-center lg:justify-between">
                            <div className="space-y-4">
                                <button
                                    type="button"
                                    onClick={() => navigate('/')}
                                    className="group inline-flex items-center gap-2 text-xs font-bold font-mono uppercase tracking-[0.3em] text-cyan-600 transition hover:text-cyan-400"
                                >
                                    <ArrowLeft className="h-4 w-4 transition group-hover:-translate-x-0.5"/>
                                    返回概览
                                </button>
                                <div className="flex items-start gap-4">
                                    <div
                                        className="flex h-14 w-14 items-center justify-center rounded-xl bg-cyan-500/10 border border-cyan-500/30 text-cyan-400">
                                        <Server className="h-7 w-7"/>
                                    </div>
                                    <div>
                                        <div className="flex flex-wrap items-center gap-3">
                                            <h1 className="text-3xl font-bold text-cyan-100">{displayName}</h1>
                                            <span
                                                className={cn(
                                                    "inline-flex items-center gap-1 rounded-full px-3 py-0.5 text-xs font-bold font-mono uppercase tracking-wider",
                                                    isOnline
                                                        ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/50'
                                                        : 'bg-rose-500/20 text-rose-400 border border-rose-500/50'
                                                )}
                                            >
                                                <span className={cn("h-1.5 w-1.5 rounded-full", statusDotStyles)}/>
                                                {statusText}
                                            </span>
                                        </div>
                                        <p className="mt-2 text-sm text-cyan-600 font-mono">
                                            {[agent.hostname, agent.ip].filter(Boolean).join(' · ') || '-'}
                                        </p>
                                    </div>
                                </div>
                            </div>

                            <div className="grid w-full gap-3 sm:grid-cols-2 lg:w-auto lg:grid-cols-2 xl:grid-cols-4">
                                {heroStats.map((stat) => (
                                    <LittleStatCard label={stat.label} value={stat.value}/>
                                ))}
                            </div>
                        </div>
                        <div
                            className="flex flex-wrap items-center gap-3 text-xs text-cyan-600 font-mono pt-4 border-t border-cyan-900/30">
                            <span>探针 ID：{agent.id}</span>
                            <span className="hidden h-1 w-1 rounded-full bg-cyan-900 sm:inline-block"/>
                            <span>版本：{agent.version || '-'}</span>
                            <span className="hidden h-1 w-1 rounded-full bg-cyan-900 sm:inline-block"/>
                            <span>网络累计：{networkSummary}</span>
                        </div>
                    </div>
                </section>

                <main className="flex-1 py-10 space-y-10">
                    <Card title="系统信息" description="探针基础属性、运行状态与资源概览" variant="dark">
                        <div className="space-y-6">
                            <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
                                <div
                                    className="rounded-xl border border-cyan-900/50 bg-black/30 p-4 backdrop-blur-sm">
                                    <h3 className="text-xs font-bold font-mono uppercase tracking-widest text-cyan-400">运行环境</h3>
                                    <p className="mt-1 text-[10px] text-cyan-700">来自最近一次探针上报的硬件与系统信息</p>
                                    <div className="mt-4">
                                        <InfoGrid items={environmentInfo}/>
                                    </div>
                                </div>
                                <div
                                    className="rounded-xl border border-cyan-900/50 bg-black/30 p-4 backdrop-blur-sm">
                                    <h3 className="text-xs font-bold font-mono uppercase tracking-widest text-cyan-400">运行状态</h3>
                                    <p className="mt-1 text-[10px] text-cyan-700">关键时间与网络指标，帮助快速判断主机健康状况</p>
                                    <div className="mt-4">
                                        <InfoGrid items={statusInfo}/>
                                    </div>
                                </div>
                            </div>
                            {latestMetrics?.networkConnection && (
                                <div
                                    className="rounded-xl border border-cyan-900/50 bg-black/30 p-4 backdrop-blur-sm">
                                    <h3 className="text-xs font-bold font-mono uppercase tracking-widest text-cyan-400">网络连接统计</h3>
                                    <p className="mt-1 text-[10px] text-cyan-700">TCP 连接各状态的实时统计数据</p>
                                    <div className="mt-4 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
                                        <div className="text-center">
                                            <div
                                                className="text-[10px] text-cyan-600 font-mono uppercase tracking-wider">Total
                                            </div>
                                            <div
                                                className="mt-1 text-lg font-semibold text-cyan-100">{latestMetrics.networkConnection.total}</div>
                                        </div>
                                        <div className="text-center">
                                            <div
                                                className="text-[10px] text-cyan-600 font-mono uppercase tracking-wider">ESTABLISHED
                                            </div>
                                            <div
                                                className="mt-1 text-lg font-semibold text-emerald-400">{latestMetrics.networkConnection.established}</div>
                                        </div>
                                        <div className="text-center">
                                            <div
                                                className="text-[10px] text-cyan-600 font-mono uppercase tracking-wider">TIME_WAIT
                                            </div>
                                            <div
                                                className="mt-1 text-lg font-semibold text-amber-400">{latestMetrics.networkConnection.timeWait}</div>
                                        </div>
                                        <div className="text-center">
                                            <div
                                                className="text-[10px] text-cyan-600 font-mono uppercase tracking-wider">LISTEN
                                            </div>
                                            <div
                                                className="mt-1 text-lg font-semibold text-blue-400">{latestMetrics.networkConnection.listen}</div>
                                        </div>
                                        <div className="text-center">
                                            <div
                                                className="text-[10px] text-cyan-600 font-mono uppercase tracking-wider">CLOSE_WAIT
                                            </div>
                                            <div
                                                className="mt-1 text-lg font-semibold text-rose-400">{latestMetrics.networkConnection.closeWait}</div>
                                        </div>
                                        <div className="text-center">
                                            <div
                                                className="text-[10px] text-cyan-600 font-mono uppercase tracking-wider">OTHER
                                            </div>
                                            <div
                                                className="mt-1 text-lg font-semibold text-cyan-500">
                                                {latestMetrics.networkConnection.synSent +
                                                    latestMetrics.networkConnection.synRecv +
                                                    latestMetrics.networkConnection.finWait1 +
                                                    latestMetrics.networkConnection.finWait2 +
                                                    latestMetrics.networkConnection.close +
                                                    latestMetrics.networkConnection.lastAck +
                                                    latestMetrics.networkConnection.closing}
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            )}
                            <SnapshotSection cards={snapshotCards}/>
                        </div>
                    </Card>

                    <Card
                        title="历史趋势"
                        description="针对选定时间范围展示 CPU、内存与网络的变化趋势"
                        variant="dark"
                        action={<TimeRangeSelector value={timeRange} onChange={setTimeRange}
                                                   options={SERVER_TIME_RANGE_OPTIONS} variant="dark"/>}
                    >
                        <div className="grid gap-6 md:grid-cols-2">
                            <section>
                                <h3 className="mb-3 flex items-center gap-2 text-xs font-bold font-mono uppercase tracking-widest text-cyan-400">
                                    <span
                                        className="flex h-8 w-8 items-center justify-center rounded-lg bg-cyan-500/10 text-cyan-400">
                                        <Cpu className="h-4 w-4"/>
                                    </span>
                                    CPU 使用率
                                </h3>
                                {cpuChartData.length > 0 ? (
                                    <ResponsiveContainer width="100%" height={220}>
                                        <AreaChart data={cpuChartData}>
                                            <defs>
                                                <linearGradient id="cpuAreaGradient" x1="0" y1="0" x2="0" y2="1">
                                                    <stop offset="5%" stopColor="#2563eb" stopOpacity={0.4}/>
                                                    <stop offset="95%" stopColor="#2563eb" stopOpacity={0}/>
                                                </linearGradient>
                                            </defs>
                                            <CartesianGrid stroke="currentColor" strokeDasharray="4 4"
                                                           className="stroke-cyan-900/30"/>
                                            <XAxis
                                                dataKey="time"
                                                stroke="currentColor"
                                                className="stroke-cyan-600"
                                                style={{fontSize: '12px'}}
                                            />
                                            <YAxis
                                                domain={[0, 100]}
                                                stroke="currentColor"
                                                className="stroke-cyan-600"
                                                style={{fontSize: '12px'}}
                                                tickFormatter={(value) => `${value}%`}
                                            />
                                            <Tooltip content={<CustomTooltip unit="%" variant="dark"/>}/>
                                            <Area
                                                type="monotone"
                                                dataKey="usage"
                                                name="CPU 使用率"
                                                stroke="#2563eb"
                                                strokeWidth={2}
                                                fill="url(#cpuAreaGradient)"
                                                activeDot={{r: 3}}
                                            />
                                        </AreaChart>
                                    </ResponsiveContainer>
                                ) : (
                                    <ChartPlaceholder variant="dark"/>
                                )}
                            </section>

                            <section>
                                <h3 className="mb-3 flex items-center gap-2 text-xs font-bold font-mono uppercase tracking-widest text-cyan-400">
                                    <span
                                        className="flex h-8 w-8 items-center justify-center rounded-lg bg-cyan-500/10 text-cyan-400">
                                        <MemoryStick className="h-4 w-4"/>
                                    </span>
                                    内存使用率
                                </h3>
                                {memoryChartData.length > 0 ? (
                                    <ResponsiveContainer width="100%" height={220}>
                                        <AreaChart data={memoryChartData}>
                                            <defs>
                                                <linearGradient id="memoryAreaGradient" x1="0" y1="0" x2="0" y2="1">
                                                    <stop offset="5%" stopColor="#10b981" stopOpacity={0.4}/>
                                                    <stop offset="95%" stopColor="#10b981" stopOpacity={0}/>
                                                </linearGradient>
                                            </defs>
                                            <CartesianGrid stroke="currentColor" strokeDasharray="4 4"
                                                           className="stroke-cyan-900/30"/>
                                            <XAxis
                                                dataKey="time"
                                                stroke="currentColor"
                                                className="stroke-cyan-600"
                                                style={{fontSize: '12px'}}
                                            />
                                            <YAxis
                                                domain={[0, 100]}
                                                stroke="currentColor"
                                                className="stroke-cyan-600"
                                                style={{fontSize: '12px'}}
                                                tickFormatter={(value) => `${value}%`}
                                            />
                                            <Tooltip content={<CustomTooltip unit="%" variant="dark"/>}/>
                                            <Area
                                                type="monotone"
                                                dataKey="usage"
                                                name="内存使用率"
                                                stroke="#10b981"
                                                strokeWidth={2}
                                                fill="url(#memoryAreaGradient)"
                                                activeDot={{r: 3}}
                                            />
                                        </AreaChart>
                                    </ResponsiveContainer>
                                ) : (
                                    <ChartPlaceholder variant="dark"/>
                                )}
                            </section>

                            <section>
                                <div className="mb-3 flex items-center justify-between">
                                    <h3 className="flex items-center gap-2 text-xs font-bold font-mono uppercase tracking-widest text-cyan-400">
                                        <span
                                            className="flex h-8 w-8 items-center justify-center rounded-lg bg-cyan-500/10 text-cyan-400">
                                            <Network className="h-4 w-4"/>
                                        </span>
                                        网络流量（MB/s）
                                    </h3>
                                    {availableInterfaces.length > 0 && (
                                        <select
                                            value={selectedInterface}
                                            onChange={(e) => setSelectedInterface(e.target.value)}
                                            className="rounded-lg border border-cyan-900/50 bg-black/40 px-3 py-1.5 text-xs font-mono text-cyan-300 hover:border-cyan-700 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/20"
                                        >
                                            {/*<option value="all">所有网卡（聚合）</option>*/}
                                            {availableInterfaces.map((iface) => (
                                                <option key={iface} value={iface}>
                                                    {iface}
                                                </option>
                                            ))}
                                        </select>
                                    )}
                                </div>
                                {networkChartData.length > 0 ? (
                                    <ResponsiveContainer width="100%" height={220}>
                                        <AreaChart data={networkChartData}>
                                            <defs>
                                                <linearGradient id="color-upload" x1="0" y1="0" x2="0" y2="1">
                                                    <stop offset="5%" stopColor={INTERFACE_COLORS[0].upload}
                                                          stopOpacity={0.3}/>
                                                    <stop offset="95%" stopColor={INTERFACE_COLORS[0].upload}
                                                          stopOpacity={0}/>
                                                </linearGradient>
                                                <linearGradient id="color-download" x1="0" y1="0" x2="0" y2="1">
                                                    <stop offset="5%" stopColor={INTERFACE_COLORS[0].download}
                                                          stopOpacity={0.3}/>
                                                    <stop offset="95%" stopColor={INTERFACE_COLORS[0].download}
                                                          stopOpacity={0}/>
                                                </linearGradient>
                                            </defs>
                                            <CartesianGrid stroke="currentColor" strokeDasharray="4 4"
                                                           className="stroke-cyan-900/30"/>
                                            <XAxis
                                                dataKey="time"
                                                stroke="currentColor"
                                                className="stroke-cyan-600"
                                                style={{fontSize: '12px'}}
                                            />
                                            <YAxis
                                                stroke="currentColor"
                                                className="stroke-cyan-600"
                                                style={{fontSize: '12px'}}
                                                tickFormatter={(value) => `${value} MB`}
                                            />
                                            <Tooltip content={<CustomTooltip unit=" MB/s" variant="dark"/>}/>
                                            <Legend/>
                                            {/* 渲染上行和下行区域 */}
                                            <Area
                                                type="monotone"
                                                dataKey="upload"
                                                name="上行"
                                                stroke={INTERFACE_COLORS[0].upload}
                                                strokeWidth={2}
                                                fill="url(#color-upload)"
                                                activeDot={{r: 3}}
                                            />
                                            <Area
                                                type="monotone"
                                                dataKey="download"
                                                name="下行"
                                                stroke={INTERFACE_COLORS[0].download}
                                                strokeWidth={2}
                                                fill="url(#color-download)"
                                                activeDot={{r: 3}}
                                            />
                                        </AreaChart>
                                    </ResponsiveContainer>
                                ) : (
                                    <ChartPlaceholder subtitle="稍后再次尝试刷新网络流量" variant="dark"/>
                                )}
                            </section>

                            <section>
                                <h3 className="mb-3 flex items-center gap-2 text-xs font-bold font-mono uppercase tracking-widest text-cyan-400">
                                    <span
                                        className="flex h-8 w-8 items-center justify-center rounded-lg bg-cyan-500/10 text-cyan-400">
                                        <HardDrive className="h-4 w-4"/>
                                    </span>
                                    磁盘 I/O (MB/s)
                                </h3>
                                {diskIOChartData.length > 0 ? (
                                    <ResponsiveContainer width="100%" height={220}>
                                        <AreaChart data={diskIOChartData}>
                                            <defs>
                                                <linearGradient id="colorDiskRead" x1="0" y1="0" x2="0" y2="1">
                                                    <stop offset="5%" stopColor="#2C70F6" stopOpacity={0.3}/>
                                                    <stop offset="95%" stopColor="#2C70F6" stopOpacity={0}/>
                                                </linearGradient>
                                                <linearGradient id="colorDiskWrite" x1="0" y1="0" x2="0" y2="1">
                                                    <stop offset="5%" stopColor="#6FD598" stopOpacity={0.3}/>
                                                    <stop offset="95%" stopColor="#6FD598" stopOpacity={0}/>
                                                </linearGradient>
                                            </defs>
                                            <CartesianGrid stroke="currentColor" strokeDasharray="4 4"
                                                           className="stroke-cyan-900/30"/>
                                            <XAxis
                                                dataKey="time"
                                                stroke="currentColor"
                                                className="stroke-cyan-600"
                                                style={{fontSize: '12px'}}
                                            />
                                            <YAxis
                                                stroke="currentColor"
                                                className="stroke-cyan-600"
                                                style={{fontSize: '12px'}}
                                                tickFormatter={(value) => `${value} MB`}
                                            />
                                            <Tooltip content={<CustomTooltip unit=" MB" variant="dark"/>}/>
                                            <Legend/>
                                            <Area
                                                type="monotone"
                                                dataKey="read"
                                                name="读取"
                                                stroke="#2C70F6"
                                                strokeWidth={2}
                                                fill="url(#colorDiskRead)"
                                                activeDot={{r: 3}}
                                            />
                                            <Area
                                                type="monotone"
                                                dataKey="write"
                                                name="写入"
                                                stroke="#6FD598"
                                                strokeWidth={2}
                                                fill="url(#colorDiskWrite)"
                                                activeDot={{r: 3}}
                                            />
                                        </AreaChart>
                                    </ResponsiveContainer>
                                ) : (
                                    <ChartPlaceholder subtitle="暂无磁盘 I/O 采集数据" variant="dark"/>
                                )}
                            </section>

                            <section>
                                <h3 className="mb-3 flex items-center gap-2 text-xs font-bold font-mono uppercase tracking-widest text-cyan-400">
                                    <span
                                        className="flex h-8 w-8 items-center justify-center rounded-lg bg-cyan-500/10 text-cyan-400">
                                        <Network className="h-4 w-4"/>
                                    </span>
                                    网络连接统计
                                </h3>
                                {networkConnectionChartData.length > 0 ? (
                                    <ResponsiveContainer width="100%" height={220}>
                                        <LineChart data={networkConnectionChartData}>
                                            <CartesianGrid stroke="currentColor" strokeDasharray="4 4"
                                                           className="stroke-cyan-900/30"/>
                                            <XAxis
                                                dataKey="time"
                                                stroke="currentColor"
                                                className="stroke-cyan-600"
                                                style={{fontSize: '12px'}}
                                            />
                                            <YAxis
                                                stroke="currentColor"
                                                className="stroke-cyan-600"
                                                style={{fontSize: '12px'}}
                                            />
                                            <Tooltip content={<CustomTooltip unit="" variant="dark"/>}/>
                                            <Legend/>
                                            <Line
                                                type="monotone"
                                                dataKey="established"
                                                name="ESTABLISHED"
                                                stroke="#10b981"
                                                strokeWidth={2}
                                                dot={false}
                                                activeDot={{r: 3}}
                                            />
                                            <Line
                                                type="monotone"
                                                dataKey="timeWait"
                                                name="TIME_WAIT"
                                                stroke="#f59e0b"
                                                strokeWidth={2}
                                                dot={false}
                                                activeDot={{r: 3}}
                                            />
                                            <Line
                                                type="monotone"
                                                dataKey="closeWait"
                                                name="CLOSE_WAIT"
                                                stroke="#ef4444"
                                                strokeWidth={2}
                                                dot={false}
                                                activeDot={{r: 3}}
                                            />
                                            <Line
                                                type="monotone"
                                                dataKey="listen"
                                                name="LISTEN"
                                                stroke="#3b82f6"
                                                strokeWidth={2}
                                                dot={false}
                                                activeDot={{r: 3}}
                                            />
                                        </LineChart>
                                    </ResponsiveContainer>
                                ) : (
                                    <ChartPlaceholder subtitle="暂无网络连接统计数据" variant="dark"/>
                                )}
                            </section>

                            {gpuChartData.length > 0 && (
                                <section>
                                    <h3 className="mb-3 flex items-center gap-2 text-xs font-bold font-mono uppercase tracking-widest text-cyan-400">
                                        <span
                                            className="flex h-8 w-8 items-center justify-center rounded-lg bg-cyan-500/10 text-cyan-400">
                                            <Zap className="h-4 w-4"/>
                                        </span>
                                        GPU 使用率与温度
                                    </h3>
                                    <ResponsiveContainer width="100%" height={220}>
                                        <LineChart data={gpuChartData}>
                                            <CartesianGrid stroke="currentColor" strokeDasharray="4 4"
                                                           className="stroke-cyan-900/30"/>
                                            <XAxis
                                                dataKey="time"
                                                stroke="currentColor"
                                                className="stroke-cyan-600"
                                                style={{fontSize: '12px'}}
                                            />
                                            <YAxis
                                                yAxisId="left"
                                                stroke="currentColor"
                                                className="stroke-cyan-600"
                                                style={{fontSize: '12px'}}
                                                tickFormatter={(value) => `${value}%`}
                                            />
                                            <YAxis
                                                yAxisId="right"
                                                orientation="right"
                                                stroke="currentColor"
                                                className="stroke-cyan-600"
                                                style={{fontSize: '12px'}}
                                                tickFormatter={(value) => `${value}°C`}
                                            />
                                            <Tooltip content={<CustomTooltip unit="" variant="dark"/>}/>
                                            <Legend/>
                                            <Line
                                                yAxisId="left"
                                                type="monotone"
                                                dataKey="utilization"
                                                name="使用率 (%)"
                                                stroke="#7c3aed"
                                                strokeWidth={2}
                                                dot={false}
                                                activeDot={{r: 3}}
                                            />
                                            <Line
                                                yAxisId="right"
                                                type="monotone"
                                                dataKey="temperature"
                                                name="温度 (°C)"
                                                stroke="#f97316"
                                                strokeWidth={2}
                                                dot={false}
                                                activeDot={{r: 3}}
                                            />
                                        </LineChart>
                                    </ResponsiveContainer>
                                </section>
                            )}

                            {temperatureChartData.length > 0 && temperatureTypes.length > 0 && (
                                <section>
                                    <div className="mb-3 flex items-center justify-between">
                                        <h3 className="flex items-center gap-2 text-xs font-bold font-mono uppercase tracking-widest text-cyan-400">
                                            <span
                                                className="flex h-8 w-8 items-center justify-center rounded-lg bg-cyan-500/10 text-cyan-400">
                                                <Thermometer className="h-4 w-4"/>
                                            </span>
                                            系统温度
                                        </h3>
                                        {temperatureTypes.length > 1 && (
                                            <select
                                                value={selectedTempType}
                                                onChange={(e) => setSelectedTempType(e.target.value)}
                                                className="rounded-lg border border-cyan-900/50 bg-black/40 px-3 py-1.5 text-xs font-mono text-cyan-300 hover:border-cyan-700 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/20"
                                            >
                                                <option value="all">所有类型</option>
                                                {temperatureTypes.map((type) => (
                                                    <option key={type} value={type}>
                                                        {type}
                                                    </option>
                                                ))}
                                            </select>
                                        )}
                                    </div>
                                    <ResponsiveContainer width="100%" height={220}>
                                        <LineChart data={temperatureChartData}>
                                            <CartesianGrid stroke="currentColor" strokeDasharray="4 4"
                                                           className="stroke-slate-200 dark:stroke-slate-600"/>
                                            <XAxis
                                                dataKey="time"
                                                stroke="currentColor"
                                                className="stroke-slate-400 dark:stroke-slate-500"
                                                style={{fontSize: '12px'}}
                                            />
                                            <YAxis
                                                stroke="currentColor"
                                                className="stroke-slate-400 dark:stroke-slate-500"
                                                style={{fontSize: '12px'}}
                                                tickFormatter={(value) => `${value}°C`}
                                            />
                                            <Tooltip content={<CustomTooltip unit="°C" variant="dark"/>}/>
                                            <Legend/>
                                            {/* 为选中的温度类型渲染线条 */}
                                            {filteredTemperatureTypes.map((type, index) => {
                                                // 使用预定义颜色，如果没有则使用默认颜色
                                                const color = TEMPERATURE_COLORS[type] || `hsl(${(index * 60) % 360}, 70%, 50%)`;
                                                return (
                                                    <Line
                                                        key={type}
                                                        type="monotone"
                                                        dataKey={type}
                                                        name={type}
                                                        stroke={color}
                                                        strokeWidth={2}
                                                        dot={false}
                                                        activeDot={{r: 3}}
                                                        connectNulls
                                                    />
                                                );
                                            })}
                                        </LineChart>
                                    </ResponsiveContainer>
                                </section>
                            )}
                        </div>
                    </Card>


                    {/* GPU 监控 */}
                    {latestMetrics?.gpu && latestMetrics.gpu.length > 0 && (
                        <Card title="GPU 监控" description="显卡使用情况和温度监控" variant="dark">
                            <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
                                {latestMetrics.gpu.map((gpu) => (
                                    <div
                                        key={gpu.index}
                                        className="rounded-xl border border-cyan-900/50 bg-black/30 p-4 backdrop-blur-sm hover:border-cyan-700/50 transition"
                                    >
                                        <div className="mb-3 flex items-center justify-between">
                                            <div className="flex items-center gap-2">
                                                <span
                                                    className="flex h-9 w-9 items-center justify-center rounded-lg bg-cyan-500/10 text-cyan-400">
                                                    <Zap className="h-4 w-4"/>
                                                </span>
                                                <div>
                                                    <p className="text-sm font-bold font-mono text-cyan-100">GPU {gpu.index}</p>
                                                    <p className="text-[10px] text-cyan-600">{gpu.name}</p>
                                                </div>
                                            </div>
                                            <span className="text-2xl font-bold text-purple-400">
                                                {gpu.utilization.toFixed(1)}%
                                            </span>
                                        </div>
                                        <div className="space-y-2 text-xs">
                                            <div className="flex items-center justify-between">
                                                <span
                                                    className="text-cyan-600 font-mono text-[10px] uppercase tracking-wider">温度</span>
                                                <span
                                                    className="font-medium text-cyan-200">{gpu.temperature.toFixed(1)}°C</span>
                                            </div>
                                            <div className="flex items-center justify-between">
                                                <span
                                                    className="text-cyan-600 font-mono text-[10px] uppercase tracking-wider">显存</span>
                                                <span className="font-medium text-cyan-200">
                                                    {formatBytes(gpu.memoryUsed)} / {formatBytes(gpu.memoryTotal)}
                                                </span>
                                            </div>
                                            <div className="flex items-center justify-between">
                                                <span
                                                    className="text-cyan-600 font-mono text-[10px] uppercase tracking-wider">功耗</span>
                                                <span
                                                    className="font-medium text-cyan-200">{gpu.powerDraw.toFixed(1)}W</span>
                                            </div>
                                            <div className="flex items-center justify-between">
                                                <span
                                                    className="text-cyan-600 font-mono text-[10px] uppercase tracking-wider">风扇转速</span>
                                                <span
                                                    className="font-medium text-cyan-200">{gpu.fanSpeed.toFixed(0)}%</span>
                                            </div>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        </Card>
                    )}

                    {/* 温度监控 */}
                    {latestMetrics?.temperature && latestMetrics.temperature.length > 0 && (
                        <Card title="温度监控" description="系统各部件温度传感器数据" variant="dark">
                            <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
                                {latestMetrics.temperature.sort((a, b) => a.sensorKey.localeCompare(b.sensorKey)).map((temp) => (
                                    <div
                                        key={temp.sensorKey}
                                        className="rounded-xl border border-cyan-900/50 bg-black/30 p-4 backdrop-blur-sm hover:border-cyan-700/50 transition"
                                    >
                                        <div className="flex items-center gap-2 mb-2">
                                            <Thermometer className="h-4 w-4 text-cyan-600"/>
                                            <p className="text-[10px] font-bold font-mono uppercase tracking-wider text-cyan-400 truncate">{temp.sensorLabel}</p>
                                        </div>
                                        <p className="text-2xl font-bold text-orange-400">{temp.temperature.toFixed(1)}°C</p>
                                    </div>
                                ))}
                            </div>
                        </Card>
                    )}
                </main>
            </div>
        </div>
    );
};

export default ServerDetail;
