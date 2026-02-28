import { test, expect } from '@playwright/test'

test.describe('首页', () => {
  test('应该成功加载首页', async ({ page }) => {
    await page.goto('/')
    
    await expect(page).toHaveTitle(/首页 - 视频服务器/)
    await expect(page.locator('h1')).toContainText('欢迎使用视频服务器')
  })

  test('应该显示导航栏', async ({ page }) => {
    await page.goto('/')
    
    await expect(page.locator('.navbar-brand')).toContainText('视频服务器')
    await expect(page.locator('a[href="/"]')).toBeVisible()
    await expect(page.locator('a[href="/videos"]')).toBeVisible()
  })

  test('应该显示特性卡片', async ({ page }) => {
    await page.goto('/')
    
    await expect(page.locator('.features-section')).toBeVisible()
    await expect(page.locator('text=快速转码')).toBeVisible()
    await expect(page.locator('text=响应式设计')).toBeVisible()
    await expect(page.locator('text=云端存储')).toBeVisible()
  })

  test('应该可以导航到视频列表页', async ({ page }) => {
    await page.goto('/')
    
    await page.click('a[href="/videos"]')
    
    await expect(page).toHaveURL('/videos')
    await expect(page.locator('h1')).toContainText('视频列表')
  })

  test('应该显示登录/注册按钮（未登录状态）', async ({ page }) => {
    await page.goto('/')
    
    await expect(page.locator('a[href="/login"]')).toBeVisible()
    await expect(page.locator('a[href="/register"]')).toBeVisible()
  })
})
