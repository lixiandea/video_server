# Vue SSR 视频网站

基于 Vue 3 和服务端渲染（SSR）的现代化视频网站。

## 功能特性

- 🎥 **视频播放功能** - 支持多种格式的视频播放
- 📱 **响应式设计** - 适配各种设备屏幕尺寸
- 🚀 **服务端渲染** - 提升首屏加载速度和SEO效果
- 🗂️ **状态管理** - 使用 Vuex 管理应用状态
- 🛣️ **客户端路由** - 使用 Vue Router 实现单页应用体验
- ⚡ **性能优化** - 优化加载速度和用户体验

## 目录结构

```
frontend-vue/
├── src/                    # Vue 源码
│   └── app.js             # 应用入口
├── pages/                  # 页面组件
│   ├── Home.vue           # 首页
│   └── VideoDetail.vue    # 视频详情页
├── components/             # 可复用组件
├── assets/                 # 静态资源
├── static/                 # 静态文件
├── dist/                   # 构建输出
├── server.js               # SSR 服务端
├── webpack.*.js            # Webpack 配置
├── entry-*.js              # 入口文件
├── package.json            # 依赖配置
└── README.md               # 文档
```

## 安装与运行

### 开发环境

1. 安装依赖：
   ```bash
   npm install
   ```

2. 启动开发服务器：
   ```bash
   npm run dev
   ```

3. 访问 `http://localhost:3000`

### 生产环境

1. 构建项目：
   ```bash
   npm run build
   ```

2. 启动生产服务器：
   ```bash
   npm start
   ```

## API 集成

前端通过代理连接到后端 API 服务。默认情况下，API 代理指向 `http://localhost:8080`，这对应于主视频服务的 API 端点。

- 用户管理: `/api/v1/users/*`
- 视频管理: `/api/v1/videos/*`
- 评论管理: `/api/v1/comments/*`

## 组件说明

### Home.vue
主页组件，展示热门视频列表和推荐内容。

### VideoDetail.vue
视频详情页组件，包含视频播放器和相关信息。

### App.vue
根组件，包含导航栏和全局布局。

## 技术栈

- **Vue 3** - 渐进式 JavaScript 框架
- **Vue Router** - 客户端路由
- **Vuex** - 状态管理
- **Express** - 服务端渲染框架
- **Webpack** - 模块打包器
- **Axios** - HTTP 客户端
- **CSS** - 样式处理

## 自定义配置

你可以通过修改以下文件来自定义应用：

- `src/app.js` - 应用配置和状态管理
- `server.js` - 服务端配置和 API 代理
- `webpack.*.js` - 构建配置
- 组件文件 - UI 和交互逻辑

## 部署

将构建后的 `dist` 目录部署到支持 Node.js 的服务器上，或者将静态资源部署到 CDN 上。

## 常见问题

### 如何连接到不同的后端服务？

修改 `server.js` 文件中的 API 代理目标地址。

### 如何自定义样式？

在组件的 `<style>` 标签中添加 CSS，或在 `assets` 目录下添加全局样式。

### 如何添加新页面？

1. 在 `pages` 目录下创建新的 Vue 组件
2. 在 `src/app.js` 中添加对应的路由