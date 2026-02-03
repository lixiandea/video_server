const express = require('express')
const compression = require('compression')
const { createServer: createViteServer } = require('vite')
const vue = require('@vitejs/plugin-vue').default
const fs = require('fs')
const path = require('path')

async function createServer() {
  const app = express()

  // 中间件
  app.use(compression())
  
  // 静态资源服务
  app.use('/static', express.static(path.join(__dirname, 'static')))
  app.use('/dist', express.static(path.join(__dirname, 'dist')))

  // 开发环境下使用 Vite
  let vite
  if (process.env.NODE_ENV !== 'production') {
    vite = await createViteServer({
      server: { middlewareMode: true },
      appType: 'custom'
    })
    app.use(vite.middlewares)
  }

  // API 代理（连接到后端视频服务）
  const axios = require('axios')
  app.use('/api', async (req, res) => {
    try {
      // 这里应该代理到你的后端 API 服务
      const backendResponse = await axios({
        method: req.method,
        url: `http://localhost:8080${req.url}`, // 假设后端运行在8080端口
        data: req.body,
        headers: req.headers,
        responseType: req.path.includes('/stream') ? 'stream' : 'json'
      })

      if (req.path.includes('/stream')) {
        // 对于视频流，直接转发
        res.set(backendResponse.headers)
        backendResponse.data.pipe(res)
      } else {
        res.json(backendResponse.data)
      }
    } catch (error) {
      console.error('API proxy error:', error.message)
      res.status(500).json({ error: 'API proxy error' })
    }
  })

  // 静态资源路由处理
  app.get(/\.(css|js|map|json|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$/, (req, res) => {
    res.status(404).end()
  })

  // SSR 路由
  app.get('*', async (req, res) => {
    try {
      const url = req.originalUrl
      
      // 排除静态资源和图标请求
      if (url.startsWith('/assets/') || url.endsWith('.css') || url.endsWith('.js') || url === '/favicon.ico') {
        res.status(404).end()
        return
      }

      let template, render
      if (process.env.NODE_ENV === 'production') {
        // 生产环境：使用预构建的资源
        template = fs.readFileSync(
          path.resolve(__dirname, 'dist/index.html'),
          'utf-8'
        )
        render = require('./dist/server/server.js').render
      } else {
        // 开发环境：从 Vite 加载最新模板
        template = fs.readFileSync(
          path.resolve(__dirname, 'index.html'),
          'utf-8'
        )
        template = await vite.transformIndexHtml(url, template)
        render = (await vite.ssrLoadModule('./entry-server.js')).render
      }

      const { app, store } = await render(url, {})

      // 在服务端渲染时不调用 mount，而是使用 renderToString
      const { renderToString } = await import('@vue/server-renderer')
      const appContent = await renderToString(app)
      const initialState = store.state

      const html = template
        .replace('{{ APP_CONTENT }}', appContent)
        .replace('{{ INITIAL_STATE }}', JSON.stringify(initialState))

      res.status(200).set({ 'Content-Type': 'text/html' }).end(html)
    } catch (e) {
      if (process.env.NODE_ENV !== 'production') {
        // 忽略 favicon 等非关键资源的错误
        if (!req.originalUrl.includes('favicon.ico')) {
          console.log(e.stack)
        }
      }
      // 对于 favicon.ico 等请求，返回适当的状态
      if (req.originalUrl === '/favicon.ico') {
        res.status(204).end() // No Content
        return
      }
      res.status(500).end('Internal Server Error')
    }
  })

  const port = process.env.PORT || 3001
  app.listen(port, () => {
    console.log(`Server is running at http://localhost:${port}`)
  })
}

createServer().catch(err => {
  console.error(err)
})