# 视频服务微服务架构平台

基于 Go 语言开发的分布式微服务视频平台，提供视频上传、播放、用户管理、评论等功能。

## 📋 目录

- [项目简介](#项目简介)
- [功能特性](#功能特性)
- [技术栈](#技术栈)
- [目录结构](#目录结构)
- [环境要求](#环境要求)
- [快速开始](#快速开始)
- [配置说明](#配置说明)
- [API 接口文档](#api-接口文档)
- [Docker 部署](#docker-部署)
- [前端测试界面](#前端测试界面)
- [Postman 测试](#postman-测试)
- [开发指南](#开发指南)
- [安全机制](#安全机制)
- [性能优化](#性能优化)
- [常见问题](#常见问题)
- [贡献者](#贡献者)
- [许可证](#许可证)

## 项目简介

本项目采用微服务架构设计，包含 API 网关、调度服务、工作服务等多个组件，为视频应用提供完整的后端解决方案。

### 架构概述

- **API 网关层** ([`/cmd/api-server`](/Users/lixiande/project/video_server/cmd/api-server)) - 主要入口点，处理用户请求
- **调度服务** ([`/cmd/scheduler`](/Users/lixiande/project/video_server/cmd/scheduler)) - 后台任务处理器
- **工作服务** ([`/cmd/worker`](/Users/lixiande/project/video_server/cmd/worker)) - 通用后台处理
- **核心服务层** ([`/internal/services`](/Users/lixiande/project/video_server/internal/services)) - 业务逻辑实现
- **数据模型层** ([`/internal/models`](/Users/lixiande/project/video_server/internal/models)) - 使用 GORM 的 ORM 模型
- **存储层** ([`/pkg/storage`](/Users/lixiande/project/video_server/pkg/storage)) - 文件存储抽象
- **认证模块** ([`/pkg/auth`](/Users/lixiande/project/video_server/pkg/auth)) - JWT 基础认证
- **数据库层** ([`/pkg/database`](/Users/lixiande/project/video_server/pkg/database)) - 数据库连接和 ORM
- **验证层** ([`/pkg/validation`](/Users/lixiande/project/video_server/pkg/validation)) - 输入验证工具
- **处理器层** ([`/internal/handlers`](/Users/lixiande/project/video_server/internal/handlers)) - HTTP 请求处理器
- **中间件层** ([`/internal/middleware`](/Users/lixiande/project/video_server/internal/middleware)) - 认证和其他中间件
- **API 定义** ([`/api`](/Users/lixiande/project/video_server/api)) - API 合约定义

## 功能特性

- 🔐 **JWT 认证** - 基于 JWT 的安全身份验证
- 📹 **视频上传与流媒体** - 支持视频上传和流式播放
- 👤 **用户管理系统** - 完整的用户注册、登录、资料管理功能
- 💬 **评论系统** - 视频评论功能
- 🧹 **自动清理** - 自动清理过期视频
- 💾 **可配置存储** - 灵活的存储配置
- ⚡ **限流机制** - 请求频率限制
- ✅ **输入验证** - 全面的输入验证
- 🛡️ **错误处理** - 完善的错误处理机制

## 技术栈

### 后端技术
- **编程语言**: Go 1.21+
- **Web 框架**: Gin
- **数据库**: MySQL 8.0+
- **缓存**: Redis
- **ORM**: GORM
- **认证**: JWT
- **配置管理**: Viper
- **加密**: bcrypt

### 部署技术
- **容器化**: Docker & Docker Compose
- **环境管理**: Shell 脚本自动化

## 目录结构

```
video_server/
├── api/                        # API 定义（按服务划分）
│   ├── comments/               # 评论 API 定义
│   ├── user/                   # 用户 API 定义
│   └── videos/                 # 视频 API 定义
├── bin/                        # 编译后的二进制文件
├── cmd/                        # 应用主包
│   ├── api-server/             # 主 API 服务
│   ├── scheduler/              # 任务调度服务
│   └── worker/                 # 后台工作服务
├── docker/                     # Docker 配置
│   └── mysql/                  # MySQL 初始化脚本
├── frontend/                   # 传统前端测试界面
│   ├── css/                    # 样式表
│   ├── js/                     # JavaScript 文件
│   ├── server.go               # 前端服务器
│   └── index.html              # 主页面
├── frontend-vue/               # Vue SSR 视频网站
│   ├── src/                    # Vue 源码
│   ├── pages/                  # 页面组件
│   ├── components/             # 可复用组件
│   ├── assets/                 # 静态资源
│   ├── static/                 # 静态文件
│   ├── dist/                   # 构建输出
│   ├── server.js               # SSR 服务端
│   ├── webpack.*.js            # Webpack 配置
│   ├── entry-*.js              # 入口文件
│   ├── package.json            # 依赖配置
│   └── README.md               # 文档
├── internal/                   # 内部应用代码
│   ├── config/                 # 配置管理
│   ├── handlers/               # HTTP 请求处理器
│   ├── middleware/             # HTTP 中间件
│   ├── models/                 # 数据模型（GORM）
│   ├── services/               # 业务逻辑服务
│   └── utils/                  # 工具函数
├── pkg/                        # 共享库
│   ├── auth/                   # 认证工具
│   ├── database/               # 数据库工具
│   ├── storage/                # 文件存储抽象
│   └── validation/             # 输入验证
├── storage/                    # 运行时存储目录
│   ├── videos/                 # 视频文件
│   └── temp/                   # 临时文件
├── config.yaml                 # 主配置文件
├── config-scheduler.yaml       # 调度器配置文件
├── docker-compose.yml          # Docker Compose 配置
├── Dockerfile.*                # 各服务 Dockerfile
├── .env                        # 环境变量
├── build.sh                    # 构建脚本
├── start.sh                    # 启动后端服务脚本
├── start-frontend.sh           # 启动前端服务脚本
├── start-docker.sh             # 启动 Docker 环境脚本
├── stop-docker.sh              # 停止 Docker 环境脚本
├── check-docker.sh             # Docker 环境检查脚本
├── cleanup.sh                  # 清理脚本
├── postman_collection.json     # Postman API 集合
├── postman_environment.json    # Postman 环境配置
├── go.mod                      # Go 模块定义
└── README.md                   # 项目文档
```

## 环境要求

### 系统要求
- **操作系统**: Linux/macOS/Windows
- **Go 版本**: 1.21 或更高版本
- **内存**: 最少 2GB RAM
- **磁盘空间**: 至少 1GB 可用空间

### 依赖服务
- **MySQL**: 8.0 或更高版本
- **Redis**: 6.0 或更高版本
- **Docker**: 20.10 或更高版本（如使用容器化部署）

## 快速开始

### 本地开发环境设置

1. **克隆项目**
   ```bash
   git clone https://github.com/lixiandea/video_server.git
   cd video_server
   ```

2. **安装依赖**
   ```bash
   go mod tidy
   ```

3. **配置数据库**
   - 修改 [config.yaml](/Users/lixiande/project/video_server/config.yaml) 中的数据库连接信息
   - 执行 MySQL 初始化脚本

4. **构建服务**
   ```bash
   chmod +x build.sh
   ./build.sh
   ```

5. **启动服务**
   ```bash
   chmod +x start.sh
   ./start.sh
   ```

6. **访问服务**
   - API 服务: http://localhost:8080
   - 前端测试界面: http://localhost:3000

### Docker 方式部署

1. **启动 Docker 环境**
   ```bash
   chmod +x start-docker.sh
   ./start-docker.sh
   ```

2. **验证环境**
   ```bash
   chmod +x check-docker.sh
   ./check-docker.sh
   ```

3. **访问服务**
   - API 服务: http://localhost:8080
   - 前端测试界面: http://localhost:3000

## 配置说明

### 主配置文件 ([config.yaml](/Users/lixiande/project/video_server/config.yaml))

```yaml
server:
  port: "8080"              # 服务端口
  mode: "debug"             # 运行模式 (debug, release, test)
  read_timeout: 30          # 读取超时（秒）
  write_timeout: 30         # 写入超时（秒）
  max_file_size: 52428800   # 最大文件大小（字节，50MB）

database:
  host: "localhost"         # 数据库主机
  port: 3306                # 数据库端口
  user: "root"              # 数据库用户名
  password: "Cz05180921."   # 数据库密码
  name: "video_server"      # 数据库名称
  charset: "utf8mb4"        # 字符集

storage:
  video_dir: "./storage/videos/"    # 视频存储目录
  template_dir: "./templates/"      # 模板目录
  temp_dir: "./storage/temp/"       # 临时文件目录
```

### 环境变量配置 (.env)

项目支持通过环境变量覆盖配置文件中的设置：
- `DB_HOST`: 数据库主机地址
- `DB_PORT`: 数据库端口
- `DB_USER`: 数据库用户名
- `DB_PASSWORD`: 数据库密码
- `DB_NAME`: 数据库名
- `SERVER_PORT`: 服务端口

## API 接口文档

### 用户管理接口

| 方法 | 路径 | 描述 | 认证 |
|------|------|------|------|
| POST | `/api/v1/users/register` | 注册新用户 | ❌ |
| POST | `/api/v1/users/login` | 用户登录 | ❌ |
| GET | `/api/v1/users/profile` | 获取用户资料 | ✅ |
| PUT | `/api/v1/users/profile` | 更新用户资料 | ✅ |
| DELETE | `/api/v1/users/account` | 删除用户账户 | ✅ |

#### 请求示例：用户注册

```json
{
  "login_name": "testuser",
  "password": "securepassword123"
}
```

#### 响应示例：成功注册

```json
{
  "code": 200,
  "data": {
    "user_id": 1,
    "login_name": "testuser",
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  },
  "message": "User registered successfully"
}
```

### 视频管理接口

| 方法 | 路径 | 描述 | 认证 |
|------|------|------|------|
| POST | `/api/v1/videos/upload` | 上传视频 | ✅ |
| GET | `/api/v1/videos/{id}` | 获取视频信息 | ✅ |
| GET | `/api/v1/videos/{id}/stream` | 流式播放视频 | ✅ |
| GET | `/api/v1/users/videos` | 获取用户视频列表 | ✅ |
| DELETE | `/api/v1/videos/{id}` | 删除视频 | ✅ |

#### 请求示例：视频上传

使用 multipart/form-data 格式上传视频文件。

### 评论管理接口

| 方法 | 路径 | 描述 | 认证 |
|------|------|------|------|
| POST | `/api/v1/videos/{id}/comments` | 为视频添加评论 | ✅ |
| GET | `/api/v1/videos/{id}/comments` | 获取视频评论 | ✅ |
| GET | `/api/v1/comments/{id}` | 获取特定评论 | ✅ |
| PUT | `/api/v1/comments/{id}` | 更新评论 | ✅ |
| DELETE | `/api/v1/comments/{id}` | 删除评论 | ✅ |

## Docker 部署

### Docker 环境设置

1. **检查 Docker 环境**
   ```bash
   ./check-docker.sh
   ```

2. **启动完整环境**
   ```bash
   ./start-docker.sh
   ```

3. **停止环境**
   ```bash
   ./stop-docker.sh
   ```

### Docker 服务组成

- **MySQL**: 端口 3306，用于数据持久化
- **Redis**: 端口 6379，用于缓存和会话存储
- **API Server**: 端口 8080，主 API 服务
- **Scheduler**: 后台任务调度器
- **Worker**: 后台工作进程
- **Frontend**: 端口 3000，前端测试界面

### 数据库初始化

MySQL 会在启动时自动执行 [docker/mysql/init.sql](/Users/lixiande/project/video_server/docker/mysql/init.sql) 脚本，创建以下表：

- `users`: 用户表
- `video_info`: 视频信息表
- `comments`: 评论表
- `sessions`: 会话表
- `video_del_rec`: 视频删除记录表

## 前端测试界面

项目包含简单的前端测试界面，用于验证 API 功能。

### 启动前端

1. 确保后端服务正在运行
2. 启动前端服务器：
   ```bash
   chmod +x start-frontend.sh
   ./start-frontend.sh
   ```
3. 在浏览器中打开 `http://localhost:3000`

### 前端功能

- 用户管理（注册、登录、资料管理）
- 视频管理（上传、检索、删除）
- 评论管理（创建、读取、更新、删除）
- 实时 API 响应显示
- 可配置的 API 基础 URL 和认证令牌

## Postman 测试

项目提供了完整的 Postman 集合用于 API 测试。

### 使用方法

1. 导入 `postman_collection.json` 到 Postman
2. 导入 `postman_environment.json` 到 Postman
3. 根据需要更新环境变量
4. 集合包含了所有 API 端点，按服务组织

### 包含的测试套件

- 用户服务端点（注册、登录、资料管理）
- 视频服务端点（上传、流式传输、管理）
- 评论服务端点（创建、读取、更新、删除）

## 开发指南

### 代码规范

- 使用 Go 官方格式化工具 `gofmt`
- 函数命名采用驼峰命名法
- 变量命名清晰明确
- 添加必要的注释和文档

### 架构模式

- **分层架构**: 分离业务逻辑、数据访问和表示层
- **依赖注入**: 通过构造函数注入依赖
- **接口隔离**: 定义清晰的接口契约
- **错误处理**: 统一的错误处理机制

### 测试策略

- 单元测试覆盖核心业务逻辑
- 集成测试验证服务间交互
- API 测试确保接口正确性

## 安全机制

### 认证与授权

- **密码安全**: 使用 bcrypt 加密存储密码
- **会话管理**: JWT 令牌进行状态管理
- **权限控制**: 基于角色的访问控制

### 输入验证

- 所有用户输入都会进行验证
- 防止 SQL 注入攻击（通过 GORM）
- 文件上传验证防止恶意上传

### 数据保护

- 敏感信息加密存储
- 访问日志记录
- 数据备份策略

## 性能优化

### 数据库优化

- 连接池配置
- 查询优化和索引
- 读写分离支持

### 缓存策略

- Redis 缓存热点数据
- 会话存储优化
- 响应缓存

### 并发处理

- Go 协程并发处理
- 限流机制防止单个用户的过度请求
- 异步任务处理

## 常见问题

### Q: 无法连接数据库
A: 检查 [config.yaml](/Users/lixiande/project/video_server/config.yaml) 中的数据库配置，确保 MySQL 服务正在运行

### Q: 上传大文件失败
A: 检查 [config.yaml](/Users/lixiande/project/video_server/config.yaml) 中的 `max_file_size` 设置以及服务器磁盘空间

### Q: Docker 启动失败
A: 确保 Docker 和 Docker Compose 已正确安装，检查端口是否被占用

### Q: JWT 令牌过期
A: 重新登录获取新的令牌，或调整配置中的令牌有效期

## 贡献者

- lixiandea@163.com - 项目主要开发者

## 许可证

本项目采用 MIT 许可证。详情请参阅 [LICENSE](/Users/lixiande/project/video_server/LICENSE) 文件。