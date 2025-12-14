import {useMemo} from 'react';
import {Cpu} from 'lucide-react';
import {Area, AreaChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis} from 'recharts';
import {ChartPlaceholder, CustomTooltip} from '@/components/common';
import {useMetricsQuery} from '@/hooks/server/queries';
import {ChartContainer} from './ChartContainer';

interface CpuChartProps {
    agentId: string;
    timeRange: string;
}

/**
 * CPU 使用率图表组件
 */
export const CpuChart = ({agentId, timeRange}: CpuChartProps) => {
    // 数据查询
    const {data: metricsResponse, isLoading} = useMetricsQuery({
        agentId,
        type: 'cpu',
        range: timeRange,
    });

    // 数据转换
    const chartData = useMemo(() => {
        const cpuSeries = metricsResponse?.data.series?.find(s => s.name === 'usage');
        if (!cpuSeries) return [];

        return cpuSeries.data.map((point) => ({
            time: new Date(point.timestamp).toLocaleTimeString('zh-CN', {
                hour: '2-digit',
                minute: '2-digit',
            }),
            usage: Number(point.value.toFixed(2)),
            timestamp: point.timestamp,
        }));
    }, [metricsResponse]);

    // 渲染
    if (isLoading) {
        return (
            <ChartContainer title="CPU 使用率" icon={Cpu}>
                <ChartPlaceholder variant="dark"/>
            </ChartContainer>
        );
    }

    return (
        <ChartContainer title="CPU 使用率" icon={Cpu}>
            {chartData.length > 0 ? (
                <ResponsiveContainer width="100%" height={220}>
                    <AreaChart data={chartData}>
                        <defs>
                            <linearGradient id="cpuAreaGradient" x1="0" y1="0" x2="0" y2="1">
                                <stop offset="5%" stopColor="#2563eb" stopOpacity={0.4}/>
                                <stop offset="95%" stopColor="#2563eb" stopOpacity={0}/>
                            </linearGradient>
                        </defs>
                        <CartesianGrid stroke="currentColor" strokeDasharray="4 4" className="stroke-cyan-900/30"/>
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
        </ChartContainer>
    );
};
