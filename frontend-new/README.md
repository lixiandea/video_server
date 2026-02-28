# Video Server Frontend 2.0

现代化的视频服务器前端，基于 Vue 3 + Vite 构建，支持视频播放、响应式布局和自动化测试。

## 功能特性

### 核心功能
- 🎬 **视频播放模块** - 支持 HLS/DASH 流媒体，兼容移动端和桌面端
- 📱 **响应式设计** - 完美适配手机、平板和电脑浏览器
- 🔐 **用户认证** - 登录、注册、个人中心
- 📤 **视频上传** - 支持断点续传和进度显示
- 🔍 **视频搜索** - 快速查找感兴趣的视频
- 💬 **评论系统** - 支持视频评论和互动

### 技术栈
- **框架**: Vue 3 (Composition API)
- **构建工具**: Vite 5
- **状态管理**: Pinia
- **路由**: Vue Router 4
- **HTTP 客户端**: Axios
- **视频播放**: Video.js + HLS.js
- **UI 框架**: Bootstrap 5
- **测试框架**: Vitest (单元测试) + Playwright (E2E 测试)

## 快速开始

### 环境要求
- Node.js >= 18.x
- npm >= 9.x

### 安装依赖

```bash
cd frontend-new
npm install
```

### 本地开发

```bash
# 启动开发服务器（带热重载）
npm run dev

# 访问 http://localhost:3000
```

### 生产构建

```bash
# 构建生产版本
npm run build

# 预览生产构建
npm run preview
```

### 运行测试

```bash
# 运行单元测试
npm run test:unit

# 生成测试覆盖率报告
npm run test:unit:coverage

# 运行 E2E 测试（需要先启动开发服务器）
npm run test:e2e

# 运行 E2E 测试（带 UI 界面）
npm run test:e2e:ui

# 运行所有测试
npm test
```

### 代码检查

```bash
npm run lint
```

## 项目结构

```
frontend-new/
├── src/
│   ├── api/              # API 客户端
│   │   ├── client.js     # Axios 实例配置
│   │   ├── user.js       # 用户相关 API
│   │   └── video.js      # 视频相关 API
│   ├── components/       # 可复用组件
│   │   └── VideoPlayer.vue  # 视频播放器组件
│   ├── router/          # 路由配置
│   │   └── index.js
│   ├── stores/          # Pinia 状态管理
│   │   ├── user.js      # 用户状态
│   │   └── video.js     # 视频状态
│   ├── tests/           # 测试文件
│   │   ├── e2e/         # E2E 测试
│   │   ├── unit/        # 单元测试
│   │   └── setup.js     # 测试配置
│   ├── views/           # 页面组件
│   │   ├── Home.vue     # 首页
│   │   ├── Videos.vue   # 视频列表
│   │   ├── VideoDetail.vue  # 视频详情
│   │   ├── VideoPlayer.vue  # 播放器页面
│   │   ├── Login.vue    # 登录页
│   │   ├── Register.vue # 注册页
│   │   ├── Profile.vue  # 个人中心
│   │   └── Upload.vue   # 上传视频
│   ├── App.vue          # 根组件
│   └── main.js          # 入口文件
├── public/              # 静态资源
├── index.html           # HTML 模板
├── package.json         # 依赖配置
├── vite.config.js       # Vite 配置
├── vitest.config.js     # Vitest 配置
├── playwright.config.js # Playwright 配置
└── nginx.conf           # Nginx 配置
```

## Docker 部署

### 构建镜像

```bash
# 使用 Node.js serve
docker build -f Dockerfile.frontend.new -t video-frontend:latest .

# 使用 Nginx（推荐）
docker build -f Dockerfile.frontend.nginx -t video-frontend:nginx .
```

### 运行容器

```bash
# 使用 Node.js serve
docker run -d -p 3000:3000 --name video-frontend video-frontend:latest

# 使用 Nginx
docker run -d -p 80:80 --name video-frontend video-frontend:nginx
```

### 使用 Docker Compose

更新 `docker-compose.yml` 中的 frontend 服务：

```yaml
frontend:
  build:
    context: ./frontend-new
    dockerfile: Dockerfile.frontend.nginx
  container_name: video_server_frontend
  restart: always
  depends_on:
    - api-server
  ports:
    - "80:80"
  networks:
    - video_network
```

然后运行：

```bash
docker-compose up -d frontend
```

## 视频播放器功能

### 支持的功能
- ✅ HLS 流媒体播放
- ✅ 自适应码率
- ✅ 播放速度控制 (0.5x - 2x)
- ✅ 清晰度选择
- ✅ 全屏播放
- ✅ 画中画模式
- ✅ 键盘快捷键
- ✅ 移动端手势支持

### 使用示例

```vue
<template>
  <VideoPlayer
    :src="videoUrl"
    :poster="posterUrl"
    :autoPlay="true"
    :qualities="qualities"
    @ready="onPlayerReady"
    @error="onPlayerError"
  />
</template>

<script setup>
import VideoPlayer from '@/components/VideoPlayer.vue'

const videoUrl = '/api/v1/videos/1/stream'
const posterUrl = '/api/v1/videos/1/cover'
const qualities = [
  { label: '1080P', url: '...' },
  { label: '720P', url: '...' },
  { label: '480P', url: '...' }
]
</script>
```

## 响应式设计

### 支持的断点
- **手机**: < 768px
- **平板**: 768px - 991px
- **桌面**: ≥ 992px

### 移动端优化
- 触摸友好的控件
- 横屏全屏播放
- 节省流量的自适应码率
- 优化的导航菜单

## API 配置

在 `src/api/client.js` 中配置 API 地址：

```javascript
const apiClient = axios.create({
  baseURL: '/api/v1',  // 开发环境使用代理
  timeout: 30000,
})
```

开发环境代理配置在 `vite.config.js`：

```javascript
server: {
  proxy: {
    '/api': {
      target: 'http://localhost:8080',
      changeOrigin: true,
    },
  },
}
```

## 测试说明

### 单元测试
- 组件测试
- Store 测试
- 工具函数测试

### E2E 测试
- 页面加载测试
- 导航测试
- 表单提交测试
- 响应式布局测试
- 跨浏览器测试

## 浏览器支持

- Chrome >= 90
- Firefox >= 88
- Safari >= 14
- Edge >= 90
- iOS Safari >= 14
- Chrome for Android >= 90

## 开发规范

### 代码风格
- 使用 Composition API
- 使用 `<script setup>` 语法
- 组件名使用 PascalCase
- 使用 TypeScript（可选）

### 提交规范
```
feat: 新功能
fix: 修复 bug
docs: 文档更新
style: 代码格式调整
refactor: 重构代码
test: 测试相关
chore: 构建/工具链相关
```

## 常见问题

### Q: 视频无法播放？
A: 确保后端 API 服务正常运行，检查 CORS 配置和视频格式支持。

### Q: 移动端播放不流畅？
A: 检查网络状况，尝试降低视频码率，确保使用 HLS 格式。

### Q: 测试失败？
A: 确保安装了所有依赖，检查后端服务是否运行。

## License

MIT
