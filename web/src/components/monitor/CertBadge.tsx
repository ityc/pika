import { ShieldCheck } from 'lucide-react';
import { cn } from '@/lib/utils';

interface CertBadgeProps {
    expiryTime: number;
    daysLeft: number;
}

export const CertBadge = ({ expiryTime, daysLeft }: CertBadgeProps) => {
    if (!expiryTime || daysLeft === undefined) return null;

    const isExpired = daysLeft < 0;
    let colorClass = "text-emerald-600 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-500/10 border-emerald-200 dark:border-emerald-500/20";

    if (isExpired) {
        colorClass = "text-rose-600 dark:text-rose-400 bg-rose-50 dark:bg-rose-500/10 border-rose-200 dark:border-rose-500/20";
    } else if (daysLeft < 30) {
        colorClass = "text-amber-600 dark:text-amber-400 bg-amber-50 dark:bg-amber-500/10 border-amber-200 dark:border-amber-500/20";
    }

    return (
        <div className={cn("flex items-center gap-1.5 px-2 py-1 rounded text-xs border", colorClass)}>
            <ShieldCheck className="w-3 h-3" />
            <span>{isExpired ? "已过期" : `${daysLeft} 天后过期`}</span>
        </div>
    );
};
