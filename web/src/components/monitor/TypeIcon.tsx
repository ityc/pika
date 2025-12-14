import { Globe, Server, ShieldCheck, Wifi } from 'lucide-react';

interface TypeIconProps {
    type: string;
}

export const TypeIcon = ({ type }: TypeIconProps) => {
    switch (type.toLowerCase()) {
        case 'https':
            return <ShieldCheck className="w-4 h-4 text-purple-500 dark:text-purple-400" />;
        case 'http':
            return <Globe className="w-4 h-4 text-blue-500 dark:text-blue-400" />;
        case 'tcp':
            return <Server className="w-4 h-4 text-orange-500 dark:text-orange-400" />;
        case 'icmp':
        case 'ping':
            return <Wifi className="w-4 h-4 text-cyan-500 dark:text-cyan-400" />;
        default:
            return <Server className="w-4 h-4 text-slate-500 dark:text-slate-400" />;
    }
};
