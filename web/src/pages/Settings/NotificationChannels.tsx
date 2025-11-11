import { useState } from 'react';
import { Button, Table, Space, App, Modal, Form, Input, Switch, Tag, Select } from 'antd';
import { Plus, Pencil, Trash2, TestTube } from 'lucide-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { NotificationChannel } from '../../types';
import {
    getNotificationChannels,
    saveNotificationChannels,
    testNotificationChannel,
} from '../../api/notification-channel';
import { getErrorMessage } from '../../lib/utils';

// 支持的渠道类型
const CHANNEL_TYPES = [
    { label: '钉钉', value: 'dingtalk' },
    { label: '企业微信', value: 'wecom' },
    { label: '飞书', value: 'feishu' },
    { label: '自定义Webhook', value: 'webhook' },
];

const NotificationChannels = () => {
    const [form] = Form.useForm();
    const { message: messageApi, modal: modalApi } = App.useApp();
    const queryClient = useQueryClient();
    const [isModalOpen, setIsModalOpen] = useState(false);
    const [isSelectModalOpen, setIsSelectModalOpen] = useState(false);
    const [editingType, setEditingType] = useState<string | null>(null);
    const [selectedType, setSelectedType] = useState<string>('');

    // 获取通知渠道列表
    const { data: channels = [], isLoading } = useQuery({
        queryKey: ['notificationChannels'],
        queryFn: getNotificationChannels,
    });

    // 保存 mutation
    const saveMutation = useMutation({
        mutationFn: saveNotificationChannels,
        onSuccess: () => {
            messageApi.success('保存成功');
            queryClient.invalidateQueries({ queryKey: ['notificationChannels'] });
            setIsModalOpen(false);
            setEditingType(null);
            form.resetFields();
        },
        onError: (error: unknown) => {
            messageApi.error(getErrorMessage(error, '保存失败'));
        },
    });

    // 测试 mutation
    const testMutation = useMutation({
        mutationFn: testNotificationChannel,
        onSuccess: () => {
            messageApi.success('测试通知已发送');
        },
        onError: (error: unknown) => {
            messageApi.error(getErrorMessage(error, '测试失败'));
        },
    });

    const handleAdd = () => {
        setIsSelectModalOpen(true);
    };

    const handleSelectType = () => {
        if (!selectedType) {
            messageApi.warning('请选择渠道类型');
            return;
        }

        // 检查是否已存在
        const exists = channels.find((ch) => ch.type === selectedType);
        if (exists) {
            messageApi.warning('该渠道类型已存在，请直接编辑');
            setIsSelectModalOpen(false);
            return;
        }

        setEditingType(selectedType);
        form.setFieldsValue({
            type: selectedType,
            enabled: true,
        });
        setIsSelectModalOpen(false);
        setIsModalOpen(true);
        setSelectedType('');
    };

    const handleEdit = (record: NotificationChannel) => {
        setEditingType(record.type);
        // 将配置展开到表单字段
        form.setFieldsValue({
            type: record.type,
            enabled: record.enabled,
            ...record.config, // 展开配置对象
        });
        setIsModalOpen(true);
    };

    const handleDelete = (type: string) => {
        modalApi.confirm({
            title: '确认删除',
            content: '确定要删除这个通知渠道配置吗？',
            onOk: () => {
                // 从列表中删除该类型的渠道
                const newChannels = channels.filter((ch) => ch.type !== type);
                saveMutation.mutate(newChannels);
            },
        });
    };

    const handleTest = (type: string) => {
        testMutation.mutate(type);
    };

    const handleSubmit = async () => {
        try {
            const values = await form.validateFields();
            const { type, enabled, ...configFields } = values;

            // 构建配置对象
            const config: Record<string, any> = {};
            Object.keys(configFields).forEach((key) => {
                if (configFields[key] !== undefined && configFields[key] !== '') {
                    config[key] = configFields[key];
                }
            });

            const newChannel: NotificationChannel = {
                type,
                enabled,
                config,
            };

            // 更新或添加渠道
            const existingIndex = channels.findIndex((ch) => ch.type === type);
            let newChannels: NotificationChannel[];
            if (existingIndex >= 0) {
                // 更新现有渠道
                newChannels = [...channels];
                newChannels[existingIndex] = newChannel;
            } else {
                // 添加新渠道
                newChannels = [...channels, newChannel];
            }

            saveMutation.mutate(newChannels);
        } catch (error) {
            // 表单验证失败
        }
    };

    const getTypeLabel = (type: string) => {
        const types = {
            dingtalk: '钉钉',
            wecom: '企业微信',
            feishu: '飞书',
            email: '邮件',
            webhook: '自定义Webhook',
        };
        return types[type as keyof typeof types] || type;
    };

    const columns = [
        {
            title: '渠道类型',
            dataIndex: 'type',
            key: 'type',
            render: (type: string) => getTypeLabel(type),
        },
        {
            title: '状态',
            dataIndex: 'enabled',
            key: 'enabled',
            render: (enabled: boolean) =>
                enabled ? <Tag color="success">启用</Tag> : <Tag>禁用</Tag>,
        },
        {
            title: '操作',
            key: 'action',
            render: (_: unknown, record: NotificationChannel) => (
                <Space>
                    <Button
                        type="link"
                        size="small"
                        icon={<TestTube size={14} />}
                        onClick={() => handleTest(record.type)}
                        loading={testMutation.isPending}
                        disabled={!record.enabled}
                    >
                        测试
                    </Button>
                    <Button type="link" size="small" icon={<Pencil size={14} />} onClick={() => handleEdit(record)}>
                        配置
                    </Button>
                    <Button
                        type="link"
                        size="small"
                        danger
                        icon={<Trash2 size={14} />}
                        onClick={() => handleDelete(record.type)}
                    >
                        删除
                    </Button>
                </Space>
            ),
        },
    ];

    // 获取未配置的渠道类型
    const availableTypes = CHANNEL_TYPES.filter((type) => !channels.find((ch) => ch.type === type.value));

    return (
        <div>
            <div className="mb-4 flex justify-between items-start">
                <div>
                    <h2 className="text-xl font-bold">通知渠道管理</h2>
                    <p className="text-gray-500 mt-2">配置钉钉、企业微信、飞书和自定义Webhook通知渠道</p>
                </div>
                <Button type="primary" icon={<Plus size={16} />} onClick={handleAdd} disabled={availableTypes.length === 0}>
                    添加渠道
                </Button>
            </div>

            <Table columns={columns} dataSource={channels} rowKey="type" loading={isLoading} />

            {/* 选择渠道类型对话框 */}
            <Modal
                title="选择通知渠道类型"
                open={isSelectModalOpen}
                onOk={handleSelectType}
                onCancel={() => {
                    setIsSelectModalOpen(false);
                    setSelectedType('');
                }}
                width={400}
            >
                <Form layout="vertical">
                    <Form.Item label="渠道类型" required>
                        <Select
                            placeholder="请选择要添加的渠道类型"
                            value={selectedType || undefined}
                            onChange={setSelectedType}
                            options={availableTypes}
                        />
                    </Form.Item>
                </Form>
            </Modal>

            <Modal
                title={`配置${getTypeLabel(editingType || '')}`}
                open={isModalOpen}
                onOk={handleSubmit}
                onCancel={() => {
                    setIsModalOpen(false);
                    setEditingType(null);
                    form.resetFields();
                }}
                confirmLoading={saveMutation.isPending}
                width={600}
            >
                <Form form={form} layout="vertical">
                    <Form.Item name="type" hidden>
                        <Input />
                    </Form.Item>

                    <Form.Item label="启用" name="enabled" valuePropName="checked" initialValue={true}>
                        <Switch />
                    </Form.Item>

                    <Form.Item noStyle shouldUpdate={(prev, curr) => prev.type !== curr.type}>
                        {({ getFieldValue }) => {
                            const type = getFieldValue('type');
                            if (type === 'dingtalk') {
                                return (
                                    <>
                                        <Form.Item
                                            label="访问令牌 (Access Token)"
                                            name="secretKey"
                                            rules={[{ required: true, message: '请输入访问令牌' }]}
                                            tooltip="在钉钉机器人配置中获取的 access_token"
                                        >
                                            <Input placeholder="输入访问令牌" />
                                        </Form.Item>
                                        <Form.Item label="加签密钥（可选）" name="signSecret" tooltip="如果启用了加签，请填写 SEC 开头的密钥">
                                            <Input.Password placeholder="SEC 开头的加签密钥" />
                                        </Form.Item>
                                    </>
                                );
                            }
                            if (type === 'wecom') {
                                return (
                                    <Form.Item
                                        label="Webhook Key"
                                        name="secretKey"
                                        rules={[{ required: true, message: '请输入 Webhook Key' }]}
                                        tooltip="企业微信群机器人的 Webhook Key"
                                    >
                                        <Input placeholder="输入 Webhook Key" />
                                    </Form.Item>
                                );
                            }
                            if (type === 'feishu') {
                                return (
                                    <>
                                        <Form.Item
                                            label="Webhook Token"
                                            name="secretKey"
                                            rules={[{ required: true, message: '请输入 Webhook Token' }]}
                                            tooltip="飞书群机器人的 Webhook Token"
                                        >
                                            <Input placeholder="输入 Webhook Token" />
                                        </Form.Item>
                                        <Form.Item label="签名密钥（可选）" name="signSecret" tooltip="如果启用了签名验证，请填写密钥">
                                            <Input.Password placeholder="输入签名密钥" />
                                        </Form.Item>
                                    </>
                                );
                            }
                            if (type === 'webhook') {
                                return (
                                    <Form.Item
                                        label="Webhook URL"
                                        name="url"
                                        rules={[
                                            { required: true, message: '请输入 Webhook URL' },
                                            { type: 'url', message: '请输入有效的 URL' },
                                        ]}
                                    >
                                        <Input placeholder="https://your-server.com/webhook" />
                                    </Form.Item>
                                );
                            }
                            return null;
                        }}
                    </Form.Item>
                </Form>
            </Modal>
        </div>
    );
};

export default NotificationChannels;
