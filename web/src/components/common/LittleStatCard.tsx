// 统计卡片组件
const LittleStatCard = ({
                      label,
                      value,
                  }: {
    label: string;
    value: string | number;
    sublabel?: string;
}) => (
    <div
        key={label}
        className="rounded-xl bg-black/40 border border-cyan-900/50 p-4 text-left hover:border-cyan-700/50 transition"
    >
        <p className="text-[10px] uppercase tracking-[0.3em] text-cyan-600 font-mono font-bold">{label}</p>
        <p className="mt-2 text-base font-semibold text-cyan-100">{value}</p>
    </div>
);

export default LittleStatCard;