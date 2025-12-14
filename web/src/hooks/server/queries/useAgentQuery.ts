import {useQuery} from '@tanstack/react-query';
import {getAgent} from '@/api/agent';

/**
 * 查询 Agent 基础信息
 * @param agentId Agent ID
 * @returns Agent 查询结果
 */
export const useAgentQuery = (agentId?: string) => {
    return useQuery({
        queryKey: ['agent', agentId],
        queryFn: () => getAgent(agentId!),
        enabled: !!agentId,
        staleTime: 60000, // 1分钟缓存
    });
};
