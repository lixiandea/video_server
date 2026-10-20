import axios from 'axios'

let accessToken = ''
let refreshPromise = null

export function setAccessToken(token) {
  accessToken = token || ''
}

const apiClient = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
})

apiClient.interceptors.request.use(
  (config) => {
    if (accessToken) {
      config.headers.Authorization = `Bearer ${accessToken}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

apiClient.interceptors.response.use(
  (response) => response,
  async (error) => {
    const original = error.config
    const isAuthEndpoint = original?.url?.includes('/auth/refresh') ||
      original?.url?.includes('/users/login') ||
      original?.url?.includes('/users/register')

    if (error.response?.status === 401 && !original?._retry && !isAuthEndpoint) {
      original._retry = true
      try {
        if (!refreshPromise) {
          refreshPromise = axios.post(
            '/api/v1/auth/refresh',
            {},
            { withCredentials: true, timeout: 15000 }
          ).then(({ data }) => {
            setAccessToken(data.token)
            window.dispatchEvent(new CustomEvent('auth:refreshed', { detail: data.user }))
            return data.token
          }).finally(() => {
            refreshPromise = null
          })
        }
        const token = await refreshPromise
        original.headers = original.headers || {}
        original.headers.Authorization = `Bearer ${token}`
        return apiClient(original)
      } catch {
        setAccessToken('')
        window.dispatchEvent(new Event('auth:expired'))
        if (!window.location.pathname.startsWith('/login')) {
          window.location.href = '/login'
        }
      }
    }
    return Promise.reject(error)
  }
)

export default apiClient
