import type {ReactNode} from 'react';

interface InfoGridProps {
    items: Array<{ label: string; value: ReactNode }>;
}

export const InfoGrid = ({items}: InfoGridProps) => (
    <dl className="grid grid-cols-1 gap-4 text-sm sm:grid-cols-2">
        {items.map((item) => (
            <div key={item.label}>
                <dt className="text-[10px] font-mono uppercase tracking-widest text-cyan-600">{item.label}</dt>
                <dd className="mt-1 font-medium text-cyan-100">{item.value}</dd>
            </div>
        ))}
    </dl>
);
