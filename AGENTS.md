# Video Server Agents 架构文档

## 概述

本项目采用微服务架构，包含多个独立的服务组件（Agents），每个组件都有明确的职责和边界。以下是各组件的详细说明。

## 核心服务组件

### 1. API Server (apiserver)
**文件位置**: `cmd/api-server/main.go`

**职责**:
- 提供 RESTful API 接口
- 处理用户请求和业务逻辑
- 用户认证和授权
- 视频上传、下载、流媒体服务
- 评论管理
- 数据验证和错误处理

**技术栈**:
- Go HTTP 标准库
- Gin/Echo 框架（如果使用）
- JWT 认证
- MySQL 数据库交互

**端口**: 默认 8080

### 2. Scheduler (任务调度器)
**文件位置**: `cmd/scheduler/main.go`

**职责**:
- 定时任务调度
- 视频转码队列管理
- 系统维护任务
- 数据清理和归档
- 性能监控和报告生成

**配置文件**: `config-scheduler.yaml`

**技术特性**:
- Cron 表达式支持
- 任务并发控制
- 失败重试机制
- 任务状态跟踪

### 3. Worker (工作节点)
**文件位置**: `cmd/worker/main.go`

**职责**:
- 执行具体的业务任务
- 视频转码处理
- 文件存储操作
- 数据同步
- 异步任务处理

**特点**:
- 可水平扩展
- 任务队列消费
- 资源隔离
- 健康检查

### 4. Frontend (前端服务)
**文件位置**: `frontend/server.go`

**职责**:
- 提供静态文件服务
- Web 应用界面
- 用户交互处理
- 客户端路由

**技术栈**:
- Go HTTP 服务器
- 静态资源托管
- 简单的模板渲染

**端口**: 默认 3000

### 5. Vue Frontend (Vue.js 前端)
**文件位置**: `frontend-vue/server.js`

**职责**:
- 现代化的 SPA 应用
- 组件化 UI 开发
- 客户端状态管理
- SEO 友好的服务端渲染

**技术栈**:
- Vue 3 + Vite
- Vue Router
- Vuex/Pinia 状态管理
- SSR (服务端渲染)
- Webpack 构建工具

**端口**: 默认 3001

## 基础设施组件

### 1. 数据库 (MySQL)
**配置文件**: `docker/mysql/init.sql`

**职责**:
- 用户数据存储
- 视频元数据管理
- 评论数据存储
- 系统配置存储

**数据库结构**:
- users 表：用户信息
- videos 表：视频信息
- comments 表：评论数据
- tasks 表：任务队列

### 2. 存储系统
**目录**: `storage/videos/`

**职责**:
- 视频文件存储
- 分布式文件管理
- 存储空间优化
- 文件访问控制

### 3. 监控系统
**目录**: `deploy/monitoring/`

**组件**:
- Prometheus：指标收集
- Grafana：数据可视化
- 自定义仪表板

**监控指标**:
- API 请求延迟
- 系统资源使用率
- 数据库性能
- 任务执行状态

## 包结构说明

### `pkg/` 目录 - 共享包

#### auth (认证模块)
**文件**: `pkg/auth/auth.go`

**功能**:
- JWT token 生成和验证
- 用户身份认证
- 权限控制
- 会话管理

#### database (数据库模块)
**文件**: `pkg/database/database.go`

**功能**:
- 数据库连接池管理
- ORM 映射
- 查询构建器
- 事务处理

#### logging (日志模块)
**文件**: `pkg/logging/logger.go`

**功能**:
- 结构化日志记录
- 日志级别控制
- 日志轮转
- 错误追踪

#### metrics (指标模块)
**文件**: `pkg/metrics/metrics.go`

**功能**:
- Prometheus 指标暴露
- 自定义业务指标
- 性能监控
- 健康检查端点

#### storage (存储模块)
**文件**: `pkg/storage/storage.go`

**功能**:
- 文件上传下载
- 存储路径管理
- 文件类型检测
- 存储配额控制

