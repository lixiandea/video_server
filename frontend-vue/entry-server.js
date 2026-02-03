import { createApp } from './src/app.js'

export async function render(url, manifest) {
  const { app, router, store } = createApp(url)

  // 设置服务器端路由
  router.push(url)
  await router.isReady()

  // 获取匹配的组件
  const matchedComponents = router.currentRoute.value.matched.map(
    record => record.components.default
  )

  // 如果路由匹配不到组件且不是静态资源请求，才返回404
  if (!matchedComponents.length) {
    // 检查是否是静态资源请求
    const staticAssetRegex = /\.(css|js|map|json|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$/i
    if (staticAssetRegex.test(url)) {
      // 如果是静态资源请求，则不处理为页面
      return { app, store }
    }
    throw new Error('Page not found')
  }

  // 调用asyncData方法预加载数据
  for (const Component of matchedComponents) {
    if (Component.asyncData) {
      await Component.asyncData({
        store,
        route: router.currentRoute.value
      })
    }
  }

  // 返回应用实例、状态和路径
  return { app, store }
}