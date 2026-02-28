import { test, expect } from '@playwright/test'

test.describe('视频列表页', () => {
  test('应该成功加载视频列表页', async ({ page }) => {
    await page.goto('/videos')
    
    await expect(page).toHaveTitle(/视频列表 - 视频服务器/)
    await expect(page.locator('h1')).toContainText('视频列表')
  })

  test('应该显示搜索框', async ({ page }) => {
    await page.goto('/videos')
    
    await expect(page.locator('input[placeholder="搜索视频..."]')).toBeVisible()
    await expect(page.locator('button:has-text("搜索")')).toBeVisible()
  })

  test('应该可以输入搜索关键词', async ({ page }) => {
    await page.goto('/videos')
    
    const searchInput = page.locator('input[placeholder="搜索视频..."]')
    await searchInput.fill('测试视频')
    
    await expect(searchInput).toHaveValue('测试视频')
  })

  test('应该显示空状态（无视频时）', async ({ page }) => {
    await page.goto('/videos')
    
    // 等待加载完成
    await page.waitForSelector('.spinner-border', { state: 'detached' })
    
    // 如果没有视频，应该显示空状态
    const emptyState = page.locator('text=暂无视频')
    const videoCards = page.locator('.video-card')
    
    // 要么显示空状态，要么显示视频卡片
    const emptyStateVisible = await emptyState.isVisible().catch(() => false)
    const cardsCount = await videoCards.count()
    
    expect(emptyStateVisible || cardsCount > 0).toBeTruthy()
  })

  test('应该支持响应式布局', async ({ page }) => {
    // 移动端视图
    await page.setViewportSize({ width: 375, height: 667 })
    await page.goto('/videos')
    
    await expect(page.locator('h1')).toBeVisible()
    
    // 桌面端视图
    await page.setViewportSize({ width: 1920, height: 1080 })
    await page.goto('/videos')
    
    await expect(page.locator('.search-box')).toBeVisible()
  })
})
