import {useRef, useState} from 'react';
import type {ActionType, ProColumns} from '@ant-design/pro-components';
import {ProTable} from '@ant-design/pro-components';
import {App, Button, Form, Input, Modal, Popconfirm} from 'antd';
import {Edit, Key, Plus, RefreshCw, Trash2} from 'lucide-react';
import {createUser, deleteUser, listUsers, resetPassword, updateUser} from '../../api/user';
import type {CreateUserRequest, UpdateUserRequest, User} from '../../types';
import dayjs from 'dayjs';
import {getErrorMessage} from '../../lib/utils';
import {PageHeader} from '../../components';

const UserList = () => {
    const {message: messageApi, modal} = App.useApp();
    const actionRef = useRef<ActionType>(null);
    const [submitting, setSubmitting] = useState(false);
    const [isModalVisible, setIsModalVisible] = useState(false);
    const [editingUser, setEditingUser] = useState<User | null>(null);
    const [form] = Form.useForm();

    const handleCreate = () => {
        setEditingUser(null);
        setIsModalVisible(true);
        form.resetFields();
    };

    const handleEdit = (user: User) => {
        setEditingUser(user);
        form.setFieldsValue({
            username: user.username,
            nickname: user.nickname,
        });
        setIsModalVisible(true);
    };

    const handleDelete = async (userId: string) => {
        try {
            await deleteUser(userId);
            messageApi.success('删除成功');
            actionRef.current?.reload();
        } catch (error: unknown) {
            messageApi.error(getErrorMessage(error, '删除失败'));
        }
    };

    const handleResetPassword = (user: User) => {
        modal.confirm({
            title: `重置 ${user.username} 的密码`,
            content: (
                <Form
                    id="resetPasswordForm"
                    layout="vertical"
                    onFinish={async (values) => {
                        try {
                            await resetPassword(user.id, values.newPassword);
                            messageApi.success('密码重置成功');
                            Modal.destroyAll();
                        } catch (error: unknown) {
                            messageApi.error(getErrorMessage(error, '密码重置失败'));
                        }
                    }}
                >
                    <Form.Item
                        label="新密码"
                        name="newPassword"
                        rules={[
                            {required: true, message: '请输入新密码'},
                            {min: 6, message: '密码至少6位'},
                        ]}
                    >
                        <Input.Password placeholder="请输入新密码"/>
                    </Form.Item>
                </Form>
            ),
            okText: '重置',
            cancelText: '取消',
            onOk: () =>
                new Promise((resolve) => {
                    const formElement = document.getElementById('resetPasswordForm') as HTMLFormElement | null;
                    if (formElement) {
                        formElement.dispatchEvent(new Event('submit', {cancelable: true, bubbles: true}));
                    }
                    resolve(undefined);
                }),
        });
    };

    const handleModalOk = async () => {
        try {
            const values = await form.validateFields();
            const nickname = values.nickname?.trim();

            if (!nickname) {
                messageApi.warning('昵称不能为空');
                return;
            }

            const isEdit = Boolean(editingUser);
            let request: Promise<unknown> | null = null;

            if (editingUser) {
                if (nickname === editingUser.nickname) {
                    messageApi.info('昵称未发生变化');
                    return;
                }

                const updateData: UpdateUserRequest = {nickname};
                request = updateUser(editingUser.id, updateData);
            } else {
                const username = values.username.trim();
                if (!username) {
                    messageApi.warning('用户名不能为空');
                    return;
                }

                const createData: CreateUserRequest = {
                    username,
                    nickname,
                    password: values.password,
                };
                request = createUser(createData);
            }

            if (!request) {
                return;
            }

            setSubmitting(true);
            await request;
            messageApi.success(isEdit ? '更新成功' : '创建成功');

            setIsModalVisible(false);
            form.resetFields();
            actionRef.current?.reload();
        } catch (error: unknown) {
            if (typeof error === 'object' && error !== null && 'errorFields' in error) {
                return;
            }
            messageApi.error(getErrorMessage(error, '操作失败'));
        } finally {
            setSubmitting(false);
        }
    };

    const columns: ProColumns<User>[] = [
        {
            title: '用户名',
            dataIndex: 'username',
            key: 'username',
            render: (text) => <span className="font-medium text-gray-900">{text}</span>,
        },
        {
            title: '昵称',
            dataIndex: 'nickname',
            key: 'nickname',
            hideInSearch: true,
            render: (text) => <span className="text-gray-600">{text || '未设置'}</span>,
        },
        {
            title: '创建时间',
            dataIndex: 'createdAt',
            key: 'createdAt',
            hideInSearch: true,
            render: (value: number) => (
                <span className="text-gray-600">{dayjs(value).format('YYYY-MM-DD HH:mm')}</span>
            ),
            width: 180,
        },
        {
            title: '最近更新',
            dataIndex: 'updatedAt',
            key: 'updatedAt',
            hideInSearch: true,
            render: (value: number) => (
                <span className="text-gray-600">{dayjs(value).format('YYYY-MM-DD HH:mm')}</span>
            ),
            width: 180,
        },
        {
            title: '操作',
            key: 'action',
            valueType: 'option',
            width: 200,
            render: (_, record) => [
                <Button type="link"
                        key="edit"
                        size="small"
                        icon={<Edit size={14}/>}
                        onClick={() => handleEdit(record)}
                        style={{margin: 0, padding: 0}}
                >
                    编辑
                </Button>,
                <Button type="link"
                        key="reset"
                        size="small"
                        icon={<Key size={14}/>}
                        onClick={() => handleResetPassword(record)}
                        style={{margin: 0, padding: 0}}
                >
                    重置密码
                </Button>,
                <Popconfirm
                    key="delete"
                    title="确定要删除这个用户吗?"
                    onConfirm={() => handleDelete(record.id)}
                    okText="确定"
                    cancelText="取消"
                >
                    <Button type="link"
                            size="small"
                            danger
                            icon={<Trash2 size={14}/>}
                            style={{margin: 0, padding: 0}}
                    >
                        删除
                    </Button>
                </Popconfirm>,
            ],
        },
    ];

    return (
        <div className="space-y-6">
            {/* 页面头部 */}
            <PageHeader
                title="用户管理"
                description="管理系统用户账号和权限"
                actions={[
                    {
                        key: 'refresh',
                        label: '刷新',
                        icon: <RefreshCw size={16}/>,
                        onClick: () => actionRef.current?.reload(),
                    },
                    {
                        key: 'create',
                        label: '新建用户',
                        icon: <Plus size={16}/>,
                        type: 'primary',
                        onClick: handleCreate,
                    },
                ]}
            />

            {/* 用户列表 */}
            <div className="rounded-md p-4 border border-gray-200">
                <ProTable<User>
                    actionRef={actionRef}
                    rowKey="id"
                    search={{labelWidth: 80}}
                    columns={columns}
                    pagination={{
                        defaultPageSize: 10,
                        showSizeChanger: true,
                    }}
                    options={false}
                    request={async (params) => {
                        const {current = 1, pageSize = 10, username} = params;
                        try {
                            const response = await listUsers(current, pageSize, username);
                            const items = response.data.items || [];
                            return {
                                data: items,
                                success: true,
                                total: response.data.total,
                            };
                        } catch (error: unknown) {
                            messageApi.error(getErrorMessage(error, '获取用户列表失败'));
                            return {
                                data: [],
                                success: false,
                            };
                        }
                    }}
                />
            </div>

            {/* 新建/编辑用户弹窗 */}
            <Modal
                title={editingUser ? '编辑用户' : '新建用户'}
                open={isModalVisible}
                onOk={handleModalOk}
                onCancel={() => {
                    setIsModalVisible(false);
                    form.resetFields();
                }}
                okText={editingUser ? '保存' : '创建'}
                cancelText="取消"
                confirmLoading={submitting}
                destroyOnClose
            >
                <Form form={form} layout="vertical" autoComplete="off">
                    <Form.Item
                        label="用户名"
                        name="username"
                        rules={[
                            {required: true, message: '请输入用户名'},
                            {min: 3, message: '用户名至少3位'},
                        ]}
                    >
                        <Input placeholder="请输入用户名" disabled={!!editingUser}/>
                    </Form.Item>

                    <Form.Item
                        label="昵称"
                        name="nickname"
                        rules={[{required: true, message: '请输入昵称'}]}
                    >
                        <Input placeholder="请输入昵称"/>
                    </Form.Item>

                    {!editingUser && (
                        <Form.Item
                            label="密码"
                            name="password"
                            rules={[
                                {required: true, message: '请输入密码'},
                                {min: 6, message: '密码至少6位'},
                            ]}
                        >
                            <Input.Password placeholder="请输入密码"/>
                        </Form.Item>
                    )}
                </Form>
            </Modal>
        </div>
    );
};

export default UserList;
