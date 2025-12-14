import {useQuery} from '@tanstack/react-query';
import {getAvailableNetworkInterfaces} from '@/api/agent';

/**
 * 查询 Agent 可用网卡列表
 * @param agentId Agent ID
 * @returns 网卡列表查询结果
 */
export const useNetworkInterfacesQuery = (agentId?: string) => {
    return useQuery({
        queryKey: ['agent', agentId, 'network-interfaces'],
        queryFn: () => getAvailableNetworkInterfaces(agentId!),
        enabled: !!agentId,
    });
};
