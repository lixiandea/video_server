# 基础设施服务部署文档

本目录包含视频服务项目的基础架构组件部署文档，主要包括数据库、缓存、监控等基础设施服务的配置和管理。

## 目录结构

```
infrastructure/
├── database/          # 数据库相关文档
│   ├── mysql.md      # MySQL部署和配置
│   └── redis.md      # Redis部署和配置
├── monitoring/        # 监控系统文档
│   ├── prometheus.md # Prometheus配置
│   ├── grafana.md    # Grafana仪表板
│   └── jaeger.md     # 分布式追踪
├── networking/        # 网络配置
│   └── docker-network.md # Docker网络配置
└── README.md         # 本文件
```

## 基础设施组件概览

### 1. 数据库服务
- **MySQL 8.0**: 主数据库，存储用户、视频、评论等核心数据
- **Redis**: 缓存和会话存储，提升系统性能

### 2. 监控系统
- **Prometheus**: 指标收集和存储
- **Grafana**: 数据可视化和仪表板
- **Jaeger**: 分布式追踪系统

### 3. 网络配置
- **Docker Network**: 容器间通信网络
- **端口映射**: 服务对外暴露的端口配置

## 部署方式

基础设施服务可以通过以下方式部署：

1. **Docker Compose** (推荐): 一键部署所有基础设施服务
2. **独立部署**: 分别部署各个组件
3. **云服务**: 使用云厂商托管服务

## 快速开始

### 使用Docker Compose部署

```bash
# 进入项目根目录
cd /path/to/video_server

# 启动基础设施服务
docker-compose up -d mysql redis prometheus grafana jaeger

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f mysql
```

### 独立部署各组件

请参考各组件的具体文档：
- [MySQL部署指南](./database/mysql.md)
- [Redis部署指南](./database/redis.md)
- [监控系统部署](./monitoring/README.md)

## 服务端口映射

| 服务 | 容器端口 | 主机端口 | 用途 |
|------|----------|----------|------|
| MySQL | 3306 | 3306 | 数据库服务 |
| Redis | 6379 | 6379 | 缓存服务 |
| Prometheus | 9090 | 9090 | 指标收集 |
| Grafana | 3000 | 3001 | 监控仪表板 |
| Jaeger | 16686 | 16686 | 分布式追踪UI |

## 环境变量配置

基础设施服务使用以下环境变量：

```bash
# 数据库配置
MYSQL_ROOT_PASSWORD=your_root_password
MYSQL_DATABASE=video_server
MYSQL_USER=video_user
MYSQL_PASSWORD=video_password

# Redis配置
REDIS_PASSWORD=your_redis_password

# 监控配置
GF_SECURITY_ADMIN_PASSWORD=admin_password
```

## 常见问题

### 1. 端口冲突
如果默认端口被占用，可以修改docker-compose.yml中的端口映射。

### 2. 数据持久化
所有基础设施服务都配置了数据卷，确保数据在容器重启后不会丢失。

### 3. 性能调优
根据实际负载情况，可以调整各服务的资源配置和参数。

## 下一步

- [部署应用服务](../applications/README.md)
- [查看监控配置](./monitoring/README.md)
- [了解安全配置](../security/README.md)