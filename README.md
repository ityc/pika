# Pika 探针监控系统

<div align="center">

一个基于 Go + PostgreSQL 的实时探针监控系统

[快速开始](#快速开始) • [功能特性](#功能特性) • [文档](#文档) • [架构](#架构)

</div>

## 简介

Pika 是一个轻量级的探针监控系统，支持实时数据采集、存储和查询。系统采用 WebSocket 进行探针与服务端的通信，使用 PostgreSQL 存储时序数据，提供完整的 RESTful API 和用户管理功能。

## 功能特性

- ✅ **实时通信**: 基于 WebSocket 的探针通信，支持实时数据上报
- ✅ **多指标支持**: CPU、内存、磁盘、网络、负载等多种系统指标
- ✅ **时序存储**: PostgreSQL 存储时序数据，自动清理过期数据（30天）
- ✅ **用户管理**: 完整的用户 CRUD、角色管理、密码加密
- ✅ **用户认证**: Session 管理和 Token 认证
- ✅ **RESTful API**: 提供完整的 API 接口
- ✅ **依赖注入**: 使用 Wire 进行依赖管理
- ✅ **自动重连**: 探针支持自动重连和心跳检测
- ✅ **易于扩展**: 清晰的分层架构，易于添加新功能
- ✅ **psutil 采集**: 探针使用 gopsutil 库进行系统信息采集

## 快速开始

### 前置要求

- Go 1.25+
- Docker & Docker Compose
- Make

### 一键启动

```bash
# 克隆项目
git clone <repository-url>
cd pika

# 启动服务（自动启动数据库、编译、运行）
./start.sh
```

服务将在 `http://localhost:18888` 启动

### 手动启动

```bash
# 1. 启动数据库
docker-compose up -d

# 2. 编译项目
make build-backend

# 3. 运行服务端
./bin/pika

# 4. 运行探针客户端（可选，用于测试）
./bin/agent
```

### 测试 API

```bash
./test_api.sh
```

## 技术栈

| 类别 | 技术 |
|------|------|
| 语言 | Go 1.25 |
| Web 框架 | Echo v4 |
| 数据库 | PostgreSQL |
| ORM | GORM |
| WebSocket | Gorilla WebSocket |
| 依赖注入 | Google Wire |
| 日志 | Uber Zap |
| 配置 | Viper |
| 系统信息采集 | gopsutil v4 |
| 密码加密 | bcrypt |

## 架构

```
┌─────────────────────────────────────┐
│         Web/WebSocket Layer         │
│    (Echo + Gorilla WebSocket)       │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│         Handler Layer               │
│  (account, agent, user handlers)    │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│         Service Layer               │
│  (account, agent, user services)    │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│       Repository Layer              │
│  (session, agent, user, metric)     │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│         Database (PostgreSQL)       │
└─────────────────────────────────────┘
```

## API 接口

### 认证相关
- `POST /api/auth` - Token 登录
- `POST /api/login` - 用户名密码登录
- `GET /api/account/info` - 获取用户信息
- `POST /api/logout` - 退出登录

### 探针管理
- `GET /api/agents` - 获取所有探针
- `GET /api/agents/online` - 获取在线探针
- `GET /api/agents/:id` - 获取探针详情
- `GET /api/agents/:id/metrics` - 获取指标数据（支持时间戳查询）
- `GET /api/agents/:id/metrics/latest` - 获取最新指标

### 用户管理
- `GET /api/users` - 获取用户列表（分页）
- `POST /api/users` - 创建用户
- `GET /api/users/:id` - 获取用户详情
- `PUT /api/users/:id` - 更新用户
- `DELETE /api/users/:id` - 删除用户
- `POST /api/users/:id/password` - 修改密码
- `POST /api/users/:id/reset-password` - 重置密码（管理员）
- `POST /api/users/:id/status` - 更新用户状态

### WebSocket
- `GET /ws/agent` - 探针连接

## 数据模型

### 用户表 (users)
- id (主键, UUID)
- username (唯一索引)
- password (bcrypt 加密)
- nickname
- email
- phone
- avatar
- role (admin/user)
- status (0-禁用, 1-启用)
- created_at (时间戳毫秒)
- updated_at (时间戳毫秒)

### 探针表 (agents)
- id (主键, UUID)
- name
- hostname
- ip
- os
- arch
- version
- status (0-离线, 1-在线)
- last_seen_at (时间戳毫秒)
- created_at (时间戳毫秒)
- updated_at (时间戳毫秒)

### 指标表
所有指标表都使用 `int64` 类型的时间戳（毫秒）：
- cpu_metrics
- memory_metrics
- disk_metrics
- network_metrics
- load_metrics

## 项目结构

```
pika/
├── cmd/                    # 应用程序入口
│   ├── serv/              # 服务端
│   └── agent/             # 探针客户端
├── internal/              # 内部代码
│   ├── handler/           # HTTP/WebSocket 处理器
│   ├── service/           # 业务逻辑
│   ├── repo/              # 数据访问
│   ├── models/            # 数据模型
│   └── websocket/         # WebSocket 管理
├── web/                   # 前端资源
├── config.yaml            # 配置文件
├── docker-compose.yml     # Docker 配置
├── Makefile              # 构建脚本
└── *.md                  # 文档
```

## 配置

配置文件位于 `config.yaml`：

```yaml
Database:
  Type: postgres
  Postgres:
    Hostname: localhost
    Port: 15432
    Username: pika
    Password: pika
    Database: pika

log:
  Level: debug
  Filename: ./logs/pika.log

Server:
  Addr: "0.0.0.0:18888"
```

## 开发

### 编译

```bash
# 编译服务端
make build-server

# 编译探针客户端
make build-agent

# 编译所有后端
make build-backend

# 清理
make clean
```

### 运行

```bash
# 运行服务端
make run-server

# 运行探针客户端
make run-agent
```

### 测试

```bash
# 运行测试
make test

# API 测试
./test_api.sh
```

## WebSocket 消息协议

### 探针注册

```json
{
  "type": "register",
  "data": {
    "name": "探针名称",
    "hostname": "主机名",
    "ip": "192.168.1.100",
    "os": "linux",
    "arch": "amd64",
    "version": "1.0.0"
  }
}
```

### 心跳

```json
{
  "type": "heartbeat",
  "data": {}
}
```

### 指标数据

```json
{
  "type": "metrics",
  "data": {
    "type": "cpu",
    "data": {
      "usagePercent": 45.5,
      "coreCount": 8
    }
  }
}
```

## 监控指标

系统支持以下监控指标（使用 gopsutil 采集）：

- **CPU**: 使用率、核心数
- **内存**: 总量、已用、空闲、使用率
- **磁盘**: 挂载点、容量、使用情况
- **网络**: 网卡流量统计
- **负载**: 1/5/15分钟负载

## 时间戳说明

所有时间字段均使用 `int64` 类型的毫秒时间戳：

- 数据库存储：毫秒时间戳
- API 查询：支持毫秒时间戳参数
- WebSocket 消息：服务端自动添加时间戳

示例：
```bash
# 获取最近1小时的数据
START=$(( $(date +%s) * 1000 - 3600000 ))
END=$(( $(date +%s) * 1000 ))
curl "http://localhost:18888/api/agents/{id}/metrics?type=cpu&start=$START&end=$END"
```

## 数据保留

- 默认保留 30 天的历史数据
- 每小时自动清理过期数据
- 可在代码中自定义保留策略

## 性能

- 支持数千个探针同时连接
- WebSocket 实时通信，低延迟
- PostgreSQL 时序数据存储
- 自动清理机制，保持数据库大小可控

## 安全

生产环境建议：

1. 修改 WebSocket 的 `CheckOrigin` 函数
2. 使用 HTTPS/WSS
3. 配置防火墙规则
4. 使用强密码（bcrypt 加密）
5. 定期备份数据库
6. 实现完整的 JWT 认证

## 更新日志

### v2.0.0 (2025-10-20)

- ✅ 将 Probe 重命名为 Agent
- ✅ 时间字段改为 int64 时间戳（毫秒）
- ✅ 添加用户表和完整的用户管理功能
- ✅ 探针使用 gopsutil 进行系统信息采集
- ✅ 密码使用 bcrypt 加密
- ✅ 完善的用户 CRUD 接口
- ✅ 支持用户角色和状态管理

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

[添加许可证信息]

---

**开发状态**: ✅ v2.0 开发完成  
**版本**: 2.0.0  
**最后更新**: 2025-10-20
