import { get, post, put } from './request';
import type { Agent, LatestMetrics } from '../types';

export interface ListAgentsResponse {
    items: Agent[];
    total: number;
}

export interface GetAgentMetricsRequest {
    agentId: string;
    type: 'cpu' | 'memory' | 'disk' | 'network' | 'load';
    range?: '1h' | '6h' | '12h' | '24h' | '1d' | '3d' | '7d' | '1w' | '30d' | '1M' | '90d' | '3M';
}

export interface GetAgentMetricsResponse {
    agentId: string;
    type: string;
    range: string;
    start: number;
    end: number;
    interval: number;
    metrics: any[];
}

// 管理员接口 - 获取所有探针（需要认证）
export const getAgentPaging = (pageIndex: number = 1, pageSize: number = 10, hostname?: string, ip?: string, status?: string) => {
    const params = new URLSearchParams();
    params.append('pageIndex', pageIndex.toString());
    params.append('pageSize', pageSize.toString());
    if (hostname) {
        params.append('hostname', hostname);
    }
    if (ip) {
        params.append('ip', ip);
    }
    if (status) {
        params.append('status', status);
    }
    return get<ListAgentsResponse>(`/admin/agents?${params.toString()}`);
};

export const listAgents = () => {
    return get<ListAgentsResponse>('/agents');
};

// 获取所有探针（不分页，用于选择器）
export const getAgents = async () => {
    const response = await get<ListAgentsResponse>('/api/agents');
    return response.data;
};

export const getAgent = (id: string) => {
    return get<Agent>(`/agents/${id}`);
};

// 管理员接口 - 获取探针详情（显示完整信息）
export const getAgentForAdmin = (id: string) => {
    return get<Agent>(`/admin/agents/${id}`);
};

export const getAgentMetrics = (params: GetAgentMetricsRequest) => {
    const { agentId, type, range = '1h' } = params;
    const query = new URLSearchParams();
    query.append('type', type);
    query.append('range', range);
    return get<GetAgentMetricsResponse>(`/agents/${agentId}/metrics?${query.toString()}`);
};

export const getAgentLatestMetrics = (agentId: string) => {
    return get<LatestMetrics>(`/agents/${agentId}/metrics/latest`);
};

// VPS 安全审计相关接口

export interface SystemInfo {
    hostname: string;
    os: string;
    kernelVersion: string;
    uptime: number;
    publicIP?: string;
}

export interface Evidence {
    fileHash?: string;
    processTree?: string[];
    filePath?: string;
    timestamp?: number;
    networkConn?: string;
    riskLevel?: 'low' | 'medium' | 'high';
}

export interface SecurityCheckSub {
    name: string;
    status: 'pass' | 'fail' | 'warn' | 'skip';
    message: string;
    evidence?: Evidence;
}

export interface SecurityCheck {
    category: string;
    status: 'pass' | 'fail' | 'warn' | 'skip';
    message: string;
    details?: SecurityCheckSub[];
}

export interface VPSAuditResult {
    systemInfo: SystemInfo;
    securityChecks: SecurityCheck[];
    startTime: number;
    endTime: number;
    riskScore: number;
    threatLevel: 'low' | 'medium' | 'high' | 'critical';
    recommendations?: string[];
}

export interface AuditResultSummary {
    id: number;
    agentId: string;
    type: string;
    startTime: number;
    endTime: number;
    createdAt: number;
    passCount: number;
    failCount: number;
    warnCount: number;
    totalCount: number;
    systemInfo: SystemInfo;
}

export interface SendCommandResponse {
    commandId: string;
    status: string;
}

// 发送审计指令
export const sendAuditCommand = (agentId: string) => {
    return post<SendCommandResponse>(`/admin/agents/${agentId}/command?type=vps_audit`, {});
};

// 获取最新的审计结果（管理员接口）
export const getAuditResult = (agentId: string) => {
    return get<VPSAuditResult>(`/admin/agents/${agentId}/audit/result`);
};

// 获取审计结果列表（管理员接口）
export const listAuditResults = (agentId: string) => {
    return get<{ items: AuditResultSummary[]; total: number }>(`/admin/agents/${agentId}/audit/results`);
};

// 更新探针名称
export const updateAgentName = (agentId: string, name: string) => {
    return put(`/admin/agents/${agentId}/name`, { name });
};

// 更新探针信息（名称、平台、位置、到期时间）
export interface UpdateAgentInfoRequest {
    name?: string;
    platform?: string;
    location?: string;
    expireTime?: number;
}

export const updateAgentInfo = (agentId: string, data: UpdateAgentInfoRequest) => {
    return put(`/admin/agents/${agentId}`, data);
};

// 获取探针统计数据
export interface AgentStatistics {
    total: number;
    online: number;
    offline: number;
    onlineRate: number;
}

export const getAgentStatistics = () => {
    return get<AgentStatistics>('/admin/agents/statistics');
};

// 获取服务端版本信息
export interface VersionInfo {
    version: string;
}

export const getServerVersion = () => {
    return get<VersionInfo>('/agent/version');
};

