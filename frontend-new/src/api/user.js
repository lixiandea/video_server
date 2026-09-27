import apiClient from './client'

const userApi = {
  // 邮箱为可选字段：未填写时不提交该键，后端按「未填写」处理
  register(username, password, email) {
    const payload = { username, password }
    if (email) {
      payload.email = email
    }
    return apiClient.post('/users/register', payload)
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

  // 后端 PUT /users/profile 支持 username / password / email / bio
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
