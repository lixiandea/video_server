import { test, expect } from '@playwright/test'
import fs from 'node:fs'

const API = process.env.API_URL || 'http://localhost:8080'
// 相对 frontend-new 运行目录；CI 与本地都使用仓库内置固件
const FIXTURE = process.env.TEST_VIDEO_SRC || '../testdata/sample.mp4'

test.setTimeout(90_000)

async function createActiveVideo(request) {
  const username = `e2e_player_${Date.now()}`
  const reg = await request.post(`${API}/api/v1/users/register`, {
    data: { username, password: 'TestPass123' },
  })
  expect(reg.status()).toBe(201)
  const auth = await reg.json()

  const upload = await request.post(`${API}/api/v1/videos/upload`, {
    headers: { Authorization: `Bearer ${auth.token}` },
    multipart: {
      file: { name: 'sample.mp4', mimeType: 'video/mp4', buffer: fs.readFileSync(FIXTURE) },
      name: 'player-smoke',
    },
  })
  expect(upload.status()).toBe(201)
  const { video_id: videoId } = await upload.json()

  let status = ''
  for (let i = 0; i < 40; i++) {
    const info = await request.get(`${API}/api/v1/videos/${videoId}`, {
      headers: { Authorization: `Bearer ${auth.token}` },
    })
    if (info.ok()) {
      status = (await info.json()).status
      if (status === 'active') break
    }
    await new Promise((resolve) => setTimeout(resolve, 1000))
  }
  expect(status).toBe('active')

  return { token: auth.token, videoId, username, password: 'TestPass123' }
}

test('HLS 播放并在失败时回退到 MP4', async ({ page, request, baseURL }) => {
  const { token, videoId, username, password } = await createActiveVideo(request)
  try {
    await page.goto(`${baseURL}/login`)
    await page.locator('#username').fill(username)
    await page.locator('#password').fill(password)
    await page.click('button[type="submit"]')
    await expect(page).toHaveURL('/')

    const hlsRequests = []
    page.on('request', (req) => {
      if (req.url().includes('/hls/')) hlsRequests.push(req.url())
    })

    await page.goto(`${baseURL}/video/${videoId}`)
    await page.waitForTimeout(5000)

    // 1) 播放器确实请求了 HLS playlist/分片，且没有报错层
    expect(hlsRequests.some((url) => url.includes('index.m3u8'))).toBe(true)
    expect(hlsRequests.some((url) => url.includes('.ts'))).toBe(true)
    await expect(page.locator('video.native-video')).toHaveCount(1)
    await expect(page.locator('.video-error')).toHaveCount(0)

    // 2) 拦截 HLS，强制失败 → 应自动回退到 MP4 并显示提示
    await page.route('**/hls/**', (route) => route.abort())
    await page.goto(`${baseURL}/video/${videoId}`)
    await expect(page.locator('.video-fallback-hint')).toHaveCount(1, { timeout: 15000 })

    const fallbackSrc = await page.locator('video.native-video').evaluate((v) => v.src)
    expect(fallbackSrc).toContain('/stream?')
  } finally {
    await request.delete(`${API}/api/v1/users/account`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { password },
    })
  }
})
