import { useState, useEffect } from 'react';
import { Form, Input, Switch, InputNumber, Button, Space, App, Card, Select, Divider } from 'antd';
import { Save, TestTube } from 'lucide-react';
import type { AlertConfig } from '../../types';
import { getAlertConfigsByAgent, createAlertConfig, updateAlertConfig, testNotification } from '../../api/alert';
import { getAgents } from '../../api/agent';
import { getErrorMessage } from '../../lib/utils';

const AlertSettings = () => {
    const [form] = Form.useForm();
    const { message: messageApi } = App.useApp();
    const [loading, setLoading] = useState(false);
    const [testLoading, setTestLoading] = useState(false);
    const [configId, setConfigId] = useState<string | null>(null);
    const [agents, setAgents] = useState<{ id: string; name: string }[]>([]);

    // 加载探针列表
    const loadAgents = async () => {
        try {
            const data = await getAgents();
            setAgents(data.items || []);
        } catch (error: unknown) {
            console.error('获取探针列表失败', error);
        }
    };

    // 加载配置
    const loadConfig = async () => {
        try {
            // 获取全局配置（使用特殊的 agentId "global"）
            const configs = await getAlertConfigsByAgent('global');
            if (configs && configs.length > 0) {
                const config = configs[0];
                setConfigId(config.id || null);
                form.setFieldsValue(config);
            } else {
                // 使用默认值
                form.setFieldsValue({
                    name: '全局告警配置',
                    enabled: true,
                    agentIds: [], // 空数组表示监控所有探针
                    rules: {
                        cpuEnabled: true,
                        cpuThreshold: 80,
                        cpuDuration: 60,
                        memoryEnabled: true,
                        memoryThreshold: 80,
                        memoryDuration: 60,
                        diskEnabled: true,
                        diskThreshold: 85,
                        diskDuration: 60,
                        networkEnabled: false,
                        networkDuration: 60,
                    },
                    notification: {
                        dingTalkEnabled: false,
                        dingTalkWebhook: '',
                        dingTalkSecret: '',
                        weComEnabled: false,
                        weComWebhook: '',
                        feishuEnabled: false,
                        feishuWebhook: '',
                        emailEnabled: false,
                        emailAddresses: [],
                        customWebhookEnabled: false,
                        customWebhookUrl: '',
                    },
                });
            }
        } catch (error: unknown) {
            messageApi.error(getErrorMessage(error, '加载配置失败'));
        }
    };

    useEffect(() => {
        loadAgents();
        loadConfig();
    }, []);

    const handleSubmit = async () => {
        try {
            const values = await form.validateFields();
            setLoading(true);

            const alertConfig: AlertConfig = {
                ...values,
                agentId: 'global', // 全局配置使用特殊的 agentId
            };

            if (configId) {
                // 更新
                await updateAlertConfig(configId, alertConfig);
                messageApi.success('告警配置更新成功');
            } else {
                // 创建
                const created = await createAlertConfig(alertConfig);
                setConfigId(created.id || null);
                messageApi.success('告警配置创建成功');
            }
        } catch (error: unknown) {
            messageApi.error(getErrorMessage(error, '保存配置失败'));
        } finally {
            setLoading(false);
        }
    };

    const handleTest = async () => {
        if (!configId) {
            messageApi.warning('请先保存配置后再测试');
            return;
        }

        try {
            setTestLoading(true);
            await testNotification(configId);
            messageApi.success('测试通知已发送，请检查您的通知渠道');
        } catch (error: unknown) {
            messageApi.error(getErrorMessage(error, '发送测试通知失败'));
        } finally {
            setTestLoading(false);
        }
    };

    return (
        <div>
            <Form form={form}>
                <Space direction={'vertical'} className={'w-full'}>
                    {/* 基本信息 */}
                    <Card title="基本信息" type={'inner'}>
                        <Form.Item
                            label="配置名称"
                            name="name"
                            rules={[{ required: true, message: '请输入配置名称' }]}
                        >
                            <Input placeholder="例如：全局告警配置" />
                        </Form.Item>

                        <Form.Item label="启用告警" name="enabled" valuePropName="checked">
                            <Switch checkedChildren="开启" unCheckedChildren="关闭" />
                        </Form.Item>

                        <Form.Item
                            label="监控范围"
                            name="agentIds"
                            tooltip="留空表示监控所有探针，否则只监控选中的探针"
                        >
                            <Select
                                mode="multiple"
                                placeholder="留空监控所有探针，或选择特定探针"
                                allowClear
                                options={agents.map((agent) => ({
                                    label: agent.name,
                                    value: agent.id,
                                }))}
                            />
                        </Form.Item>
                    </Card>

                    <Divider orientation={'left'}>告警规则</Divider>

                    {[
                        {
                            key: 'cpu',
                            title: 'CPU 告警规则',
                            thresholdLabel: 'CPU 使用率阈值 (%)',
                        },
                        {
                            key: 'memory',
                            title: '内存告警规则',
                            thresholdLabel: '内存使用率阈值 (%)',
                        },
                        {
                            key: 'disk',
                            title: '磁盘告警规则',
                            thresholdLabel: '磁盘使用率阈值 (%)',
                        },
                    ].map((rule) => (
                        <Card key={rule.key} title={rule.title} type={'inner'}>
                            <Form.Item
                                noStyle
                                shouldUpdate={(prevValues, currentValues) =>
                                    prevValues.rules?.[`${rule.key}Enabled`] !==
                                    currentValues.rules?.[`${rule.key}Enabled`]
                                }
                            >
                                {({ getFieldValue }) => {
                                    const enabled = Boolean(getFieldValue(['rules', `${rule.key}Enabled`]));
                                    return (
                                        <div className="flex items-center gap-8">
                                            <Form.Item
                                                label={'开关'}
                                                name={['rules', `${rule.key}Enabled`]}
                                                valuePropName="checked"
                                                className="mb-0"
                                            >
                                                <Switch />
                                            </Form.Item>
                                            <Form.Item
                                                label={rule.thresholdLabel}
                                                name={['rules', `${rule.key}Threshold`]} className="mb-0">
                                                <InputNumber
                                                    min={0}
                                                    max={100}
                                                    precision={0}
                                                    style={{ width: '100%' }}
                                                    disabled={!enabled}
                                                />
                                            </Form.Item>
                                            <Form.Item
                                                label={'持续时间（秒）'}
                                                name={['rules', `${rule.key}Duration`]}
                                                className="mb-0"
                                                tooltip="超过阈值并持续此时间后才触发告警"
                                            >
                                                <InputNumber
                                                    min={1}
                                                    max={3600}
                                                    style={{ width: '100%' }}
                                                    disabled={!enabled}
                                                />
                                            </Form.Item>
                                        </div>
                                    );
                                }}
                            </Form.Item>
                        </Card>
                    ))}

                    <Divider orientation={'left'}>通知渠道</Divider>

                    {/* 钉钉通知 */}
                    <Card title="钉钉通知" type={'inner'}>
                        <Form.Item
                            label="启用钉钉通知"
                            name={['notification', 'dingTalkEnabled']}
                            valuePropName="checked"
                        >
                            <Switch />
                        </Form.Item>

                        <Form.Item
                            noStyle
                            shouldUpdate={(prevValues, currentValues) =>
                                prevValues.notification?.dingTalkEnabled !== currentValues.notification?.dingTalkEnabled
                            }
                        >
                            {({ getFieldValue }) =>
                                getFieldValue(['notification', 'dingTalkEnabled']) ? (
                                    <>
                                        <Form.Item
                                            label="Webhook URL"
                                            name={['notification', 'dingTalkWebhook']}
                                            rules={[
                                                { required: true, message: '请输入钉钉 Webhook URL' },
                                                { type: 'url', message: '请输入有效的 URL' },
                                            ]}
                                        >
                                            <Input placeholder="https://oapi.dingtalk.com/robot/send?access_token=..." />
                                        </Form.Item>
                                        <Form.Item
                                            label="加签密钥（可选）"
                                            name={['notification', 'dingTalkSecret']}
                                            tooltip="如果钉钉机器人启用了加签，请填写密钥"
                                        >
                                            <Input.Password placeholder="SEC 开头的加签密钥" />
                                        </Form.Item>
                                    </>
                                ) : null
                            }
                        </Form.Item>
                    </Card>

                    {/* 企业微信通知 */}
                    <Card title="企业微信通知" type={'inner'}>
                        <Form.Item label="启用企业微信通知" name={['notification', 'weComEnabled']} valuePropName="checked">
                            <Switch />
                        </Form.Item>

                        <Form.Item
                            noStyle
                            shouldUpdate={(prevValues, currentValues) =>
                                prevValues.notification?.weComEnabled !== currentValues.notification?.weComEnabled
                            }
                        >
                            {({ getFieldValue }) =>
                                getFieldValue(['notification', 'weComEnabled']) ? (
                                    <Form.Item
                                        label="Webhook URL"
                                        name={['notification', 'weComWebhook']}
                                        rules={[
                                            { required: true, message: '请输入企业微信 Webhook URL' },
                                            { type: 'url', message: '请输入有效的 URL' },
                                        ]}
                                    >
                                        <Input placeholder="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=..." />
                                    </Form.Item>
                                ) : null
                            }
                        </Form.Item>
                    </Card>

                    {/* 飞书通知 */}
                    <Card title="飞书通知" type={'inner'}>
                        <Form.Item label="启用飞书通知" name={['notification', 'feishuEnabled']} valuePropName="checked">
                            <Switch />
                        </Form.Item>

                        <Form.Item
                            noStyle
                            shouldUpdate={(prevValues, currentValues) =>
                                prevValues.notification?.feishuEnabled !== currentValues.notification?.feishuEnabled
                            }
                        >
                            {({ getFieldValue }) =>
                                getFieldValue(['notification', 'feishuEnabled']) ? (
                                    <Form.Item
                                        label="Webhook URL"
                                        name={['notification', 'feishuWebhook']}
                                        rules={[
                                            { required: true, message: '请输入飞书 Webhook URL' },
                                            { type: 'url', message: '请输入有效的 URL' },
                                        ]}
                                    >
                                        <Input placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/..." />
                                    </Form.Item>
                                ) : null
                            }
                        </Form.Item>
                    </Card>

                    {/* 自定义 Webhook */}
                    <Card title="自定义 Webhook" type={'inner'}>
                        <Form.Item
                            label="启用自定义 Webhook"
                            name={['notification', 'customWebhookEnabled']}
                            valuePropName="checked"
                        >
                            <Switch />
                        </Form.Item>

                        <Form.Item
                            noStyle
                            shouldUpdate={(prevValues, currentValues) =>
                                prevValues.notification?.customWebhookEnabled !==
                                currentValues.notification?.customWebhookEnabled
                            }
                        >
                            {({ getFieldValue }) =>
                                getFieldValue(['notification', 'customWebhookEnabled']) ? (
                                    <Form.Item
                                        label="Webhook URL"
                                        name={['notification', 'customWebhookUrl']}
                                        rules={[
                                            { required: true, message: '请输入自定义 Webhook URL' },
                                            { type: 'url', message: '请输入有效的 URL' },
                                        ]}
                                        tooltip="将发送完整的告警信息 JSON 到此地址"
                                    >
                                        <Input placeholder="https://your-server.com/webhook" />
                                    </Form.Item>
                                ) : null
                            }
                        </Form.Item>
                    </Card>

                    {/* 操作按钮 */}
                    <div className="flex justify-end gap-4 pt-4">
                        <Space>
                            <Button icon={<TestTube size={16} />} loading={testLoading} onClick={handleTest}>
                                测试告警
                            </Button>
                            <Button type="primary" icon={<Save size={16} />} loading={loading} onClick={handleSubmit}>
                                保存配置
                            </Button>
                        </Space>
                    </div>
                </Space>

            </Form>
        </div>
    );
};

export default AlertSettings;
