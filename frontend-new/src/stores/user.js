import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import userApi from '@/api/user'
import { setAccessToken } from '@/api/client'

function getStoredUser() {
  try {
    const raw = typeof localStorage !== 'undefined' ? localStorage.getItem('user') : null
    return raw ? JSON.parse(raw) : null
  } catch {
    return null
  }
}

export const useUserStore = defineStore('user', () => {
  const accessToken = ref('')
  const user = ref(getStoredUser())
  const initialized = ref(false)
  let initPromise = null

  const isLoggedIn = computed(() => !!accessToken.value)
  const username = computed(() => user.value?.username || '')

  function setAuth(newToken, userData) {
    accessToken.value = newToken || ''
    setAccessToken(accessToken.value)
    user.value = userData || null
    try {
      localStorage.removeItem('token')
      if (userData) {
        localStorage.setItem('user', JSON.stringify(userData))
      } else {
        localStorage.removeItem('user')
      }
    } catch {
      // ignore storage failures
    }
  }

  async function initializeSession() {
    if (initialized.value) return !!accessToken.value
    if (initPromise) return initPromise
    initPromise = userApi.refresh()
      .then(({ data }) => {
        setAuth(data.token, data.user)
        return true
      })
      .catch(() => {
        setAuth('', null)
        return false
      })
      .finally(() => {
        initialized.value = true
        initPromise = null
      })
    return initPromise
  }

  function logout() {
    setAuth('', null)
    return userApi.logout().catch(() => {})
  }

  function updateUser(userData) {
    user.value = { ...(user.value || {}), ...userData }
    try {
      localStorage.setItem('user', JSON.stringify(user.value))
    } catch {
      // ignore storage failures
    }
  }

  if (typeof window !== 'undefined') {
    window.addEventListener('auth:refreshed', (event) => {
      setAuth(accessToken.value, event.detail)
    })
    window.addEventListener('auth:expired', () => {
      setAuth('', null)
    })
  }

  return {
    accessToken,
    user,
    initialized,
    isLoggedIn,
    username,
    setAuth,
    logout,
    initializeSession,
    updateUser
  }
})
