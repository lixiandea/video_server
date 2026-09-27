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
  }
}

export default userApi
