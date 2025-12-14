import {useEffect, useMemo, useState} from 'react';
import {useNavigate, useParams} from 'react-router-dom';
import {useQuery} from '@tanstack/react-query';
import {AlertCircle, ArrowLeft, Clock, Loader2, MapPin, Shield} from 'lucide-react';
import {Area, AreaChart, CartesianGrid, Legend, ResponsiveContainer, Tooltip, XAxis, YAxis} from 'recharts';
import {type GetMetricsResponse, getMonitorAgentStats, getMonitorHistory, getMonitorStatsById} from '@/api/monitor.ts';
import type {AgentMonitorStat, PublicMonitor} from '@/types';
import {formatDateTime, formatTime} from '@/utils/util.ts';
import {StatusBadge} from '@/components/monitor/StatusBadge';
import {TypeIcon} from '@/components/monitor/TypeIcon';
import {CertBadge} from '@/components/monitor/CertBadge';
import {cn} from '@/lib/utils';
// 常量
import {AGENT_COLORS} from '@/constants/colors';
import {MONITOR_TIME_RANGE_OPTIONS} from '@/constants/time';
import LittleStatCard from "@/components/common/LittleStatCard.tsx";

// 加载状态组件
const LoadingSpinner = () => (
    <div className="flex min-h-[400px] w-full items-center justify-center">
        <div className="flex flex-col items-center gap-3 text-cyan-600">
            <Loader2 className="h-8 w-8 animate-spin text-cyan-400"/>
            <span className="text-sm font-mono">加载监控详情中...</span>
        </div>
    </div>
);

// 空状态组件
const EmptyState = () => (
    <div className="flex min-h-[400px] flex-col items-center justify-center text-cyan-500">
        <Shield className="mb-4 h-16 w-16 opacity-20"/>
        <p className="text-lg font-medium font-mono">未找到监控数据</p>
        <p className="mt-2 text-sm text-cyan-600">请检查监控 ID 是否正确</p>
    </div>
);

// 卡片容器组件
const Card = ({
                  title,
                  description,
                  action,
                  children
              }: {
    title: string;
    description?: string;
    action?: React.ReactNode;
    children: React.ReactNode;
}) => (
    <div className="group bg-[#0f1016]/80 backdrop-blur-md border border-cyan-500/20 shadow-[0_0_15px_rgba(6,182,212,0.05)] hover:border-cyan-500/30 hover:shadow-[0_0_25px_rgba(6,182,212,0.1)] transition-all duration-300 overflow-hidden relative">
        {/* 装饰性边框 */}
        <div className="absolute top-0 left-0 w-3 h-3 border-t-2 border-l-2 border-cyan-500/30 group-hover:border-cyan-400/50 transition-colors"></div>
        <div className="absolute top-0 right-0 w-3 h-3 border-t-2 border-r-2 border-cyan-500/30 group-hover:border-cyan-400/50 transition-colors"></div>
        <div className="absolute bottom-0 left-0 w-3 h-3 border-b-2 border-l-2 border-cyan-500/30 group-hover:border-cyan-400/50 transition-colors"></div>
        <div className="absolute bottom-0 right-0 w-3 h-3 border-b-2 border-r-2 border-cyan-500/30 group-hover:border-cyan-400/50 transition-colors"></div>

        <div className="relative z-10 p-6">
            <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-6">
                <div>
                    <h3 className="text-lg font-bold tracking-wide text-cyan-100 uppercase">{title}</h3>
                    {description && <p className="text-xs text-cyan-600 mt-1 font-mono">{description}</p>}
                </div>
                {action && <div className="flex-shrink-0">{action}</div>}
            </div>
            {children}
        </div>
    </div>
);

// 图表占位符组件
const ChartPlaceholder = ({subtitle, heightClass}: { subtitle: string; heightClass: string }) => (
    <div className={cn("flex items-center justify-center border-2 border-dashed border-cyan-900/50 rounded-lg", heightClass)}>
        <div className="text-center text-cyan-600">
            <Shield className="h-12 w-12 mx-auto mb-3 opacity-20"/>
            <p className="text-sm font-mono">{subtitle}</p>
        </div>
    </div>
);

