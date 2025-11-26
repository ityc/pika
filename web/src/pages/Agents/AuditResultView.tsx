import {Card, Collapse, List, Space, Tag, Descriptions, Tabs, Table, Empty, Alert} from 'antd';
import {
    CheckCircle,
    XCircle,
    AlertTriangle,
    MinusCircle,
    Server,
    Network,
    Cpu,
    Users,
    FileText,
    Shield,
    Activity,
    Calendar,
    Settings,
    PlayCircle,
    AlertOctagon,
    LogIn
} from 'lucide-react';
import type {VPSAuditResult, ProcessInfo, FileInfo, LoginRecord, LoginSession, SSHKeyInfo, SudoUserInfo} from '../../api/agent';
import dayjs from 'dayjs';
import duration from 'dayjs/plugin/duration';

dayjs.extend(duration);

const {Panel} = Collapse;

interface RiskItem {
    level: 'critical' | 'high' | 'medium' | 'low';
    title: string;
    description: string;
}

interface AuditResultViewProps {
    result: VPSAuditResult;
}

const AuditResultView = ({result}: AuditResultViewProps) => {
    const analyzeRisks = (result: VPSAuditResult): RiskItem[] => {
        const risks: RiskItem[] = [];

        // 1. Suspicious Processes
        if (result.assetInventory.processAssets?.suspiciousProcesses?.length) {
            risks.push({
                level: 'critical',
                title: '发现可疑进程',
                description: `发现 ${result.assetInventory.processAssets.suspiciousProcesses.length} 个可执行文件已删除的进程`
            });
        }

        // 2. Tmp Executables
        if (result.assetInventory.fileAssets?.tmpExecutables?.length) {
            risks.push({
                level: 'high',
                title: '发现临时目录可执行文件',
                description: `在 /tmp 或 /dev/shm 目录下发现 ${result.assetInventory.fileAssets.tmpExecutables.length} 个可执行文件`
            });
        }

        // 3. Root Equivalent Users
        const rootEquivUsers = result.assetInventory.userAssets?.systemUsers?.filter(u => u.isRootEquiv);
        if (rootEquivUsers?.length) {
            risks.push({
                level: 'high',
                title: '存在Root权限用户',
                description: `发现 ${rootEquivUsers.length} 个UID为0的非root用户: ${rootEquivUsers.map(u => u.username).join(', ')}`
            });
        }

        // 4. Loginable Users with No Password
        const noPwdUsers = result.assetInventory.userAssets?.systemUsers?.filter(u => u.isLoginable && !u.hasPassword);
        if (noPwdUsers?.length) {
            risks.push({
                level: 'medium',
                title: '存在无密码可登录用户',
                description: `发现 ${noPwdUsers.length} 个用户无密码且可登录: ${noPwdUsers.map(u => u.username).join(', ')}`
            });
        }

        // 5. Failed Login Attempts
        const failedLogins = result.assetInventory.loginAssets?.failedLogins?.length || 0;
        if (failedLogins > 50) {
            risks.push({
                level: 'high',
                title: '存在大量失败登录尝试',
                description: `发现 ${failedLogins} 次失败登录尝试，可能存在暴力破解攻击`
            });
        } else if (failedLogins > 20) {
            risks.push({
                level: 'medium',
                title: '存在失败登录尝试',
                description: `发现 ${failedLogins} 次失败登录尝试`
            });
        }

        // 6. High Frequency Login IPs
        if (result.assetInventory.loginAssets?.successfulLogins) {
            const ipCount: Record<string, number> = {};
            result.assetInventory.loginAssets.successfulLogins.forEach(login => {
                if (login.ip && login.ip !== 'localhost') {
                    ipCount[login.ip] = (ipCount[login.ip] || 0) + 1;
                }
            });

            const highFreqIPs = Object.entries(ipCount).filter(([_, count]) => count > 30);
            if (highFreqIPs.length > 0) {
                risks.push({
                    level: 'medium',
                    title: '发现高频登录IP',
                    description: `以下IP登录次数异常频繁: ${highFreqIPs.map(([ip, count]) => `${ip}(${count}次)`).join(', ')}`
                });
            }
        }

        return risks;
    };

    const risks = analyzeRisks(result);

    const getStatusIcon = (status: string) => {
        switch (status) {
            case 'pass':
                return <CheckCircle size={16} className="text-green-500"/>;
            case 'fail':
                return <XCircle size={16} className="text-red-500"/>;
            case 'warn':
                return <AlertTriangle size={16} className="text-yellow-500"/>;
            case 'skip':
                return <MinusCircle size={16} className="text-gray-400"/>;
            default:
                return null;
        }
    };

    const getStatusTag = (status: string) => {
        const configs = {
            pass: {color: 'success', text: '通过'},
            fail: {color: 'error', text: '失败'},
            warn: {color: 'warning', text: '警告'},
            skip: {color: 'default', text: '跳过'},
        };
        const config = configs[status as keyof typeof configs] || {color: 'default', text: status};
        return <Tag color={config.color}>{config.text}</Tag>;
    };

    const getSeverityTag = (severity?: string) => {
        if (!severity) return null;
        const configs = {
            high: {color: 'error', text: '高危'},
            medium: {color: 'warning', text: '中危'},
            low: {color: 'default', text: '低危'},
        };
        const config = configs[severity as keyof typeof configs];
        return config ? <Tag color={config.color}>{config.text}</Tag> : null;
    };

    const getThreatLevelTag = (level: string) => {
        const configs = {
            critical: {color: 'error', text: '危急'},
            high: {color: 'error', text: '高危'},
            medium: {color: 'warning', text: '中危'},
            low: {color: 'success', text: '低危'},
        };
        const config = configs[level as keyof typeof configs] || {color: 'default', text: level};
        return <Tag color={config.color} className="text-lg px-3 py-1">{config.text}</Tag>;
    };

    const formatUptime = (seconds: number) => {
        const d = dayjs.duration(seconds, 'seconds');
        const days = Math.floor(d.asDays());
        const hours = d.hours();
        const minutes = d.minutes();
        return `${days}天 ${hours}小时 ${minutes}分钟`;
    };

    const portColumns = [
        {title: '协议', dataIndex: 'protocol', key: 'protocol', width: 80},
        {
            title: '地址:端口',
            key: 'address',
            render: (record: any) => `${record.address}:${record.port}`,
        },
        {
            title: '公网暴露',
            dataIndex: 'isPublic',
            key: 'isPublic',
            width: 100,
            render: (isPublic: boolean) => isPublic ?
                <Tag color="warning">是</Tag> : <Tag color="default">否</Tag>
        },
        {title: '进程', dataIndex: 'processName', key: 'processName'},
        {title: 'PID', dataIndex: 'processPid', key: 'processPid', width: 80},
    ];

    const connectionColumns = [
        {title: '协议', dataIndex: 'protocol', key: 'protocol', width: 80},
        {
            title: '本地地址',
            key: 'local',
            render: (record: any) => `${record.localAddr || ''}:${record.localPort || ''}`,
        },
        {
            title: '远程地址',
            key: 'remote',
            render: (record: any) => `${record.remoteAddr || ''}:${record.remotePort || ''}`,
        },
        {title: '状态', dataIndex: 'state', key: 'state', width: 120},
        {title: 'PID', dataIndex: 'processPid', key: 'processPid', width: 80},
        {title: '进程', dataIndex: 'processName', key: 'processName'},
    ];

    const processColumns = [
        {title: 'PID', dataIndex: 'pid', key: 'pid', width: 80},
        {title: '进程名', dataIndex: 'name', key: 'name'},
        {title: '用户', dataIndex: 'username', key: 'username', width: 100},
        {
            title: 'CPU %',
            dataIndex: 'cpuPercent',
            key: 'cpuPercent',
            width: 100,
            render: (val: number) => `${val.toFixed(2)}%`
        },
        {
            title: '内存 %',
            dataIndex: 'memPercent',
            key: 'memPercent',
            width: 100,
            render: (val: number) => `${val.toFixed(2)}%`
        },
        {
            title: '内存 (MB)',
            dataIndex: 'memoryMb',
            key: 'memoryMb',
            width: 120,
            render: (val: number) => `${val} MB`
        },
    ];

    const userColumns = [
        {title: '用户名', dataIndex: 'username', key: 'username'},
        {title: 'UID', dataIndex: 'uid', key: 'uid', width: 80},
        {title: 'GID', dataIndex: 'gid', key: 'gid', width: 80},
        {title: 'Shell', dataIndex: 'shell', key: 'shell'},
        {
            title: '可登录',
            dataIndex: 'isLoginable',
            key: 'isLoginable',
            width: 100,
            render: (val: boolean) => val ? <Tag color="success">是</Tag> : <Tag color="default">否</Tag>
        },
        {
            title: 'Root权限',
            dataIndex: 'isRootEquiv',
            key: 'isRootEquiv',
            width: 100,
            render: (val: boolean) => val ? <Tag color="error">是</Tag> : <Tag color="default">否</Tag>
        },
    ];

    const loginRecordColumns = [
        {title: '用户名', dataIndex: 'username', key: 'username', width: 120},
        {title: 'IP地址', dataIndex: 'ip', key: 'ip', width: 150},
        {title: '终端', dataIndex: 'terminal', key: 'terminal', width: 100},
        {
            title: '登录时间',
            dataIndex: 'timestamp',
            key: 'timestamp',
            render: (val: number) => dayjs(val).format('YYYY-MM-DD HH:mm:ss')
        },
        {
            title: '状态',
            dataIndex: 'status',
            key: 'status',
            width: 80,
            render: (status: string) => status === 'success' ?
                <Tag color="success">成功</Tag> : <Tag color="error">失败</Tag>
        },
    ];

    const currentSessionColumns = [
        {title: '用户名', dataIndex: 'username', key: 'username', width: 120},
        {title: 'IP地址', dataIndex: 'ip', key: 'ip', width: 150},
        {title: '终端', dataIndex: 'terminal', key: 'terminal', width: 100},
        {
            title: '登录时间',
            dataIndex: 'loginTime',
            key: 'loginTime',
            render: (val: number) => dayjs(val).format('YYYY-MM-DD HH:mm:ss')
        },
        {
            title: '空闲时间',
            dataIndex: 'idleTime',
            key: 'idleTime',
            width: 100,
            render: (seconds: number) => {
                if (seconds === 0) return '-';
                const hours = Math.floor(seconds / 3600);
                const minutes = Math.floor((seconds % 3600) / 60);
                return hours > 0 ? `${hours}h ${minutes}m` : `${minutes}m`;
            }
        },
    ];

    const sshKeyColumns = [
        {title: '用户', dataIndex: 'username', key: 'username', width: 120},
        {title: '密钥类型', dataIndex: 'keyType', key: 'keyType', width: 120},
        {title: '指纹', dataIndex: 'fingerprint', key: 'fingerprint', ellipsis: true},
        {title: '注释', dataIndex: 'comment', key: 'comment', ellipsis: true},
        {
            title: '添加时间',
            dataIndex: 'addedTime',
            key: 'addedTime',
            width: 180,
            render: (val: number) => val ? dayjs(val).format('YYYY-MM-DD HH:mm:ss') : '-'
        },
    ];

    const sudoUserColumns = [
        {title: '用户名', dataIndex: 'username', key: 'username', width: 150},
        {
            title: '免密码',
            dataIndex: 'noPasswd',
            key: 'noPasswd',
            width: 100,
            render: (val: boolean) => val ? <Tag color="warning">是</Tag> : <Tag color="default">否</Tag>
        },
        {title: '规则', dataIndex: 'rules', key: 'rules', ellipsis: true},
    ];

    const cronColumns = [
        {title: '用户', dataIndex: 'user', key: 'user', width: 100},
        {title: '计划', dataIndex: 'schedule', key: 'schedule', width: 150},
        {title: '命令', dataIndex: 'command', key: 'command', ellipsis: true},
        {title: '文件路径', dataIndex: 'filePath', key: 'filePath', width: 200, ellipsis: true},
    ];

    const serviceColumns = [
        {title: '服务名', dataIndex: 'name', key: 'name'},
        {title: '状态', dataIndex: 'state', key: 'state', width: 100},
        {
            title: '开机启动',
            dataIndex: 'enabled',
            key: 'enabled',
            width: 100,
            render: (val: boolean) => val ? <Tag color="success">是</Tag> : <Tag color="default">否</Tag>
        },
        {title: '启动命令', dataIndex: 'execStart', key: 'execStart', ellipsis: true},
        {title: '描述', dataIndex: 'description', key: 'description', ellipsis: true},
    ];

    const startupScriptColumns = [
        {title: '类型', dataIndex: 'type', key: 'type', width: 150},
        {title: '名称', dataIndex: 'name', key: 'name'},
        {title: '路径', dataIndex: 'path', key: 'path', ellipsis: true},
        {
            title: '启用',
            dataIndex: 'enabled',
            key: 'enabled',
            width: 80,
            render: (val: boolean) => val ? <Tag color="success">是</Tag> : <Tag color="default">否</Tag>
        },
    ];

    const moduleColumns = [
        {title: '模块名', dataIndex: 'name', key: 'name'},
        {title: '大小', dataIndex: 'size', key: 'size', width: 120, render: (val: number) => `${val} bytes`},
        {title: '被引用次数', dataIndex: 'usedBy', key: 'usedBy', width: 120},
    ];

    return (
        <Space direction="vertical" size="large" style={{width: '100%'}}>
            {/* 风险概览 */}
            {risks.length > 0 && (
                <Card
                    title={<Space><AlertOctagon size={18} className="text-red-500"/><span className="font-semibold text-red-500">风险概览</span></Space>}
                    className="border-red-200 bg-red-50"
                >
                    <List
                        dataSource={risks}
                        renderItem={item => (
                            <List.Item>
                                <List.Item.Meta
                                    avatar={getThreatLevelTag(item.level)}
                                    title={<span className="font-bold">{item.title}</span>}
                                    description={item.description}
                                />
                            </List.Item>
                        )}
                    />
                </Card>
            )}

            {/* 系统信息 */}
            <Card
                title={<Space><Server size={18}/><span className="font-semibold">系统信息</span></Space>}
                variant={'outlined'}
            >
                <Descriptions column={{xs: 1, sm: 2}} bordered>
                    <Descriptions.Item label="主机名">
                        {result.systemInfo.hostname}
                    </Descriptions.Item>
                    <Descriptions.Item label="操作系统">
                        {result.systemInfo.os}
                    </Descriptions.Item>
                    <Descriptions.Item label="内核版本">
                        {result.systemInfo.kernelVersion}
                    </Descriptions.Item>
                    <Descriptions.Item label="运行时长">
                        {formatUptime(result.systemInfo.uptime)}
                    </Descriptions.Item>
                    {result.systemInfo.publicIP && (
                        <Descriptions.Item label="公网IP" span={2}>
                            {result.systemInfo.publicIP}
                        </Descriptions.Item>
                    )}
                    <Descriptions.Item label="采集时间" span={2}>
                        {dayjs(result.startTime).format('YYYY-MM-DD HH:mm:ss')} - {dayjs(result.endTime).format('HH:mm:ss')}
                        <span className="ml-2 text-gray-500">
                            (耗时: {((result.endTime - result.startTime) / 1000).toFixed(2)}秒)
                        </span>
                    </Descriptions.Item>
                </Descriptions>
            </Card>


            {/* 资产清单 */}
            <Card
                title={<Space><Activity size={18}/><span className="font-semibold">资产清单</span></Space>}
                variant={'outlined'}
            >
                <Tabs
                    items={[
                        {
                            key: 'network',
                            label: <Space><Network size={16}/>网络资产</Space>,
                            children: (
                                <Space direction="vertical" size="middle" style={{width: '100%'}}>
                                    <Card size="small" title="监听端口">
                                        {result.assetInventory.networkAssets?.listeningPorts?.length ? (
                                            <Table
                                                size="small"
                                                dataSource={result.assetInventory.networkAssets.listeningPorts}
                                                columns={portColumns}
                                                rowKey={(record, index) => `${record.address}:${record.port}-${index}`}
                                                pagination={false}
                                            />
                                        ) : (
                                            <Empty description="无数据"/>
                                        )}
                                    </Card>
                                    <Card size="small" title="活跃连接">
                                        {result.assetInventory.networkAssets?.connections?.length ? (
                                            <Table
                                                size="small"
                                                dataSource={result.assetInventory.networkAssets.connections}
                                                columns={connectionColumns}
                                                rowKey={(record, index) => `${record.localAddr}:${record.localPort}-${record.remoteAddr}:${record.remotePort}-${index}`}
                                                pagination={false}
                                            />
                                        ) : (
                                            <Empty description="无数据"/>
                                        )}
                                    </Card>
                                    {result.statistics?.networkStats && (
                                        <Card size="small" title="网络统计">
                                            <Descriptions size="small" column={3}>
                                                <Descriptions.Item label="监听端口总数">
                                                    {result.statistics.networkStats.totalListeningPorts || 0}
                                                </Descriptions.Item>
                                                <Descriptions.Item label="公网端口">
                                                    {result.statistics.networkStats.publicListeningPorts || 0}
                                                </Descriptions.Item>
                                                <Descriptions.Item label="活跃连接">
                                                    {result.statistics.networkStats.activeConnections || 0}
                                                </Descriptions.Item>
                                            </Descriptions>
                                        </Card>
                                    )}
                                </Space>
                            ),
                        },
                        {
                            key: 'process',
                            label: <Space><Cpu size={16}/>进程资产</Space>,
                            children: (
                                <Space direction="vertical" size="middle" style={{width: '100%'}}>
                                    {result.assetInventory.processAssets?.suspiciousProcesses && result.assetInventory.processAssets.suspiciousProcesses.length > 0 && (
                                        <Alert
                                            message="发现可疑进程"
                                            description="以下进程的可执行文件已被删除，这通常是恶意软件的迹象（或者是进程更新中）。"
                                            type="error"
                                            showIcon
                                        />
                                    )}
                                    <Card size="small" title={<Space><AlertTriangle size={16} className={result.assetInventory.processAssets?.suspiciousProcesses?.length ? "text-red-500" : ""}/>可疑进程</Space>}>
                                        {result.assetInventory.processAssets?.suspiciousProcesses?.length ? (
                                            <Table
                                                size="small"
                                                dataSource={result.assetInventory.processAssets.suspiciousProcesses}
                                                columns={[
                                                    ...processColumns,
                                                    {
                                                        title: '异常信息',
                                                        key: 'issue',
                                                        render: (record: ProcessInfo) => record.exeDeleted ? <Tag color="error">Exe已删除</Tag> : null
                                                    }
                                                ]}
                                                rowKey="pid"
                                                pagination={false}
                                            />
                                        ) : (
                                            <Empty description="无发现"/>
                                        )}
                                    </Card>
                                    <Card size="small" title="TOP CPU进程">
                                        {result.assetInventory.processAssets?.topCpuProcesses?.length ? (
                                            <Table
                                                size="small"
                                                dataSource={result.assetInventory.processAssets.topCpuProcesses}
                                                columns={processColumns}
                                                rowKey="pid"
                                                pagination={false}
                                            />
                                        ) : (
                                            <Empty description="无数据"/>
                                        )}
                                    </Card>
                                    <Card size="small" title="TOP 内存进程">
                                        {result.assetInventory.processAssets?.topMemoryProcesses?.length ? (
                                            <Table
                                                size="small"
                                                dataSource={result.assetInventory.processAssets.topMemoryProcesses}
                                                columns={processColumns}
                                                rowKey="pid"
                                                pagination={false}
                                            />
                                        ) : (
                                            <Empty description="无数据"/>
                                        )}
                                    </Card>
                                </Space>
                            ),
                        },
                        {
                            key: 'user',
                            label: <Space><Users size={16}/>用户资产</Space>,
                            children: (
                                <Space direction="vertical" size="middle" style={{width: '100%'}}>
                                    <Card size="small" title="系统用户">
                                        {result.assetInventory.userAssets?.systemUsers?.length ? (
                                            <Table
                                                size="small"
                                                dataSource={result.assetInventory.userAssets.systemUsers}
                                                columns={userColumns}
                                                rowKey="username"
                                                pagination={false}
                                            />
                                        ) : (
                                            <Empty description="无数据"/>
                                        )}
                                    </Card>
                                    <Card size="small" title="SSH 密钥">
                                        {result.assetInventory.userAssets?.sshKeys?.length ? (
                                            <Table
                                                size="small"
                                                dataSource={result.assetInventory.userAssets.sshKeys}
                                                columns={sshKeyColumns}
                                                rowKey={(record, index) => `${record.username}-${index}`}
                                                pagination={false}
                                            />
                                        ) : (
                                            <Empty description="未发现 SSH 密钥"/>
                                        )}
                                    </Card>
                                    <Card size="small" title="Sudo 用户">
                                        {result.assetInventory.userAssets?.sudoUsers?.length ? (
                                            <Table
                                                size="small"
                                                dataSource={result.assetInventory.userAssets.sudoUsers}
                                                columns={sudoUserColumns}
                                                rowKey="username"
                                                pagination={false}
                                            />
                                        ) : (
                                            <Empty description="未发现 Sudo 用户"/>
                                        )}
                                    </Card>
                                    <Card size="small" title="当前登录会话">
                                        {result.assetInventory.userAssets?.currentLogins?.length ? (
                                            <Table
                                                size="small"
                                                dataSource={result.assetInventory.userAssets.currentLogins}
                                                columns={currentSessionColumns}
                                                rowKey={(record, index) => `${record.username}-${record.terminal}-${index}`}
                                                pagination={false}
                                            />
                                        ) : (
                                            <Empty description="无当前登录会话"/>
                                        )}
                                    </Card>
                                </Space>
                            ),
                        },
                        {
                            key: 'file',
                            label: <Space><FileText size={16}/>文件资产</Space>,
                            children: (
                                <Space direction="vertical" size="middle" style={{width: '100%'}}>
                                    {result.assetInventory.fileAssets?.tmpExecutables && result.assetInventory.fileAssets.tmpExecutables.length > 0 && (
                                        <Alert
                                            message="发现临时目录可执行文件"
                                            description="在 /tmp 或 /dev/shm 目录发现可执行文件，这通常是恶意软件或webshell的特征。"
                                            type="error"
                                            showIcon
                                        />
                                    )}
                                    <Card size="small" title={<Space><AlertTriangle size={16} className={result.assetInventory.fileAssets?.tmpExecutables?.length ? "text-red-500" : ""}/>临时目录可执行文件</Space>}>
                                        {result.assetInventory.fileAssets?.tmpExecutables?.length ? (
                                            <Table
                                                size="small"
                                                dataSource={result.assetInventory.fileAssets.tmpExecutables}
                                                columns={[
                                                    {title: '路径', dataIndex: 'path', key: 'path'},
                                                    {title: '大小', dataIndex: 'size', key: 'size', render: (val: number) => `${(val / 1024).toFixed(2)} KB`},
                                                    {title: '修改时间', dataIndex: 'modTime', key: 'modTime', render: (val: number) => dayjs(val).format('YYYY-MM-DD HH:mm:ss')},
                                                    {title: '权限', dataIndex: 'permissions', key: 'permissions'},
                                                ]}
                                                rowKey="path"
                                                pagination={false}
                                            />
                                        ) : (
                                            <Empty description="无发现"/>
                                        )}
                                    </Card>
                                    <Card size="small" title={<Space><Calendar size={16}/>定时任务</Space>}>
                                        {result.assetInventory.fileAssets?.cronJobs?.length ? (
                                            <Table
                                                size="small"
                                                dataSource={result.assetInventory.fileAssets.cronJobs}
                                                columns={cronColumns}
                                                rowKey={(record, index) => `${record.user}-${index}`}
                                                pagination={false}
                                            />
                                        ) : (
                                            <Empty description="无定时任务"/>
                                        )}
                                    </Card>
                                    <Card size="small" title={<Space><Settings size={16}/>Systemd服务</Space>}>
                                        {result.assetInventory.fileAssets?.systemdServices?.length ? (
                                            <Table
                                                size="small"
                                                dataSource={result.assetInventory.fileAssets.systemdServices}
                                                columns={serviceColumns}
                                                rowKey="name"
                                                pagination={false}
                                            />
                                        ) : (
                                            <Empty description="无Systemd服务"/>
                                        )}
                                    </Card>
                                    <Card size="small" title={<Space><PlayCircle size={16}/>启动脚本</Space>}>
                                        {result.assetInventory.fileAssets?.startupScripts?.length ? (
                                            <Table
                                                size="small"
                                                dataSource={result.assetInventory.fileAssets.startupScripts}
                                                columns={startupScriptColumns}
                                                rowKey={(record, index) => `${record.path}-${index}`}
                                                pagination={false}
                                            />
                                        ) : (
                                            <Empty description="无启动脚本"/>
                                        )}
                                    </Card>
                                </Space>
                            ),
                        },
                        {
                            key: 'login',
                            label: <Space><LogIn size={16}/>登录日志</Space>,
                            children: (
                                <Space direction="vertical" size="middle" style={{width: '100%'}}>
                                    {result.assetInventory.loginAssets?.failedLogins && result.assetInventory.loginAssets.failedLogins.length > 0 && (
                                        <Alert
                                            message="发现失败登录记录"
                                            description={`系统记录到 ${result.assetInventory.loginAssets.failedLogins.length} 次失败登录尝试，请检查是否存在暴力破解行为。`}
                                            type="warning"
                                            showIcon
                                        />
                                    )}
                                    <Card size="small" title="成功登录历史">
                                        {result.assetInventory.loginAssets?.successfulLogins?.length ? (
                                            <Table
                                                size="small"
                                                dataSource={result.assetInventory.loginAssets.successfulLogins}
                                                columns={loginRecordColumns}
                                                rowKey={(record, index) => `success-${index}`}
                                                pagination={{pageSize: 20}}
                                            />
                                        ) : (
                                            <Empty description="无登录历史"/>
                                        )}
                                    </Card>
                                    <Card size="small" title={<Space><AlertTriangle size={16} className={result.assetInventory.loginAssets?.failedLogins?.length ? "text-yellow-500" : ""}/>失败登录历史</Space>}>
                                        {result.assetInventory.loginAssets?.failedLogins?.length ? (
                                            <Table
                                                size="small"
                                                dataSource={result.assetInventory.loginAssets.failedLogins}
                                                columns={loginRecordColumns}
                                                rowKey={(record, index) => `failed-${index}`}
                                                pagination={{pageSize: 20}}
                                            />
                                        ) : (
                                            <Empty description="无失败登录记录"/>
                                        )}
                                    </Card>
                                    <Card size="small" title="当前登录会话">
                                        {result.assetInventory.loginAssets?.currentSessions?.length ? (
                                            <Table
                                                size="small"
                                                dataSource={result.assetInventory.loginAssets.currentSessions}
                                                columns={currentSessionColumns}
                                                rowKey={(record, index) => `session-${index}`}
                                                pagination={false}
                                            />
                                        ) : (
                                            <Empty description="无当前登录会话"/>
                                        )}
                                    </Card>
                                </Space>
                            ),
                        },
                        {
                            key: 'kernel',
                            label: <Space><Shield size={16}/>内核信息</Space>,
                            children: (
                                <Space direction="vertical" size="middle" style={{width: '100%'}}>
                                    <Card size="small" title="已加载内核模块">
                                        {result.assetInventory.kernelAssets?.loadedModules?.length ? (
                                            <Table
                                                size="small"
                                                dataSource={result.assetInventory.kernelAssets.loadedModules}
                                                columns={moduleColumns}
                                                rowKey="name"
                                                pagination={false}
                                            />
                                        ) : (
                                            <Empty description="无已加载模块"/>
                                        )}
                                    </Card>
                                    {result.assetInventory.kernelAssets?.securityModules && (
                                        <Card size="small" title="安全模块状态">
                                            <Descriptions size="small" column={1}>
                                                <Descriptions.Item label="SELinux">
                                                    <Tag color={
                                                        result.assetInventory.kernelAssets.securityModules.selinuxStatus === 'enforcing' ? 'success' :
                                                            result.assetInventory.kernelAssets.securityModules.selinuxStatus === 'permissive' ? 'warning' : 'default'
                                                    }>
                                                        {result.assetInventory.kernelAssets.securityModules.selinuxStatus || 'unknown'}
                                                    </Tag>
                                                </Descriptions.Item>
                                                <Descriptions.Item label="AppArmor">
                                                    <Tag color={
                                                        result.assetInventory.kernelAssets.securityModules.apparmorStatus === 'enabled' ? 'success' : 'default'
                                                    }>
                                                        {result.assetInventory.kernelAssets.securityModules.apparmorStatus || 'unknown'}
                                                    </Tag>
                                                </Descriptions.Item>
                                                <Descriptions.Item label="Secure Boot">
                                                    <Tag color={
                                                        result.assetInventory.kernelAssets.securityModules.secureBootState === 'enabled' ? 'success' : 'default'
                                                    }>
                                                        {result.assetInventory.kernelAssets.securityModules.secureBootState || 'unknown'}
                                                    </Tag>
                                                </Descriptions.Item>
                                            </Descriptions>
                                        </Card>
                                    )}
                                    {result.assetInventory.kernelAssets?.kernelParameters && (
                                        <Card size="small" title="关键内核参数">
                                            <Descriptions size="small" column={1}>
                                                {Object.entries(result.assetInventory.kernelAssets.kernelParameters).map(([key, value]) => (
                                                    <Descriptions.Item key={key} label={key}>
                                                        <code>{value}</code>
                                                    </Descriptions.Item>
                                                ))}
                                            </Descriptions>
                                        </Card>
                                    )}
                                </Space>
                            ),
                        },
                    ]}
                />
            </Card>

            {/* 采集警告 */}
            {result.collectWarnings && result.collectWarnings.length > 0 && (
                <Alert
                    type="warning"
                    message="采集警告"
                    description={
                        <ul className="list-disc pl-5">
                            {result.collectWarnings.map((warning, index) => (
                                <li key={index}>{warning}</li>
                            ))}
                        </ul>
                    }
                />
            )}
        </Space>
    );
};

export default AuditResultView;
