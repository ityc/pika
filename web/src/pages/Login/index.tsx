import {useState} from 'react';
import {useNavigate} from 'react-router-dom';
import {App, Button, Form, Input} from 'antd';
import {LockOutlined, UserOutlined} from '@ant-design/icons';
import {login} from '../../api/auth';
import type {LoginRequest} from '../../types';

const Login = () => {
    const [loading, setLoading] = useState(false);
    const navigate = useNavigate();
    const {message: messageApi} = App.useApp();

    const onFinish = async (values: LoginRequest) => {
        setLoading(true);
        try {
            const response = await login(values);
            const {token, user} = response.data;

            // 保存 token 和用户信息
            localStorage.setItem('token', token);
            localStorage.setItem('userInfo', JSON.stringify(user));

            messageApi.success('登录成功');
            navigate('/admin/agents');
        } catch (error: any) {
            messageApi.error(error.response?.data?.message || '登录失败，请检查用户名和密码');
        } finally {
            setLoading(false);
        }
    };

    return (
        <div
            className="min-h-screen flex items-center justify-center">
            <div className="w-[360px] p-8 py-12 border rounded-md border-gray-200">
                <div className="text-center mb-10">
                    <h1 className="text-4xl font-semibold mb-2">Pika 探针</h1>
                    <p className="text-base">老鸡专用监控管理平台</p>
                </div>

                <Form
                    name="login"
                    onFinish={onFinish}
                    autoComplete="off"
                >
                    <Form.Item
                        name="username"
                        rules={[{required: true, message: '请输入用户名'}]}
                    >
                        <Input
                            prefix={<UserOutlined/>}
                            placeholder="用户名"
                        />
                    </Form.Item>

                    <Form.Item
                        name="password"
                        rules={[{required: true, message: '请输入密码'}]}
                    >
                        <Input.Password
                            prefix={<LockOutlined/>}
                            placeholder="密码"
                        />
                    </Form.Item>

                    <Form.Item>
                        <Button
                            type="primary"
                            htmlType="submit"
                            loading={loading}
                            block
                        >
                            登录
                        </Button>
                    </Form.Item>
                </Form>
            </div>
        </div>
    );
};

export default Login;

