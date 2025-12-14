import { Area, AreaChart, ResponsiveContainer } from 'recharts';

interface MiniChartProps {
    data: Array<{ time: string; value: number }>;
    status: string;
    id: string;
}

export const MiniChart = ({ data, status, id }: MiniChartProps) => {
    if (!data || data.length === 0) {
        return (
            <div className="h-16 w-full flex items-center justify-center text-xs text-slate-400 dark:text-slate-500">
                暂无数据
            </div>
        );
    }

    const color = status === 'up' ? '#10b981' : status === 'down' ? '#f59e0b' : '#94a3b8';

    return (
        <div className="h-16 w-full -mb-2">
            <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={data}>
                    <defs>
                        <linearGradient id={`colorLatency-${id}`} x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor={color} stopOpacity={0.2} />
                            <stop offset="95%" stopColor={color} stopOpacity={0} />
                        </linearGradient>
                    </defs>
                    <Area
                        type="monotone"
                        dataKey="value"
                        stroke={color}
                        fillOpacity={1}
                        fill={`url(#colorLatency-${id})`}
                        strokeWidth={1.5}
                        isAnimationActive={false}
                    />
                </AreaChart>
            </ResponsiveContainer>
        </div>
    );
};
