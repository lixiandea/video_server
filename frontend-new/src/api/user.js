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

  refresh() {
    return apiClient.post('/auth/refresh')
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

  logout() {
    return apiClient.post('/auth/logout')
  },

  // 吊销其他设备上的会话，当前设备保持登录
  logoutOtherDevices() {
    return apiClient.post('/users/logout-all')
  }
}

export default userApi
