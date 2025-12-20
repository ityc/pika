import {useQuery} from '@tanstack/react-query';
import {getAgentMetrics, type GetAgentMetricsRequest} from '@/api/agent';

interface UseMetricsQueryOptions {
    agentId: string;
    type: GetAgentMetricsRequest['type'];
    range: string;
    interfaceName?: string;
    aggregation?: GetAgentMetricsRequest['aggregation'];
}

/**
 * 查询 Agent 历史指标数据
 * @param options 查询选项
 * @returns 历史指标查询结果
 */
export const useMetricsQuery = ({agentId, type, range, interfaceName, aggregation}: UseMetricsQueryOptions) => {
    return useQuery({
        queryKey: ['agent', agentId, 'metrics', type, range, interfaceName, aggregation],
        queryFn: () =>
            getAgentMetrics({
                agentId,
                type,
                range,
                interface: interfaceName,
                aggregation,
            }),
        enabled: !!agentId,
        // refetchInterval: 30000, // 30秒自动刷新
    });
};
