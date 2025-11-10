// 用户相关
export interface User {
    id: string;
    username: string;
    nickname: string;
    createdAt: number;
    updatedAt: number;
}

export interface LoginRequest {
    username: string;
    password: string;
}

export interface LoginResponse {
    token: string;
    user: User;
}

// 探针相关
export interface Agent {
    id: string;
    name: string;
    hostname: string;
    ip: string;
    os: string;
    arch: string;
    version: string;
    platform?: string;       // 平台
    location?: string;       // 位置
    expireTime?: number;     // 到期时间（时间戳毫秒）
    status: number;
    lastSeenAt: string | number;  // 支持字符串或时间戳
    createdAt?: string;
    updatedAt?: string;
}

export interface AgentInfo {
    name: string;
    hostname: string;
    ip: string;
    os: string;
    arch: string;
    version: string;
}

// 聚合指标数据（所有图表查询只返回聚合数据）
export interface AggregatedCPUMetric {
    timestamp: number;
    avgUsage: number;
    minUsage: number;
    maxUsage: number;
    logicalCores: number;
}

export interface AggregatedMemoryMetric {
    timestamp: number;
    avgUsage: number;
    minUsage: number;
    maxUsage: number;
    total: number;
}

export interface AggregatedNetworkMetric {
    timestamp: number;
    interface: string;
    avgSentRate: number;
    avgRecvRate: number;
    maxSentRate: number;
    maxRecvRate: number;
    totalSent: number;
    totalRecv: number;
}

export interface AggregatedLoadMetric {
    timestamp: number;
    avgLoad1: number;
    avgLoad5: number;
    avgLoad15: number;
    maxLoad1: number;
    maxLoad5: number;
    maxLoad15: number;
}

// 最新实时数据（单点数据，不需要聚合）
export interface CPUMetric {
    id: string;
    agentId: string;
    timestamp: number;
    logicalCores: number;
    physicalCores: number;
    modelName: string;
    usagePercent: number;
}

export interface MemoryMetric {
    id: string;
    agentId: string;
    timestamp: number;
    total: number;
    used: number;
    free: number;
    usagePercent: number;
    swapTotal: number;
    swapUsed: number;
    swapFree: number;
}

export interface LoadMetric {
    id: string;
    agentId: string;
    timestamp: number;
    load1: number;
    load5: number;
    load15: number;
}

// 磁盘汇总数据
export interface DiskSummary {
    avgUsagePercent: number;  // 平均使用率
    totalDisks: number;       // 磁盘数量
    total: number;            // 总容量(字节)
    used: number;             // 已使用(字节)
    free: number;             // 空闲(字节)
}

// 磁盘详细数据
export interface DiskMetric {
    id: string;
    agentId: string;
    timestamp: number;
    device: string;
    mountPoint: string;
    fsType: string;
    total: number;
    used: number;
    free: number;
    usagePercent: number;
}

// 网络详细数据
export interface NetworkMetric {
    id: string;
    agentId: string;
    timestamp: number;
    interface: string;
    bytesSent: number;
    bytesRecv: number;
    packetsSent: number;
    packetsRecv: number;
}

// 网络汇总数据
export interface NetworkSummary {
    totalBytesSentRate: number;   // 总发送速率(字节/秒)
    totalBytesRecvRate: number;   // 总接收速率(字节/秒)
    totalBytesSentTotal: number;  // 累计总发送流量
    totalBytesRecvTotal: number;  // 累计总接收流量
    totalInterfaces: number;      // 网卡数量
}

// 主机信息指标
export interface HostMetric {
    id: number;
    agentId: string;
    hostname: string;
    os: string;
    platform: string;
    platformVersion: string;
    kernelVersion: string;
    kernelArch: string;
    uptime: number;          // 运行时间(秒)
    bootTime: number;        // 启动时间(Unix时间戳-秒)
    procs: number;           // 进程数
    timestamp: number;       // 时间戳（毫秒）
}

export interface LatestMetrics {
    cpu?: CPUMetric;
    memory?: MemoryMetric;
    disk?: DiskSummary;       // 改为汇总数据
    network?: NetworkSummary; // 改为汇总数据
    load?: LoadMetric;
    host?: HostMetric;        // 主机信息
}

// 用户管理相关
export interface CreateUserRequest {
    username: string;
    nickname: string;
    password: string;
}

export interface UpdateUserRequest {
    nickname?: string;
}

export interface ChangePasswordRequest {
    oldPassword: string;
    newPassword: string;
}

// API Key 相关
export interface ApiKey {
    id: string;
    name: string;
    key: string;
    enabled: boolean;
    createdBy: string;
    createdAt: number;
    updatedAt: number;
}

export interface GenerateApiKeyRequest {
    name: string;
}

export interface UpdateApiKeyNameRequest {
    name: string;
}

// 告警配置相关
export interface AlertRules {
    cpuEnabled: boolean;
    cpuThreshold: number;
    cpuDuration: number;
    memoryEnabled: boolean;
    memoryThreshold: number;
    memoryDuration: number;
    diskEnabled: boolean;
    diskThreshold: number;
    diskDuration: number;
    networkEnabled: boolean;
    networkDuration: number;
}

export interface NotificationConfig {
    dingTalkEnabled: boolean;
    dingTalkWebhook: string;
    dingTalkSecret: string;
    weComEnabled: boolean;
    weComWebhook: string;
    feishuEnabled: boolean;
    feishuWebhook: string;
    emailEnabled: boolean;
    emailAddresses: string[];
    customWebhookEnabled: boolean;
    customWebhookUrl: string;
}

export interface AlertConfig {
    id?: string;
    agentId: string;
    agentIds?: string[]; // 监控的探针列表，空数组表示监控所有
    name: string;
    enabled: boolean;
    rules: AlertRules;
    notification: NotificationConfig;
    createdAt?: number;
    updatedAt?: number;
}

export interface AlertRecord {
    id: number;
    agentId: string;
    configId: string;
    configName: string;
    alertType: string;
    message: string;
    threshold: number;
    actualValue: number;
    level: string;
    status: string;
    firedAt: number;
    resolvedAt?: number;
    createdAt: number;
    updatedAt: number;
}
