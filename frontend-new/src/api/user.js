import apiClient from './client'

const userApi = {
  register(username, password, email) {
    return apiClient.post('/users/register', {
      username,
      password,
      email
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
  
  updateProfile(data) {
    return apiClient.put('/users/profile', data)
  },
  
  deleteAccount(password) {
    return apiClient.delete('/users/account', {
      data: { password }
    })
  },
  
  changePassword(oldPassword, newPassword) {
    return apiClient.put('/users/password', {
      old_password: oldPassword,
      new_password: newPassword
    })
  }
}

export default userApi
