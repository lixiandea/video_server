# 快速启动指南

## 本地开发（推荐新手）

```bash
# 1. 进入前端目录
cd frontend-new

# 2. 安装依赖（首次运行）
npm install

# 3. 启动开发服务器
npm run dev
```

访问：http://localhost:3000

## Docker 部署

### 方式一：Docker Compose（完整环境）

```bash
# 启动所有服务（包括 MySQL、Redis、API、前端）
docker-compose up -d

# 查看前端日志
docker-compose logs -f frontend

# 停止所有服务
docker-compose down
```

### 方式二：仅部署前端

```bash
# 构建并启动前端容器
docker-compose up -d frontend

# 或使用部署脚本
./deploy-frontend-docker.sh
```

访问：http://localhost:3000

## 运行测试

```bash
# 单元测试
npm run test:unit

# E2E 测试
npm run test:e2e

# 所有测试
npm test
```

## 生产构建

```bash
# 构建
npm run build

# 预览
npm run preview
```

## 项目文件位置

```
video_server/
├── frontend-new/           # 新前端项目
│   ├── src/
│   ├── Dockerfile.frontend.nginx
│   └── README.md
├── docker-compose.yml      # Docker Compose 配置
├── start-frontend-new.sh   # 启动脚本
└── deploy-frontend-docker.sh # 部署脚本
```

## 常用链接

- 前端服务：http://localhost:3000
- API 服务：http://localhost:8080
- MySQL: localhost:3306
- Redis: localhost:6379

## 故障排查

```bash
# 查看 Docker 容器状态
docker-compose ps

# 查看前端容器日志
docker logs video_server_frontend

# 重启前端服务
docker-compose restart frontend

# 重新构建前端
docker-compose build frontend
```
