// 统计卡片组件
import {cn} from "@/lib/utils.ts";

interface Props {
    title: string;
    value: any;
    unit?: string
    icon: any;
    color: string;
    alert?: boolean,
    glow?: boolean,
}

const StatBlock = ({title, value, unit, icon: Icon, color, alert, glow}: Props) => {

    const colorMap = {
        cyan: 'dark:text-cyan-400 dark:border-cyan-500/30 dark:bg-cyan-500/5',
        emerald: 'dark:text-emerald-400 dark:border-emerald-500/30 dark:bg-emerald-500/5',
        rose: 'dark:text-rose-400 dark:border-rose-500/30 dark:bg-rose-500/5',
        purple: 'dark:text-purple-400 dark:border-purple-500/30 dark:bg-purple-500/5'
    };
    const style = colorMap[color] || colorMap.cyan;

    const iconColor = {
        cyan: 'text-cyan-400',
        emerald: 'text-emerald-400',
        rose: 'text-rose-400',
        purple: 'text-purple-400'
    }
    let iconStyle = iconColor[color] || colorMap.cyan;

    return (
        <div
            className={cn(
                `relative overflow-hidden rounded-xl border p-5`,
                'bg-white/80 backdrop-blur-md border border-slate-200 shadow-sm',
                style,
                alert && 'animate-pulse bg-rose-500/10',
                glow && 'shadow-[0_0_20px_rgba(16,185,129,0.1)]',
            )}>
            <div className="absolute -right-4 -bottom-4 opacity-10 rotate-[-15deg]"><Icon className="w-24 h-24"/></div>
            <div className="relative z-10 flex justify-between items-start">
                <div>
                    <div className="text-xs font-bold font-mono uppercase tracking-widest opacity-70 mb-2">{title}</div>
                    <div className="text-4xl font-black tracking-tight flex items-baseline gap-1">{value}{unit &&
                        <span className="text-sm font-normal opacity-60 ml-1">{unit}</span>}</div>
                </div>
                <div className={`p-3`}>
                    <Icon className={cn("w-6 h-6", iconStyle)}/>
                </div>
            </div>
        </div>
    );
};

export default StatBlock;