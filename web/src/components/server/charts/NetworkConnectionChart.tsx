import {useMemo} from 'react';
import {Network} from 'lucide-react';
import {CartesianGrid, Legend, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis} from 'recharts';
import {ChartPlaceholder, CustomTooltip} from '@/components/common';
import {useMetricsQuery} from '@/hooks/server/queries';
import {ChartContainer} from './ChartContainer';

interface NetworkConnectionChartProps {
    agentId: string;
    timeRange: string;
}

/**
 * 网络连接统计图表组件
 */
export const NetworkConnectionChart = ({agentId, timeRange}: NetworkConnectionChartProps) => {
    // 数据查询
    const {data: metricsResponse, isLoading} = useMetricsQuery({
        agentId,
        type: 'network_connection',
        range: timeRange,
    });

    // 数据转换
    const chartData = useMemo(() => {
        if (!metricsResponse?.data.series || metricsResponse.data.series?.length === 0) return [];

        // 按时间戳聚合所有连接状态系列
        const timeMap = new Map<number, any>();

        metricsResponse.data.series?.forEach(series => {
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
    }, [metricsResponse]);

    // 渲染
    if (isLoading) {
        return (
            <ChartContainer title="网络连接统计" icon={Network}>
                <ChartPlaceholder variant="dark"/>
            </ChartContainer>
        );
    }

    return (
        <ChartContainer title="网络连接统计" icon={Network}>
            {chartData.length > 0 ? (
                <ResponsiveContainer width="100%" height={220}>
                    <LineChart data={chartData}>
                        <CartesianGrid stroke="currentColor" strokeDasharray="4 4" className="stroke-cyan-900/30"/>
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
        </ChartContainer>
    );
};
