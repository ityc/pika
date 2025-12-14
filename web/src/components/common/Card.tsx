import type {ReactNode} from 'react';
import {cn} from '@/lib/utils';

interface CardProps {
    title?: string;
    description?: string;
    action?: ReactNode;
    children: ReactNode;
    variant?: 'light' | 'dark';
}

export const Card = ({
                         title,
                         description,
                         action,
                         children,
                         variant = 'light',
                     }: CardProps) => {
    const isDark = variant === 'dark';

    return (
        <section className={cn(
            "rounded-2xl border p-6",
            isDark
                ? "border-cyan-900/50 bg-[#0a0b10]/90 shadow-2xl backdrop-blur-sm"
                : "rounded-xl border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800"
        )}>
            {(title || description || action) && (
                <div className={cn(
                    "flex flex-col gap-3 border-b pb-4 sm:flex-row sm:items-start sm:justify-between",
                    isDark ? "border-cyan-900/30" : "border-slate-200 dark:border-slate-700"
                )}>
                    <div>
                        {title && (
                            <h2 className={cn(
                                "text-sm font-bold",
                                isDark
                                    ? "font-mono uppercase tracking-widest text-cyan-400"
                                    : "text-lg font-semibold text-slate-900 dark:text-white"
                            )}>
                                {title}
                            </h2>
                        )}
                        {description && (
                            <p className={cn(
                                "mt-1 text-xs",
                                isDark ? "text-cyan-600" : "text-sm text-slate-500 dark:text-slate-400"
                            )}>
                                {description}
                            </p>
                        )}
                    </div>
                    {action && <div className="shrink-0">{action}</div>}
                </div>
            )}
            <div className="pt-4">{children}</div>
        </section>
    );
};
