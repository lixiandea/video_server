# 前端服务部署指南

本文档介绍如何部署优化后的视频服务器前端服务。

## 目录

1. [项目概述](#项目概述)
2. [本地开发部署](#本地开发部署)
3. [Docker 部署](#docker-部署)
4. [生产环境部署](#生产环境部署)
5. [移动端和桌面端兼容性](#移动端和桌面端兼容性)
6. [故障排查](#故障排查)

## 项目概述

新的前端服务基于 Vue 3 + Vite 构建，包含以下特性：

- 🎬 视频播放模块（支持 HLS/DASH）
- 📱 响应式设计（兼容手机和电脑浏览器）
- ✅ 自动化测试（Vitest + Playwright）
- 🐳 Docker 容器化部署

### 技术栈

| 类别 | 技术 |
|------|------|
| 框架 | Vue 3 (Composition API) |
| 构建 | Vite 5 |
| 状态管理 | Pinia |
| 路由 | Vue Router 4 |
| 视频播放 | Video.js + HLS.js |
| UI 框架 | Bootstrap 5 |
| 单元测试 | Vitest |
| E2E 测试 | Playwright |
| 容器化 | Docker + Nginx |

## 本地开发部署

### 前置要求

- Node.js >= 18.x
- npm >= 9.x
- 后端 API 服务运行在 http://localhost:8080

### 安装步骤

```bash
# 1. 进入前端目录
cd frontend-new

# 2. 安装依赖
npm install

# 3. 启动开发服务器
npm run dev
```

访问 http://localhost:3000

### 开发服务器特性

- 热模块替换（HMR）
- 即时编译
- API 代理到后端服务
- 源映射支持

### 运行测试

```bash
# 单元测试
npm run test:unit

# E2E 测试（需要先启动开发服务器）
npm run test:e2e

# 生成覆盖率报告
npm run test:unit:coverage
```

### 生产构建

```bash
# 构建生产版本
npm run build

# 预览生产构建
npm run preview
```

## Docker 部署

### 方式一：使用 Docker Compose（推荐）

```bash
# 1. 确保后端服务正在运行
docker-compose up -d mysql redis api-server

# 2. 构建并启动前端服务
docker-compose up -d frontend

# 3. 查看日志
docker-compose logs -f frontend

# 4. 停止服务
docker-compose stop frontend
```

访问 http://localhost:3000

### 方式二：单独构建前端镜像

```bash
# 使用 Nginx（推荐）
docker build -f Dockerfile.frontend.nginx -t video-frontend:nginx .

# 使用 Node.js serve
docker build -f Dockerfile.frontend.new -t video-frontend:latest .

# 运行容器
docker run -d -p 3000:80 --name video-frontend video-frontend:nginx
```

### 方式三：使用部署脚本

```bash
# 执行部署脚本
./deploy-frontend-docker.sh
```

### Docker 镜像结构

```
┌─────────────────────────────────────┐
│         Nginx Alpine Image          │
├─────────────────────────────────────┤
│  /etc/nginx/conf.d/default.conf     │
│  /usr/share/nginx/html/             │
│    ├── index.html                   │
│    └── static/                      │
│      ├── *.js                       │
│      ├── *.css                      │
│      └── *.woff2                    │
└─────────────────────────────────────┘
```

## 生产环境部署

### Nginx 配置说明

生产环境使用 Nginx 提供静态资源服务，配置包括：

- Gzip 压缩
- 静态资源缓存
- API 代理
- 视频流优化
- 安全头部

### 环境变量配置

创建 `.env.production` 文件：

```bash
# API 地址
VITE_API_TARGET=http://your-api-server.com

# 应用配置
VITE_APP_TITLE=视频服务器
VITE_APP_DESCRIPTION=高性能视频转码与流媒体服务平台
```

### 性能优化

1. **代码分割**
   - 视频播放器单独打包
   - 第三方库单独打包
   - 路由懒加载

2. **资源缓存**
   - 静态资源缓存 1 年
   - HTML 不缓存
   - API 响应缓存

3. **压缩**
   - Gzip 文本资源
   - 图片优化

## 移动端和桌面端兼容性

### 支持的浏览器

| 浏览器 | 桌面版 | 移动版 |
|--------|--------|--------|
| Chrome | ≥ 90 | ≥ 90 |
| Firefox | ≥ 88 | ≥ 88 |
| Safari | ≥ 14 | ≥ 14 |
| Edge | ≥ 90 | ≥ 90 |

### 响应式断点

```scss
// 手机：< 768px
// 平板：768px - 991px
// 桌面：≥ 992px
```

### 移动端优化

1. **触摸友好**
   - 大按钮设计（最小 44x44px）
   - 触摸反馈
   - 手势支持

2. **视频播放**
   - 横屏全屏
   - 自适应码率
   - 节省流量模式

3. **导航**
   - 汉堡菜单
   - 底部导航栏
   - 滑动返回

### 测试方法

```bash
# 使用 Playwright 测试多设备兼容性
npm run test:e2e

# 测试报告位于 playwright-report/
```

### 设备测试矩阵

| 设备 | 分辨率 | 状态 |
|------|--------|------|
| iPhone 12 | 390x844 | ✅ 通过 |
| Pixel 5 | 393x851 | ✅ 通过 |
| iPad Pro | 1024x1366 | ✅ 通过 |
| Desktop Chrome | 1920x1080 | ✅ 通过 |
| Desktop Safari | 1920x1080 | ✅ 通过 |

## 故障排查

### 常见问题

#### 1. 开发服务器无法启动

```bash
# 检查端口占用
lsof -i :3000

# 清理并重启
rm -rf node_modules
npm install
npm run dev
```

#### 2. 视频无法播放

- 检查后端 API 是否运行
- 检查 CORS 配置
- 确认视频格式支持（MP4/HLS）

#### 3. Docker 容器无法启动

```bash
# 查看容器日志
docker logs video_server_frontend

# 检查网络连接
docker network inspect video_server_video_network

# 重建镜像
docker-compose build frontend
```

#### 4. 测试失败

```bash
# 清理缓存
rm -rf node_modules/.vite
rm -rf dist

# 重新安装依赖
npm install

# 重新运行测试
npm run test:unit -- --run
```

### 日志位置

```bash
# Docker 日志
docker-compose logs frontend

# Nginx 日志（容器内）
docker exec video_server_frontend cat /var/log/nginx/access.log
docker exec video_server_frontend cat /var/log/nginx/error.log
```

### 性能监控

```bash
# 查看容器资源使用
docker stats video_server_frontend

# Lighthouse 性能测试
# 在 Chrome DevTools 中运行
```

## 快速参考

### 常用命令

```bash
# 开发
npm run dev              # 启动开发服务器
npm run build            # 生产构建
npm run preview          # 预览构建结果

# 测试
npm run test:unit        # 单元测试
npm run test:e2e         # E2E 测试
npm test                 # 运行所有测试

# Docker
docker-compose up -d frontend    # 启动前端容器
docker-compose logs -f frontend  # 查看日志
docker-compose stop frontend     # 停止容器
```

### 目录结构

```
frontend-new/
├── src/
│   ├── api/           # API 客户端
│   ├── components/    # 组件
│   ├── router/        # 路由
│   ├── stores/        # 状态管理
│   ├── tests/         # 测试
│   ├── views/         # 页面
│   └── main.js        # 入口
├── dist/              # 构建输出
├── Dockerfile.*       # Docker 配置
├── nginx.conf         # Nginx 配置
├── package.json       # 依赖配置
├── vite.config.js     # Vite 配置
└── playwright.config.js # Playwright 配置
```

## 支持

如有问题，请查看：

- [README.md](./README.md) - 项目文档
- [https://vuejs.org](https://vuejs.org) - Vue 3 文档
- [https://vitejs.dev](https://vitejs.dev) - Vite 文档
- [https://videojs.com](https://videojs.com) - Video.js 文档
