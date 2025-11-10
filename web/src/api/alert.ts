import { get, post, put, del } from './request';
import type { AlertConfig, AlertRecord } from '../types';

// 获取探针的告警配置列表
export const getAlertConfigsByAgent = async (agentId: string): Promise<AlertConfig[]> => {
    const response = await get<AlertConfig[]>(`/admin/agents/${agentId}/alert-configs`);
    return response.data;
};

// 创建告警配置
export const createAlertConfig = async (config: AlertConfig): Promise<AlertConfig> => {
    const response = await post<AlertConfig>('/admin/alert-configs', config);
    return response.data;
};

// 获取告警配置详情
export const getAlertConfig = async (id: string): Promise<AlertConfig> => {
    const response = await get<AlertConfig>(`/admin/alert-configs/${id}`);
    return response.data;
};

// 更新告警配置
export const updateAlertConfig = async (id: string, config: AlertConfig): Promise<AlertConfig> => {
    const response = await put<AlertConfig>(`/admin/alert-configs/${id}`, config);
    return response.data;
};

// 删除告警配置
export const deleteAlertConfig = async (id: string): Promise<void> => {
    await del(`/admin/alert-configs/${id}`);
};

// 测试告警通知
export const testNotification = async (id: string): Promise<{ message: string }> => {
    const response = await post<{ message: string }>(`/admin/alert-configs/${id}/test`);
    return response.data;
};

// 获取告警记录列表
export const getAlertRecords = async (
    agentId?: string,
    limit?: number,
    offset?: number
): Promise<{
    records: AlertRecord[];
    total: number;
    limit: number;
    offset: number;
}> => {
    let url = '/admin/alert-records?';
    if (agentId) url += `agentId=${agentId}&`;
    if (limit) url += `limit=${limit}&`;
    if (offset) url += `offset=${offset}&`;

    const response = await get<{
        records: AlertRecord[];
        total: number;
        limit: number;
        offset: number;
    }>(url);
    return response.data;
};
