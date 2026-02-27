# 视频服务项目启动流程指南

本文档提供视频服务项目的完整启动流程，包括开发环境和生产环境的部署指南。

## 📋 目录

- [前置要求](#前置要求)
- [快速开始](#快速开始)
- [方式一：Docker Compose 一键启动](#方式一docker-compose-一键启动推荐)
- [方式二：分步启动](#方式二分步启动)
- [方式三：本地开发模式](#方式三本地开发模式)
- [服务验证](#服务验证)
- [常见问题排查](#常见问题排查)
- [停止服务](#停止服务)

## 前置要求

### 系统要求
- **操作系统**: Linux / macOS / Windows (WSL2)
- **内存**: 最少 4GB RAM (推荐 8GB)
- **磁盘**: 至少 5GB 可用空间
- **Go 版本**: 1.21+ (仅本地开发需要)

### 软件依赖

| 软件 | 最低版本 | 检查命令 |
|------|----------|----------|
| Docker | 20.10+ | `docker --version` |
| Docker Compose | 2.0+ | `docker compose version` |
| Go (可选) | 1.21+ | `go version` |

### 安装 Docker

**macOS/Windows**:
```bash
# 安装 Docker Desktop
# macOS: https://docs.docker.com/desktop/install/mac-install/
# Windows: https://docs.docker.com/desktop/install/windows-install/
```

**Linux (Ubuntu/Debian)**:
```bash
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER
```

## 快速开始

```bash
# 1. 克隆项目
git clone https://github.com/lixiandea/video_server.git
cd video_server

# 2. 一键启动所有服务
./start-docker.sh

# 3. 验证服务
./check-docker.sh

# 4. 访问服务
# API: http://localhost:8080
# 前端：http://localhost:3000
# Grafana: http://localhost:3001
```

## 方式一：Docker Compose 一键启动（推荐）

### 适用场景
- ✅ 快速搭建开发环境
- ✅ 测试环境部署
- ✅ 演示环境搭建

### 启动步骤

#### Step 1: 启动所有服务

```bash
# 给脚本添加执行权限
chmod +x start-docker.sh

# 启动所有服务（包括应用服务和监控组件）
./start-docker.sh
```

**启动过程说明**:
1. 创建 Docker 网络
2. 启动 MySQL 数据库
3. 启动 Redis 缓存
4. 启动 Prometheus 监控
5. 启动 Grafana 可视化
6. 启动 Jaeger 链路追踪
7. 构建并启动 API Server
8. 构建并启动 Scheduler
9. 构建并启动 Worker
10. 启动 Frontend 服务

#### Step 2: 等待服务就绪

```bash
# 等待所有服务启动（约 30 秒）
sleep 30
```

#### Step 3: 验证服务状态

```bash
# 检查所有容器状态
docker compose ps

# 运行健康检查脚本
./check-docker.sh
```

**预期输出**:
```
NAME                      STATUS         PORTS
video_server_api          Up 30 seconds  0.0.0.0:8080->8080/tcp
video_server_frontend     Up 30 seconds  0.0.0.0:3000->3000/tcp
video_server_mysql        Up 30 seconds  0.0.0.0:3306->3306/tcp
video_server_redis        Up 30 seconds  0.0.0.0:6379->6379/tcp
video_server_prometheus   Up 30 seconds  0.0.0.0:9090->9090/tcp
video_server_grafana      Up 30 seconds  0.0.0.0:3001->3000/tcp
video_server_scheduler    Up 30 seconds  0.0.0.0:8089->8089/tcp
video_server_worker       Up 30 seconds
```

### 访问服务

| 服务 | URL | 说明 |
|------|-----|------|
| API Server | http://localhost:8080 | RESTful API |
| Frontend | http://localhost:3000 | Web 界面 |
| Grafana | http://localhost:3001 | 监控仪表板 (admin/admin) |
| Prometheus | http://localhost:9090 | 指标查询 |
| Jaeger | http://localhost:16686 | 链路追踪 |
| MySQL | localhost:3306 | 数据库 (需客户端连接) |
| Redis | localhost:6379 | 缓存 (需客户端连接) |

## 方式二：分步启动

### 适用场景
- ✅ 需要精确控制启动顺序
- ✅ 调试特定服务
- ✅ 资源受限环境

### Step 1: 启动基础设施

```bash
# 启动 MySQL, Redis, 监控组件
./scripts/start-infrastructure.sh
```

**详细步骤**:
```bash
# 1. 创建存储目录
mkdir -p storage/videos storage/temp logs

# 2. 启动基础设施服务
docker-compose up -d mysql redis prometheus grafana jaeger

# 3. 等待服务启动
sleep 15

# 4. 验证基础设施
docker-compose exec mysql mysqladmin ping -h localhost
docker-compose exec redis redis-cli ping
```

### Step 2: 启动应用服务

```bash
# 启动 API Server, Scheduler, Worker, Frontend
./scripts/start-app-services.sh
```

**详细步骤**:
```bash
# 1. 构建应用服务镜像
docker-compose build api-server scheduler worker frontend

# 2. 启动应用服务
docker-compose up -d api-server scheduler worker frontend

# 3. 等待服务启动
sleep 10

# 4. 验证应用服务
curl http://localhost:8080/health
```

### Step 3: 检查服务状态

```bash
# 查看所有服务状态
docker compose ps

# 查看特定服务日志
docker-compose logs -f api-server
```

## 方式三：本地开发模式

### 适用场景
- ✅ Go 开发者
- ✅ 需要调试代码
- ✅ 频繁修改代码

### 前置准备

```bash
# 1. 安装 Go 1.21+
# 2. 安装依赖
go mod tidy

# 3. 启动基础设施（仅 MySQL 和 Redis）
docker-compose up -d mysql redis
```

### 启动 API Server

```bash
# 方式 1: 使用构建脚本
./build.sh
./bin/api-server

# 方式 2: 直接运行
cd cmd/api-server
go run main.go

# 方式 3: 使用 air 热加载
# 安装：go install github.com/cosmtrek/air@latest
air
```

### 启动 Scheduler

```bash
cd cmd/scheduler
go run main.go
```

### 启动 Worker

```bash
cd cmd/worker
go run main.go
```

### 启动 Frontend

```bash
# 方式 1: 使用脚本
./start-frontend.sh

# 方式 2: 手动启动
cd frontend
go run server.go
```

## 服务验证

### 1. 健康检查

```bash
# API Server 健康检查
curl http://localhost:8080/health

# 预期响应
# {"service":"api-server","status":"OK"}
```

### 2. 功能测试

```bash
# 运行完整 API 测试套件
chmod +x test-api.sh
./test-api.sh
```

**测试项目**:
1. ✅ Health Check
2. ✅ User Registration
3. ✅ User Login
4. ✅ Get User Profile
5. ✅ Video Upload
6. ✅ Get User Videos
7. ✅ Get Video Info
8. ✅ Add Comment
9. ✅ Get Comments
10. ✅ Get Single Comment

### 3. 数据库验证

```bash
# 连接 MySQL
docker-compose exec mysql mysql -u video_user -pvideo_password video_server

# 查看表
SHOW TABLES;

# 查看用户数据
SELECT * FROM users LIMIT 5;
```

### 4. 监控验证

```bash
# 访问 Grafana
open http://localhost:3001

# 访问 Prometheus
open http://localhost:9090

# 查询指标
# up - 服务状态
# http_requests_total - HTTP 请求数
# go_goroutines - Go 协程数
```

## 常见问题排查

### 问题 1: 容器启动失败

**症状**:
```
Error: Cannot start service api-server: driver failed programming external connectivity
```

**解决方案**:
```bash
# 1. 检查端口占用
lsof -i :8080
lsof -i :3306
lsof -i :6379

# 2. 停止占用端口的服务或修改 docker-compose.yml 端口映射

# 3. 重启 Docker
sudo systemctl restart docker  # Linux
# 或重启 Docker Desktop (macOS/Windows)
```

### 问题 2: 数据库连接失败

**症状**:
```
panic: Failed to connect to database: dial tcp [::1]:3306: connect: connection refused
```

**解决方案**:
```bash
# 1. 检查 MySQL 是否启动
docker-compose ps mysql

# 2. 查看 MySQL 日志
docker-compose logs mysql

# 3. 等待 MySQL 完全启动（首次启动需要 1-2 分钟）
sleep 60

# 4. 测试连接
docker-compose exec mysql mysqladmin ping -h localhost
```

### 问题 3: API Server 持续重启

**症状**:
```
video_server_api Restarting (2) 30 seconds ago
```

**解决方案**:
```bash
# 1. 查看详细日志
docker-compose logs api-server

# 2. 常见问题:
# - 数据库未启动: 先启动 MySQL
# - 配置文件错误: 检查 config.yaml
# - 端口被占用: 检查 8080 端口

# 3. 重新构建并启动
docker-compose build api-server
docker-compose up -d api-server
```

### 问题 4: 前端无法访问

**症状**:
```
浏览器显示：无法访问 localhost:3000
```

**解决方案**:
```bash
# 1. 检查前端服务状态
docker-compose ps frontend

# 2. 查看前端日志
docker-compose logs frontend

# 3. 验证后端 API 可访问
curl http://localhost:8080/health

# 4. 重启前端服务
docker-compose restart frontend
```

### 问题 5: 构建镜像失败

**症状**:
```
ERROR: failed to solve: process "/bin/sh -c go mod download" did not complete successfully
```

**解决方案**:
```bash
# 1. 配置 Go 代理（中国大陆用户）
# 在 Dockerfile 中添加:
# ENV GOPROXY=https://goproxy.cn,direct

# 2. 清理 Docker 缓存
docker builder prune -a

# 3. 重新构建
docker-compose build --no-cache
```

## 停止服务

### 停止所有服务

```bash
# 方式 1: 使用脚本（推荐）
./stop-docker.sh

# 方式 2: Docker Compose 命令
docker-compose down

# 方式 3: 停止并删除数据卷（彻底清理）
docker-compose down -v
```

### 停止特定服务

```bash
# 停止应用服务
docker-compose stop api-server scheduler worker frontend

# 停止基础设施
docker-compose stop mysql redis prometheus grafana jaeger

# 仅停止某个服务
docker-compose stop api-server
```

### 清理资源

```bash
# 删除所有容器
docker-compose down

# 删除悬空镜像
docker image prune

# 删除所有项目相关镜像
docker rmi video_server-api-server video_server-frontend ...
```

## 高级配置

### 环境变量配置

创建 `.env.local` 文件覆盖默认配置:

```bash
# 自定义端口
SERVER_PORT=9090

# 自定义数据库
DB_HOST=custom-mysql
DB_PORT=3307
DB_USER=admin
DB_PASSWORD=secure_password

# JWT 密钥（生产环境必须修改）
JWT_SECRET=your-super-secret-key-change-in-production
```

### 生产环境部署

```bash
# 1. 修改 docker-compose.yml
# - 设置 restart: always
# - 配置资源限制
# - 使用外部数据库

# 2. 配置 HTTPS
# - 添加 Nginx 反向代理
# - 配置 SSL 证书

# 3. 配置日志
# - 使用外部日志系统
# - 配置日志轮转

# 4. 配置监控告警
# - 配置 Prometheus 告警规则
# - 配置通知渠道
```

## 性能优化建议

### 1. 数据库优化

```yaml
# docker-compose.yml
services:
  mysql:
    command: >
      --default-authentication-plugin=mysql_native_password
      --innodb-buffer-pool-size=1G
      --innodb-log-file-size=256M
      --max-connections=500
```

### 2. 应用服务优化

```yaml
# docker-compose.yml
services:
  api-server:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 1G
        reservations:
          cpus: '0.5'
          memory: 512M
    environment:
      - GOMAXPROCS=2
```

### 3. Redis 优化

```yaml
# docker-compose.yml
services:
  redis:
    command: redis-server --maxmemory 512mb --maxmemory-policy allkeys-lru
```

## 相关文档

- [应用服务详细文档](./docs/applications/README.md)
- [基础设施详细文档](./docs/infrastructure/README.md)
- [监控系统文档](./docs/monitoring.md)
- [优化报告](./OPTIMIZATION_REPORT.md)

## 获取帮助

- 查看日志：`docker-compose logs -f <service_name>`
- 进入容器调试：`docker-compose exec <service_name> sh`
- 查看项目文档：`ls -la docs/`
