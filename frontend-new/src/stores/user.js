import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

// 安全的 localStorage 访问
function getLocalStorageItem(key) {
  try {
    return typeof localStorage !== 'undefined' ? localStorage.getItem(key) : null
  } catch {
    return null
  }
}

// 安全的 localStorage 设置
function setLocalStorageItem(key, value) {
  try {
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem(key, value)
    }
  } catch {
    // ignore
  }
}

export const useUserStore = defineStore('user', () => {
  const token = ref(getLocalStorageItem('token') || '')
  const user = ref(getLocalStorageItem('user') ? JSON.parse(getLocalStorageItem('user')) : null)
  
  const isLoggedIn = computed(() => !!token.value)
  const username = computed(() => user.value?.username || '')
  
  function setAuth(newToken, userData) {
    token.value = newToken
    user.value = userData
    setLocalStorageItem('token', newToken)
    if (userData) {
      setLocalStorageItem('user', JSON.stringify(userData))
    }
  }

  function logout() {
    token.value = ''
    user.value = null
    try {
      if (typeof localStorage !== 'undefined') {
        localStorage.removeItem('token')
        localStorage.removeItem('user')
      }
    } catch {
      // ignore
    }
  }
  
  function checkAuth() {
    const storedUser = localStorage.getItem('user')
    if (token.value && storedUser) {
      user.value = JSON.parse(storedUser)
    }
  }
  
  return {
    token,
    user,
    isLoggedIn,
    username,
    setAuth,
    logout,
    checkAuth
  }
})
