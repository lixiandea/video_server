<template>
  <div class="login-page">
    <div class="container">
      <div class="row justify-content-center">
        <div class="col-md-6 col-lg-4">
          <div class="card shadow">
            <div class="card-body p-5">
              <h2 class="text-center mb-4">用户登录</h2>
              
              <div v-if="error" class="alert alert-danger" role="alert">
                <i class="bi bi-exclamation-triangle"></i> {{ error }}
              </div>
              
              <form @submit.prevent="handleLogin">
                <div class="mb-3">
                  <label for="username" class="form-label">用户名</label>
                  <input 
                    type="text" 
                    class="form-control" 
                    id="username" 
                    v-model="username"
                    required
                    placeholder="请输入用户名"
                  >
                </div>
                
                <div class="mb-3">
                  <label for="password" class="form-label">密码</label>
                  <input 
                    type="password" 
                    class="form-control" 
                    id="password" 
                    v-model="password"
                    required
                    placeholder="请输入密码"
                  >
                </div>
                
                <div class="d-grid">
                  <button type="submit" class="btn btn-primary" :disabled="loading">
                    <span v-if="!loading">登录</span>
                    <span v-else>
                      <span class="spinner-border spinner-border-sm" role="status"></span>
                      登录中...
                    </span>
                  </button>
                </div>
              </form>
              
              <div class="text-center mt-3">
                <span class="text-muted">还没有账号？</span>
                <router-link to="/register">立即注册</router-link>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useUserStore } from '@/stores/user'
import userApi from '@/api/user'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()

const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref(null)

const handleLogin = async () => {
  if (!username.value || !password.value) {
    error.value = '请输入用户名和密码'
    return
  }
  
  loading.value = true
  error.value = null
  
  try {
    const response = await userApi.login(username.value, password.value)
    const { token, user } = response.data
    
    userStore.setAuth(token, user)
    
    const redirect = route.query.redirect || '/'
    router.push(redirect)
  } catch (err) {
    error.value = err.response?.data?.message || '登录失败，请检查用户名和密码'
  } finally {
    loading.value = false
  }
}
</script>

<style lang="scss" scoped>
.login-page {
  min-height: calc(100vh - 120px);
  display: flex;
  align-items: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.card {
  border: none;
  border-radius: 12px;
}

.btn-primary {
  padding: 0.75rem;
  font-weight: 500;
}
</style>
