import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useUserStore } from '@/stores/user'
import userApi from '@/api/user'

vi.mock('@/api/user', () => ({
  default: {
    refresh: vi.fn(),
    logout: vi.fn(() => Promise.resolve({})),
  },
}))

describe('User Store', () => {
  let userStore

  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    userStore = useUserStore()
    vi.clearAllMocks()
  })

  it('应该初始化时未登录状态', () => {
    expect(userStore.isLoggedIn).toBe(false)
    expect(userStore.accessToken).toBe('')
    expect(userStore.user).toBe(null)
  })

  it('应该只在内存中保存访问令牌', () => {
    const token = 'test-token-123'
    const userData = { id: 1, username: 'testuser', role: 'user' }

    userStore.setAuth(token, userData)

    expect(userStore.accessToken).toBe(token)
    expect(userStore.user).toEqual(userData)
    expect(userStore.isLoggedIn).toBe(true)
    expect(localStorage.getItem('token')).toBe(null)
    expect(JSON.parse(localStorage.getItem('user'))).toEqual(userData)
  })

  it('退出登录时应清理令牌并通过 Cookie 接口注销', async () => {
    userStore.setAuth('test-token-123', { id: 1, username: 'testuser' })

    await userStore.logout()

    expect(userStore.accessToken).toBe('')
    expect(userStore.user).toBe(null)
    expect(userStore.isLoggedIn).toBe(false)
    expect(userApi.logout).toHaveBeenCalledWith()
  })

  it('应该通过刷新 Cookie 恢复会话', async () => {
    const userData = { id: 7, username: 'refreshed', role: 'admin' }
    userApi.refresh.mockResolvedValue({ data: { token: 'fresh-token', user: userData } })

    await expect(userStore.initializeSession()).resolves.toBe(true)

    expect(userStore.accessToken).toBe('fresh-token')
    expect(userStore.user).toEqual(userData)
    expect(userStore.initialized).toBe(true)
    await expect(userStore.initializeSession()).resolves.toBe(true)
    expect(userApi.refresh).toHaveBeenCalledTimes(1)
  })

  it('刷新失败时保持未登录', async () => {
    userApi.refresh.mockRejectedValue(new Error('no cookie'))

    await expect(userStore.initializeSession()).resolves.toBe(false)

    expect(userStore.isLoggedIn).toBe(false)
    expect(userStore.user).toBe(null)
  })

  it('应该返回用户名', () => {
    userStore.setAuth('token', { id: 1, username: 'testuser' })
    expect(userStore.username).toBe('testuser')
  })

  it('在未设置用户时返回空用户名', () => {
    expect(userStore.username).toBe('')
  })
})
