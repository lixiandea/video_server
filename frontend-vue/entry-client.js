import { createApp } from './src/app.js'

const { app, router, store } = createApp()

// 恢复客户端状态
if (window.__INITIAL_STATE__) {
  store.replaceState(window.__INITIAL_STATE__)
}

// 等待路由准备就绪后再挂载应用
router.isReady().then(() => {
  app.mount('#app')
})