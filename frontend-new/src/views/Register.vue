<template>
  <div class="register-page">
    <div class="container">
      <div class="row justify-content-center">
        <div class="col-md-6 col-lg-4">
          <div class="card shadow">
            <div class="card-body p-5">
              <h2 class="text-center mb-4">用户注册</h2>
              
              <div v-if="error" class="alert alert-danger" role="alert">
                <i class="bi bi-exclamation-triangle"></i> {{ error }}
              </div>
              
              <div v-if="success" class="alert alert-success" role="alert">
                <i class="bi bi-check-circle"></i> 注册成功，即将跳转登录页面...
              </div>
              
              <form @submit.prevent="handleRegister">
                <div class="mb-3">
                  <label for="username" class="form-label">用户名</label>
                  <input 
                    type="text" 
                    class="form-control" 
                    id="username" 
                    v-model="username"
                    required
                    placeholder="请输入用户名"
                    minlength="3"
                    maxlength="20"
                  >
                </div>
                
                <div class="mb-3">
                  <label for="email" class="form-label">邮箱</label>
                  <input 
                    type="email" 
                    class="form-control" 
                    id="email" 
                    v-model="email"
                    required
                    placeholder="请输入邮箱"
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
                    minlength="6"
                  >
                </div>
                
                <div class="mb-3">
                  <label for="confirmPassword" class="form-label">确认密码</label>
                  <input 
                    type="password" 
                    class="form-control" 
                    id="confirmPassword" 
                    v-model="confirmPassword"
                    required
                    placeholder="请再次输入密码"
                  >
                </div>
                
                <div class="d-grid">
                  <button type="submit" class="btn btn-primary" :disabled="loading || success">
                    <span v-if="!loading">注册</span>
                    <span v-else>
                      <span class="spinner-border spinner-border-sm" role="status"></span>
                      注册中...
                    </span>
                  </button>
                </div>
              </form>
              
              <div class="text-center mt-3">
                <span class="text-muted">已有账号？</span>
                <router-link to="/login">立即登录</router-link>
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
import { useRouter } from 'vue-router'
import userApi from '@/api/user'

const router = useRouter()

const username = ref('')
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const loading = ref(false)
const error = ref(null)
const success = ref(false)

const handleRegister = async () => {
  error.value = null
  success.value = false
  
  if (!username.value || !email.value || !password.value) {
    error.value = '请填写所有必填项'
    return
  }
  
  if (password.value !== confirmPassword.value) {
    error.value = '两次输入的密码不一致'
    return
  }
  
  if (password.value.length < 6) {
    error.value = '密码长度至少为 6 位'
    return
  }
  
  loading.value = true
  
  try {
    await userApi.register(username.value, email.value, password.value)
    success.value = true
    
    setTimeout(() => {
      router.push('/login')
    }, 2000)
  } catch (err) {
    error.value = err.response?.data?.message || '注册失败，请重试'
  } finally {
    loading.value = false
  }
}
</script>

<style lang="scss" scoped>
.register-page {
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