// 自定义 Tooltip 组件
const CustomTooltip = ({active, payload, label}: any) => {
    if (!active || !payload || payload.length === 0) return null;

    return (
        <div className="bg-[#0a0b10]/95 backdrop-blur-xl border border-cyan-500/30 rounded-lg p-3 shadow-[0_0_20px_rgba(6,182,212,0.2)]">
            <p className="text-xs text-cyan-400 mb-2 font-mono">{label}</p>
            <div className="space-y-1">
                {payload.map((entry: any, index: number) => (
                    <div key={index} className="flex items-center gap-2">
                        <span className="w-2 h-2 rounded-full" style={{backgroundColor: entry.color}}></span>
                        <span className="text-xs text-cyan-100 font-mono">
                            {entry.name}: <span className="font-bold">{entry.value} ms</span>
                        </span>
                    </div>
                ))}
            </div>
        </div>
    );
};

// 时间范围选择器组件
const TimeRangeSelector = ({
                               value,
                               onChange,
                               options
                           }: {
    value: string;
    onChange: (value: string) => void;
    options: Array<{ value: string; label: string }>;
}) => (
    <div className="flex gap-1 bg-black/40 p-1 rounded-lg border border-cyan-900/50">
        {options.map(option => (
            <button
                key={option.value}
                onClick={() => onChange(option.value)}
                className={cn(
                    "px-3 py-1.5 text-xs font-medium rounded transition-all font-mono cursor-pointer",
                    value === option.value
                        ? 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/30'
                        : 'text-cyan-600 hover:text-cyan-400'
                )}
            >
                {option.label}
            </button>
        ))}
    </div>
);


