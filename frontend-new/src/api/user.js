import apiClient from './client'

const userApi = {
  // 后端仅支持用户名 + 密码，没有邮箱字段
  register(username, password) {
    return apiClient.post('/users/register', {
      username,
      password
    })
  },

  login(username, password) {
    return apiClient.post('/users/login', {
      username,
      password
    })
  },

  getProfile() {
    return apiClient.get('/users/profile')
  },

  // 后端 PUT /users/profile 目前只接受 username / password
  updateProfile(data) {
    return apiClient.put('/users/profile', data)
  },

  deleteAccount(password) {
    return apiClient.delete('/users/account', {
      data: { password }
    })
  },

  // 吊销服务端会话，使当前令牌立即失效。
  // 登出时本地 token 会先被清除，因此允许显式传入令牌（否则请求头会缺失）。
  logout(token) {
    const config = token ? { headers: { Authorization: `Bearer ${token}` } } : undefined
    return apiClient.post('/users/logout', null, config)
  },

  // 吊销其他设备上的会话，当前设备保持登录
  logoutOtherDevices() {
    return apiClient.post('/users/logout-all')
  }
}

export default userApi
