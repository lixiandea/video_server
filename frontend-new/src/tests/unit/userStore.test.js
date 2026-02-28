import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useUserStore } from '@/stores/user'

describe('User Store', () => {
  let userStore

  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    userStore = useUserStore()
  })

  it('应该初始化时未登录状态', () => {
    expect(userStore.isLoggedIn).toBe(false)
    expect(userStore.token).toBe('')
    expect(userStore.user).toBe(null)
  })

  it('应该设置认证信息', () => {
    const token = 'test-token-123'
    const userData = { id: 1, username: 'testuser' }

    userStore.setAuth(token, userData)

    expect(userStore.token).toBe(token)
    expect(userStore.user).toEqual(userData)
    expect(userStore.isLoggedIn).toBe(true)
    expect(localStorage.getItem('token')).toBe(token)
  })

  it('应该退出登录', () => {
    const token = 'test-token-123'
    const userData = { id: 1, username: 'testuser' }

    userStore.setAuth(token, userData)
    userStore.logout()

    expect(userStore.token).toBe('')
    expect(userStore.user).toBe(null)
    expect(userStore.isLoggedIn).toBe(false)
    expect(localStorage.getItem('token')).toBe(null)
  })

  it('应该检查认证状态', () => {
    const token = 'test-token-123'
    const userData = { id: 1, username: 'testuser' }

    // 先设置 auth
    userStore.setAuth(token, userData)
    
    // 创建新的 store 实例
    setActivePinia(createPinia())
    const newUserStore = useUserStore()
    
    expect(newUserStore.token).toBe(token)
    expect(newUserStore.user).toEqual(userData)
    expect(newUserStore.isLoggedIn).toBe(true)
  })

  it('应该返回用户名', () => {
    const userData = { id: 1, username: 'testuser' }
    userStore.setAuth('token', userData)

    expect(userStore.username).toBe('testuser')
  })

  it('在未设置用户时返回空用户名', () => {
    expect(userStore.username).toBe('')
  })
})
