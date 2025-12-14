import {useEffect, useMemo, useState} from 'react';
import {useQuery} from '@tanstack/react-query';
import {Shield} from 'lucide-react';
import {Area, AreaChart, CartesianGrid, Legend, ResponsiveContainer, Tooltip, XAxis, YAxis} from 'recharts';
import {type GetMetricsResponse, getMonitorHistory} from '@/api/monitor';
import {AGENT_COLORS} from '@/constants/colors';
import {MONITOR_TIME_RANGE_OPTIONS} from '@/constants/time';
import {cn} from '@/lib/utils';
import type {AgentMonitorStat} from '@/types';

interface ResponseTimeChartProps {
    monitorId: string;
    monitorStats: AgentMonitorStat[];
}

// 自定义 Tooltip 组件
const CustomTooltip = ({active, payload, label}: any) => {
    if (!active || !payload || payload.length === 0) return null;

    return (
        <div
            className="bg-[#0a0b10]/95 backdrop-blur-xl border border-cyan-500/30 rounded-lg p-3 shadow-[0_0_20px_rgba(6,182,212,0.2)]">
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

// 图表占位符组件
const ChartPlaceholder = ({subtitle, heightClass}: { subtitle: string; heightClass: string }) => (
    <div
        className={cn("flex items-center justify-center border-2 border-dashed border-cyan-900/50 rounded-lg", heightClass)}>
        <div className="text-center text-cyan-600">
            <Shield className="h-12 w-12 mx-auto mb-3 opacity-20"/>
            <p className="text-sm font-mono">{subtitle}</p>
        </div>
    </div>
);

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

/**
 * 响应时间趋势图表组件
 * 显示监控各探针的响应时间变化
 */
export const ResponseTimeChart = ({monitorId, monitorStats}: ResponseTimeChartProps) => {
    const [selectedAgent, setSelectedAgent] = useState<string>('all');
    const [timeRange, setTimeRange] = useState<string>('1d');

    // 获取历史数据
    const {data: historyData} = useQuery<GetMetricsResponse>({
        queryKey: ['monitorHistory', monitorId, timeRange],
        queryFn: async () => {
            if (!monitorId) throw new Error('Monitor ID is required');
            const response = await getMonitorHistory(monitorId, timeRange);
            return response.data;
        },
        refetchInterval: 30000,
        enabled: !!monitorId,
    });

    // 获取所有可用的探针列表
    const availableAgents = useMemo(() => {
        if (monitorStats.length === 0) return [];
        return monitorStats.map(stat => ({
            id: stat.agentId,
            name: stat.agentName || stat.agentId.substring(0, 8),
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
        const responseTimeSeries = historyData.series?.filter(s => s.name === 'response_time');

        // 根据选择的探针过滤（使用 agent_name，如果没有则fallback到 agent_id）
        const filteredSeries = selectedAgent === 'all'
            ? responseTimeSeries
            : responseTimeSeries.filter(s => {
                // 优先使用 agent_name，如果没有则使用 agent_id
                const agentIdentifier = s.labels?.agent_name || s.labels?.agent_id;
                return agentIdentifier === selectedAgent;
            });

        if (filteredSeries.length === 0) return [];

        // 按时间戳分组数据
        const grouped: Record<number, any> = {};

        filteredSeries.forEach(series => {
            // 优先使用 agent_name，如果没有则使用 agent_id
            const agentIdentifier = series.labels?.agent_name || series.labels?.agent_id || 'unknown';
            const agentKey = `agent_${agentIdentifier}`;

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

    return (
        <div
            className="group bg-[#0f1016]/80 backdrop-blur-md border border-cyan-500/20 shadow-[0_0_15px_rgba(6,182,212,0.05)] hover:border-cyan-500/30 hover:shadow-[0_0_25px_rgba(6,182,212,0.1)] transition-all duration-300 overflow-hidden relative">
            {/* 装饰性边框 */}
            <div
                className="absolute top-0 left-0 w-3 h-3 border-t-2 border-l-2 border-cyan-500/30 group-hover:border-cyan-400/50 transition-colors"></div>
            <div
                className="absolute top-0 right-0 w-3 h-3 border-t-2 border-r-2 border-cyan-500/30 group-hover:border-cyan-400/50 transition-colors"></div>
            <div
                className="absolute bottom-0 left-0 w-3 h-3 border-b-2 border-l-2 border-cyan-500/30 group-hover:border-cyan-400/50 transition-colors"></div>
            <div
                className="absolute bottom-0 right-0 w-3 h-3 border-b-2 border-r-2 border-cyan-500/30 group-hover:border-cyan-400/50 transition-colors"></div>

            <div className="relative z-10 p-6">
                <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-6">
                    <div>
                        <h3 className="text-lg font-bold tracking-wide text-cyan-100 uppercase">响应时间趋势</h3>
                        <p className="text-xs text-cyan-600 mt-1 font-mono">监控各探针的响应时间变化</p>
                    </div>
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
                                    <option key={agent.id} value={agent.name}>
                                        {agent.name}
                                    </option>
                                ))}
                            </select>
                        )}
                    </div>
                </div>

                {chartData.length > 0 ? (
                    <ResponsiveContainer width="100%" height={360}>
                        <AreaChart data={chartData}>
                            <defs>
                                {monitorStats
                                    .filter(stat => {
                                        const agentIdentifier = stat.agentName || stat.agentId;
                                        return selectedAgent === 'all' || agentIdentifier === selectedAgent;
                                    })
                                    .map((stat) => {
                                        const originalIndex = monitorStats.findIndex(s => s.agentId === stat.agentId);
                                        const agentIdentifier = stat.agentName || stat.agentId;
                                        const agentKey = `agent_${agentIdentifier}`;
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
                                .filter(stat => {
                                    const agentIdentifier = stat.agentName || stat.agentId;
                                    return selectedAgent === 'all' || agentIdentifier === selectedAgent;
                                })
                                .map((stat) => {
                                    const originalIndex = monitorStats.findIndex(s => s.agentId === stat.agentId);
                                    const agentIdentifier = stat.agentName || stat.agentId;
                                    const agentKey = `agent_${agentIdentifier}`;
                                    const color = AGENT_COLORS[originalIndex % AGENT_COLORS.length];
                                    const agentLabel = stat.agentName || stat.agentId.substring(0, 8);
                                    return (
                                        <Area
                                            key={agentKey}
                                            type="monotone"
                                            dataKey={agentKey}
                                            name={agentLabel}
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
            </div>
        </div>
    );
};
