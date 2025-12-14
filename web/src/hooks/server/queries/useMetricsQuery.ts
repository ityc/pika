import {useQuery} from '@tanstack/react-query';
import {getAgentMetrics, type GetAgentMetricsRequest} from '@/api/agent';

interface UseMetricsQueryOptions {
    agentId: string;
    type: GetAgentMetricsRequest['type'];
    range: string;
    interfaceName?: string;
}

/**
 * 查询 Agent 历史指标数据
 * 自动每 30 秒刷新一次
 * @param options 查询选项
 * @returns 历史指标查询结果
 */
export const useMetricsQuery = ({agentId, type, range, interfaceName}: UseMetricsQueryOptions) => {
    return useQuery({
        queryKey: ['agent', agentId, 'metrics', type, range, interfaceName],
        queryFn: () =>
            getAgentMetrics({
                agentId,
                type,
                range,
                interface: interfaceName,
            }),
        enabled: !!agentId,
        refetchInterval: 30000, // 30秒自动刷新
    });
};