#### tracing (链路追踪)
**文件**: `pkg/tracing/tracer.go`

**功能**:
- 分布式追踪
- 请求链路监控
- 性能瓶颈分析
- 调用关系可视化

#### validation (验证模块)
**文件**: `pkg/validation/validation.go`

**功能**:
- 输入数据验证
- 业务规则校验
- 错误消息格式化
- 验证规则管理

### `internal/` 目录 - 内部实现

#### config (配置管理)
**文件**: `internal/config/config.go`

**功能**:
- 配置文件解析
- 环境变量读取
- 配置热更新
- 多环境支持

#### handlers (HTTP 处理器)
**目录**: `internal/handlers/`

**组件**:
- `user_handler.go`：用户相关接口
- `video_handler.go`：视频相关接口  
- `comment_handler.go`：评论相关接口

#### middleware (中间件)
**目录**: `internal/middleware/`

**组件**:
- `auth.go`：认证中间件
- `logging.go`：日志中间件

#### models (数据模型)
**文件**: `internal/models/models.go`

**功能**:
- 数据库表结构定义
- ORM 映射关系
- 数据验证规则

#### services (业务服务)
**目录**: `internal/services/`

**组件**:
- `user_service.go`：用户业务逻辑
- `video_service.go`：视频业务逻辑
- `comment_service.go`：评论业务逻辑

#### utils (工具函数)
**文件**: `internal/utils/response.go`

**功能**:
- 统一响应格式
- 错误处理封装
- 工具函数集合

### `api/` 目录 - API 定义

#### 用户 API
**文件**: `api/user/api.go`

**接口**:
- 用户注册/登录
- 用户信息获取/更新
- 密码重置

#### 视频 API
**文件**: `api/videos/api.go`

**接口**:
- 视频上传/下载
- 视频列表查询
- 视频详情获取
- 视频删除

#### 评论 API
**文件**: `api/comments/api.go`

**接口**:
- 评论发布
- 评论列表查询
- 评论删除

## 部署架构

### Docker Compose 部署
**文件**: `docker-compose.yml`

**服务组成**:
- apiserver：API 服务
- scheduler：调度服务
- worker：工作节点
- mysql：数据库
- redis：缓存（如果配置）
- frontend：前端服务
- vue-frontend：Vue 前端
- prometheus：监控收集
- grafana：监控展示

### 环境配置
**文件**: `.env`

**配置项**:
- 数据库连接信息
- JWT 密钥
- 服务端口
- 存储路径
- 监控配置

## 启动脚本

### 主要脚本
- `start.sh`：启动所有服务
- `start-docker.sh`：Docker 方式启动
- `build.sh`：编译构建
- `test-api.sh`：API 测试

### 前端脚本
- `run-frontend.sh`：运行前端服务
- `start-all-frontends.sh`：启动所有前端

### 监控脚本
- `start-monitoring.sh`：启动监控系统
- `check-monitoring.sh`：检查监控状态

## 开发指南

### 代码规范
- 遵循 Go 语言最佳实践
- 使用统一的错误处理模式
- 实现结构化日志记录
- 编写单元测试和集成测试

### 扩展建议
1. 添加新的 API 端点时，在对应 handler 中实现
2. 新增业务逻辑时，在 service 层添加
3. 需要共享功能时，考虑提取到 pkg 包中
4. 注意保持各组件间的松耦合

### 监控和调试
- 利用 Prometheus 指标进行性能监控
- 使用 Grafana 仪表板可视化数据
- 通过日志系统进行问题排查
- 启用链路追踪定位性能瓶颈

## 故障排除

### 常见问题
1. **服务启动失败**：检查端口占用和配置文件
2. **数据库连接问题**：验证数据库服务状态和连接参数
3. **API 调用异常**：查看日志文件和监控指标
4. **前端页面无法访问**：确认服务是否正常运行

### 日志位置
- 应用日志：`logs/` 目录
- Docker 日志：通过 `docker logs` 查看
- 系统日志：根据部署环境确定

---

*本文档最后更新时间：February 24, 2026*