const MonitorDetail = () => {
    const navigate = useNavigate();
    const {id} = useParams<{ id: string }>();
    const [selectedAgent, setSelectedAgent] = useState<string>('all');
    const [timeRange, setTimeRange] = useState<string>('1d');

    // 获取监控详情（聚合数据）
    const {data: monitorDetail, isLoading} = useQuery<PublicMonitor>({
        queryKey: ['monitorDetail', id],
        queryFn: async () => {
            if (!id) throw new Error('Monitor ID is required');
            const response = await getMonitorStatsById(id);
            return response.data;
        },
        refetchInterval: 30000,
        enabled: !!id,
    });

    // 获取各探针的统计数据
    const {data: monitorStats = []} = useQuery<AgentMonitorStat[]>({
        queryKey: ['monitorAgentStats', id],
        queryFn: async () => {
            if (!id) return [];
            const response = await getMonitorAgentStats(id);
            return response.data || [];
        },
        refetchInterval: 30000,
        enabled: !!id,
    });

    // 获取历史数据
    const {data: historyData} = useQuery<GetMetricsResponse>({
        queryKey: ['monitorHistory', id, timeRange],
        queryFn: async () => {
            if (!id) throw new Error('Monitor ID is required');
            const response = await getMonitorHistory(id, timeRange);
            return response.data;
        },
        refetchInterval: 30000,
        enabled: !!id,
    });

    // 获取所有可用的探针列表
    const availableAgents = useMemo(() => {
        if (monitorStats.length === 0) return [];
        return monitorStats.map(stat => ({
            id: stat.agentId,
            label: stat.agentId.substring(0, 8),
        }));
    }, [monitorStats]);

    // 当可用探针列表变化时，检查当前选择的探针是否还存在
    useEffect(() => {
        if (selectedAgent === 'all') return;
        if (!availableAgents.find(agent => agent.id === selectedAgent)) {
            setSelectedAgent('all');
        }
    }, [availableAgents, selectedAgent]);

    // 生成图表数据
    const chartData = useMemo(() => {
        if (!historyData?.series) return [];

        // 过滤出响应时间指标的 series
        const responseTimeSeries = historyData.series.filter(s => s.name === 'response_time');

        // 根据选择的探针过滤
        const filteredSeries = selectedAgent === 'all'
            ? responseTimeSeries
            : responseTimeSeries.filter(s => s.labels?.agent_id === selectedAgent);

        if (filteredSeries.length === 0) return [];

        // 按时间戳分组数据
        const grouped: Record<number, any> = {};

        filteredSeries.forEach(series => {
            const agentId = series.labels?.agent_id || 'unknown';
            const agentKey = `agent_${agentId}`;

            series.data.forEach(point => {
                if (!grouped[point.timestamp]) {
                    grouped[point.timestamp] = {
                        time: new Date(point.timestamp).toLocaleTimeString('zh-CN', {
                            hour: '2-digit',
                            minute: '2-digit',
                        }),
                        timestamp: point.timestamp,
                    };
                }
                grouped[point.timestamp][agentKey] = point.value;
            });
        });

        // 按时间戳排序
        return Object.values(grouped).sort((a, b) => a.timestamp - b.timestamp);
    }, [historyData, selectedAgent]);

    if (isLoading) {
        return (
            <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-8">
                <LoadingSpinner/>
            </div>
        );
    }

    if (!monitorDetail) {
        return (
            <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-8">
                <EmptyState/>
            </div>
        );
    }

    return (
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-8 space-y-6">
            {/* Hero Section */}
            <div className="group bg-[#0f1016]/80 backdrop-blur-md border border-cyan-500/20 shadow-[0_0_15px_rgba(6,182,212,0.05)] hover:border-cyan-500/30 hover:shadow-[0_0_25px_rgba(6,182,212,0.1)] transition-all duration-300 overflow-hidden relative">
                {/* 装饰性边框 */}
                <div className="absolute top-0 left-0 w-3 h-3 border-t-2 border-l-2 border-cyan-500/30 group-hover:border-cyan-400/50 transition-colors"></div>
                <div className="absolute top-0 right-0 w-3 h-3 border-t-2 border-r-2 border-cyan-500/30 group-hover:border-cyan-400/50 transition-colors"></div>
                <div className="absolute bottom-0 left-0 w-3 h-3 border-b-2 border-l-2 border-cyan-500/30 group-hover:border-cyan-400/50 transition-colors"></div>
                <div className="absolute bottom-0 right-0 w-3 h-3 border-b-2 border-r-2 border-cyan-500/30 group-hover:border-cyan-400/50 transition-colors"></div>
                <div className="absolute inset-0 bg-gradient-to-b from-transparent via-cyan-500/5 to-transparent -translate-y-full group-hover:translate-y-full transition-transform duration-1000 ease-in-out pointer-events-none"/>

                <div className="relative z-10 p-6 space-y-6">
                    {/* 返回按钮 */}
                    <button
                        type="button"
                        onClick={() => navigate('/monitors')}
                        className="group inline-flex items-center gap-2 text-xs font-medium uppercase tracking-wider text-cyan-600 hover:text-cyan-400 transition font-mono"
                    >
                        <ArrowLeft className="h-4 w-4 transition-transform group-hover:-translate-x-1"/>
                        返回监控列表
                    </button>

                    {/* 监控信息 */}
                    <div className="flex flex-col gap-6 lg:flex-row lg:items-start lg:justify-between">
                        <div className="flex items-start gap-4 flex-1 min-w-0">
                            <div className="p-3 bg-cyan-950/30 border border-cyan-500/20 rounded-lg flex-shrink-0">
                                <TypeIcon type={monitorDetail.type}/>
                            </div>
                            <div className="flex-1 min-w-0">
                                <div className="text-[10px] font-mono text-cyan-500/60 mb-1 tracking-wider">
                                    MONITOR_ID: {monitorDetail.id.toString().substring(0, 8)}
                                </div>
                                <div className="flex flex-wrap items-center gap-3 mb-2">
                                    <h1 className="text-2xl sm:text-3xl font-bold truncate text-cyan-100 tracking-wide">{monitorDetail.name}</h1>
                                    <StatusBadge status={monitorDetail.status}/>
                                </div>
                                <p className="text-sm text-cyan-500/80 font-mono truncate">
                                    {monitorDetail.showTargetPublic ? monitorDetail.target : '******'}
                                </p>
                            </div>
                        </div>

                        {/* 统计卡片 */}
                        <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 w-full lg:w-auto lg:min-w-[480px]">
                            <LittleStatCard
                                label="监控类型"
                                value={monitorDetail.type.toUpperCase()}
                            />
                            <LittleStatCard
                                label="探针数量"
                                value={monitorDetail.agentCount}
                            />
                            <LittleStatCard
                                label="平均响应"
                                value={`${monitorDetail.responseTime}ms`}
                            />
                            <LittleStatCard
                                label="最慢响应"
                                value={`${monitorDetail.responseTimeMax}ms`}
                            />
                        </div>
                    </div>

                    {/* 证书信息（如果是 HTTPS）*/}
                    {monitorDetail.type === 'https' && monitorDetail.certExpiryTime && (
                        <div className="flex items-center gap-2 pt-4 border-t border-cyan-900/50">
                            <span className="text-xs text-cyan-500/60 font-mono">SSL 证书:</span>
                            <CertBadge
                                expiryTime={monitorDetail.certExpiryTime}
                                daysLeft={monitorDetail.certDaysLeft}
                            />
                        </div>
                    )}
                </div>
            </div>

            {/* 响应时间趋势图表 */}
            <Card
                title="响应时间趋势"
                description="监控各探针的响应时间变化"
                action={
                    <div className="flex flex-col sm:flex-row flex-wrap items-start sm:items-center gap-3">
                        <TimeRangeSelector
                            value={timeRange}
                            onChange={setTimeRange}
                            options={MONITOR_TIME_RANGE_OPTIONS}
                        />
                        {availableAgents.length > 0 && (
                            <select
                                value={selectedAgent}
                                onChange={(e) => setSelectedAgent(e.target.value)}
                                className="rounded-lg border border-cyan-900/50 bg-black/40 px-3 py-2 text-xs font-medium text-cyan-300 hover:border-cyan-500/50 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/20 transition-colors font-mono"
                            >
                                <option value="all">所有探针</option>
                                {availableAgents.map((agent) => (
                                    <option key={agent.id} value={agent.id}>
                                        探针 {agent.label}
                                    </option>
                                ))}
                            </select>
                        )}
                    </div>
                }
            >
                {chartData.length > 0 ? (
                    <ResponsiveContainer width="100%" height={360}>
                        <AreaChart data={chartData}>
                            <defs>
                                {monitorStats
                                    .filter(stat => selectedAgent === 'all' || stat.agentId === selectedAgent)
                                    .map((stat, index) => {
                                        const originalIndex = monitorStats.findIndex(s => s.agentId === stat.agentId);
                                        const agentKey = `agent_${stat.agentId}`;
                                        const color = AGENT_COLORS[originalIndex % AGENT_COLORS.length];
                                        return (
                                            <linearGradient key={agentKey} id={`gradient_${agentKey}`} x1="0" y1="0"
                                                            x2="0" y2="1">
                                                <stop offset="5%" stopColor={color} stopOpacity={0.3}/>
                                                <stop offset="95%" stopColor={color} stopOpacity={0}/>
                                            </linearGradient>
                                        );
                                    })}
                            </defs>
                            <CartesianGrid
                                strokeDasharray="3 3"
                                className="stroke-cyan-900/30"
                                vertical={false}
                            />
                            <XAxis
                                dataKey="time"
                                className="text-xs text-cyan-600 font-mono"
                                stroke="#164e63"
                                tickLine={false}
                                axisLine={false}
                            />
                            <YAxis
                                className="text-xs text-cyan-600 font-mono"
                                stroke="#164e63"
                                tickLine={false}
                                axisLine={false}
                                tickFormatter={(value) => `${value}ms`}
                            />
                            <Tooltip content={<CustomTooltip/>}/>
                            <Legend
                                wrapperStyle={{paddingTop: '20px'}}
                                iconType="circle"
                            />
                            {monitorStats
                                .filter(stat => selectedAgent === 'all' || stat.agentId === selectedAgent)
                                .map((stat) => {
                                    const originalIndex = monitorStats.findIndex(s => s.agentId === stat.agentId);
                                    const agentKey = `agent_${stat.agentId}`;
                                    const color = AGENT_COLORS[originalIndex % AGENT_COLORS.length];
                                    const agentLabel = stat.agentId.substring(0, 8);
                                    return (
                                        <Area
                                            key={agentKey}
                                            type="monotone"
                                            dataKey={agentKey}
                                            name={`探针 ${agentLabel}`}
                                            stroke={color}
                                            strokeWidth={2}
                                            fill={`url(#gradient_${agentKey})`}
                                            activeDot={{r: 5, strokeWidth: 0}}
                                            dot={false}
                                        />
                                    );
                                })}
                        </AreaChart>
                    </ResponsiveContainer>
                ) : (
                    <ChartPlaceholder
                        subtitle="正在收集数据，请稍后查看历史趋势"
                        heightClass="h-80"
                    />
                )}
            </Card>

            {/* 各探针详细数据 */}
            <Card title="探针监控详情" description="各探针的当前状态和统计数据">
                <div className="overflow-x-auto -mx-6 px-6">
                    <table className="min-w-full">
                        <thead>
                        <tr className="border-b border-cyan-900/50">
                            <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-widest text-cyan-500/60 font-mono">
                                探针 ID
                            </th>
                            <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-widest text-cyan-500/60 font-mono">
                                状态
                            </th>
                            <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-widest text-cyan-500/60 font-mono">
                                响应时间
                            </th>
                            <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-widest text-cyan-500/60 font-mono hidden lg:table-cell">
                                最后检测
                            </th>
                            {monitorDetail.type === 'https' && (
                                <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-widest text-cyan-500/60 font-mono hidden xl:table-cell">
                                    证书信息
                                </th>
                            )}
                            <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-widest text-cyan-500/60 font-mono hidden xl:table-cell">
                                错误信息
                            </th>
                        </tr>
                        </thead>
                        <tbody className="divide-y divide-cyan-900/30">
                        {monitorStats.map((stat, index) => {
                            const color = AGENT_COLORS[index % AGENT_COLORS.length];
                            return (
                                <tr key={stat.agentId}
                                    className="hover:bg-cyan-950/20 transition-colors">
                                    <td className="px-4 py-4">
                                        <div className="flex items-center gap-3">
                                            <span
                                                className="inline-block h-2 w-2 rounded-full flex-shrink-0"
                                                style={{backgroundColor: color}}
                                            />
                                            <div className="flex items-center gap-2">
                                                <MapPin className="h-3.5 w-3.5 text-cyan-600"/>
                                                <span
                                                    className="font-mono text-sm text-cyan-200">
                                                    {stat.agentId.substring(0, 8)}...
                                                </span>
                                            </div>
                                        </div>
                                    </td>
                                    <td className="px-4 py-4">
                                        <StatusBadge status={stat.status}/>
                                    </td>
                                    <td className="px-4 py-4">
                                        <div className="flex items-center gap-2">
                                            <Clock className="h-4 w-4 text-cyan-600"/>
                                            <span className="text-sm font-semibold text-cyan-100 font-mono">
                                                {formatTime(stat.responseTime)}
                                            </span>
                                        </div>
                                    </td>
                                    <td className="px-4 py-4 text-sm text-cyan-400 font-mono hidden lg:table-cell">
                                        {formatDateTime(stat.checkedAt)}
                                    </td>
                                    {monitorDetail.type === 'https' && (
                                        <td className="px-4 py-4 hidden xl:table-cell">
                                            {stat.certExpiryTime ? (
                                                <CertBadge
                                                    expiryTime={stat.certExpiryTime}
                                                    daysLeft={stat.certDaysLeft}
                                                />
                                            ) : (
                                                <span
                                                    className="text-xs text-cyan-600">-</span>
                                            )}
                                        </td>
                                    )}
                                    <td className="px-4 py-4 hidden xl:table-cell">
                                        {stat.status === 'down' && stat.message ? (
                                            <div className="flex items-start gap-2 max-w-xs">
                                                <AlertCircle
                                                    className="h-4 w-4 text-rose-400 flex-shrink-0 mt-0.5"/>
                                                <span
                                                    className="text-xs text-rose-300 break-words line-clamp-2 font-mono">
                                                    {stat.message}
                                                </span>
                                            </div>
                                        ) : (
                                            <span className="text-xs text-cyan-600">-</span>
                                        )}
                                    </td>
                                </tr>
                            );
                        })}
                        </tbody>
                    </table>
                </div>
            </Card>
        </div>
    );
};

export default MonitorDetail;
