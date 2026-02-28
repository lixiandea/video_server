import { test, expect } from '@playwright/test'

test.describe('用户认证', () => {
  test('应该可以访问登录页面', async ({ page }) => {
    await page.goto('/login')
    
    await expect(page).toHaveTitle(/登录 - 视频服务器/)
    await expect(page.locator('h2')).toContainText('用户登录')
  })

  test('应该可以访问注册页面', async ({ page }) => {
    await page.goto('/register')
    
    await expect(page).toHaveTitle(/注册 - 视频服务器/)
    await expect(page.locator('h2')).toContainText('用户注册')
  })

  test('应该显示登录表单', async ({ page }) => {
    await page.goto('/login')
    
    await expect(page.locator('#username')).toBeVisible()
    await expect(page.locator('#password')).toBeVisible()
    await expect(page.locator('button[type="submit"]')).toContainText('登录')
  })

  test('应该显示注册表单', async ({ page }) => {
    await page.goto('/register')
    
    await expect(page.locator('#username')).toBeVisible()
    await expect(page.locator('#email')).toBeVisible()
    await expect(page.locator('#password')).toBeVisible()
    await expect(page.locator('#confirmPassword')).toBeVisible()
  })

  test('应该可以从登录页跳转到注册页', async ({ page }) => {
    await page.goto('/login')
    
    await page.click('text=立即注册')
    
    await expect(page).toHaveURL('/register')
  })

  test('应该可以从注册页跳转到登录页', async ({ page }) => {
    await page.goto('/register')
    
    await page.click('text=立即登录')
    
    await expect(page).toHaveURL('/login')
  })

  test('应该验证登录表单的必填项', async ({ page }) => {
    await page.goto('/login')
    
    await page.click('button[type="submit"]')
    
    // 浏览器应该会显示必填验证提示
    const username = page.locator('#username')
    await expect(username).toHaveAttribute('required')
  })

  test('应该验证注册表单的必填项', async ({ page }) => {
    await page.goto('/register')
    
    const username = page.locator('#username')
    const email = page.locator('#email')
    const password = page.locator('#password')
    
    await expect(username).toHaveAttribute('required')
    await expect(email).toHaveAttribute('required')
    await expect(password).toHaveAttribute('required')
  })

  test('应该支持移动端视图', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 })
    await page.goto('/login')
    
    await expect(page.locator('h2')).toBeVisible()
    await expect(page.locator('#username')).toBeVisible()
  })
})
