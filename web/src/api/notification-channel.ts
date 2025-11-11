import { get, post, put } from './request';
import type { NotificationChannel } from '../types';

const PROPERTY_ID_NOTIFICATION_CHANNELS = 'notification_channels';

// 获取通知渠道列表
export const getNotificationChannels = async (): Promise<NotificationChannel[]> => {
    const response = await get<{ id: string; name: string; value: NotificationChannel[] }>(
        `/admin/properties/${PROPERTY_ID_NOTIFICATION_CHANNELS}`
    );
    return response.data.value || [];
};

// 保存通知渠道列表
export const saveNotificationChannels = async (channels: NotificationChannel[]): Promise<void> => {
    await put(`/admin/properties/${PROPERTY_ID_NOTIFICATION_CHANNELS}`, {
        name: '通知渠道配置',
        value: channels,
    });
};

// 测试通知渠道（从数据库读取配置）
export const testNotificationChannel = async (type: string): Promise<{ message: string }> => {
    const response = await post<{ message: string }>(`/admin/notification-channels/${type}/test`);
    return response.data;
};
