// @ts-check
import { defineConfig, devices } from '@playwright/test'

// 前端地址优先取 E2E_BASE_URL（可直接指向已在运行的实例，例如容器里的 :8000）；
// 否则由 Playwright 拉起 Vite 开发服务器，端口可用 E2E_FRONTEND_PORT / VITE_PORT 覆盖。
const frontendPort = process.env.E2E_FRONTEND_PORT || process.env.VITE_PORT || '3000'
const baseURL = process.env.E2E_BASE_URL || `http://localhost:${frontendPort}`

export default defineConfig({
  testDir: './src/tests/e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  // 所有浏览器请求都来自本机同一 IP；并行 worker 会触发应用自身的限流，导致误报。
  workers: 1,
  reporter: process.env.CI
    ? [['github'], ['list'], ['html', { open: 'never' }]]
    : [['html', { open: 'never' }], ['list']],
  
  use: {
    baseURL,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },
    {
      name: 'Mobile Chrome',
      use: { ...devices['Pixel 5'] },
    },
    {
      name: 'Mobile Safari',
      use: { ...devices['iPhone 12'] },
    },
  ],

  // 直接复用外部实例时不需要自建 webServer
  webServer: process.env.E2E_BASE_URL
    ? undefined
    : {
        command: `npm run dev -- --port ${frontendPort} --strictPort`,
        url: baseURL,
        reuseExistingServer: !process.env.CI,
        timeout: 120000,
      },
})
