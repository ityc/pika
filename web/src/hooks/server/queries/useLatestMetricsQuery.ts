import {useQuery} from '@tanstack/react-query';
import {getAgentLatestMetrics} from '@/api/agent';

/**
 * 查询 Agent 最新指标
 * 自动每 5 秒刷新一次
 * @param agentId Agent ID
 * @returns 最新指标查询结果
 */
export const useLatestMetricsQuery = (agentId?: string) => {
    return useQuery({
        queryKey: ['agent', agentId, 'metrics', 'latest'],
        queryFn: () => getAgentLatestMetrics(agentId!),
        enabled: !!agentId,
        refetchInterval: 5000, // 5秒自动刷新
    });
};
