<template>
  <div id="app" class="app-container">
    <nav class="navbar navbar-expand-lg navbar-dark bg-primary">
      <div class="container-fluid">
        <router-link class="navbar-brand" to="/">
          <i class="bi bi-camera-video"></i> 视频服务器
        </router-link>
        <button 
          class="navbar-toggler" 
          type="button" 
          data-bs-toggle="collapse" 
          data-bs-target="#navbarNav"
          aria-controls="navbarNav" 
          aria-expanded="false" 
          aria-label="Toggle navigation"
        >
          <span class="navbar-toggler-icon"></span>
        </button>
        <div class="collapse navbar-collapse" id="navbarNav">
          <ul class="navbar-nav me-auto">
            <li class="nav-item">
              <router-link class="nav-link" to="/">首页</router-link>
            </li>
            <li class="nav-item">
              <router-link class="nav-link" to="/videos">视频列表</router-link>
            </li>
          </ul>
          <ul class="navbar-nav">
            <li class="nav-item" v-if="!userStore.isLoggedIn">
              <router-link class="nav-link" to="/login">登录</router-link>
            </li>
            <li class="nav-item" v-if="!userStore.isLoggedIn">
              <router-link class="nav-link" to="/register">注册</router-link>
            </li>
            <li class="nav-item dropdown" v-if="userStore.isLoggedIn">
              <a 
                class="nav-link dropdown-toggle" 
                href="#" 
                id="navbarDropdown" 
                role="button"
                data-bs-toggle="dropdown" 
                aria-expanded="false"
              >
                <i class="bi bi-person-circle"></i> {{ userStore.username }}
              </a>
              <ul class="dropdown-menu dropdown-menu-end" aria-labelledby="navbarDropdown">
                <li><router-link class="dropdown-item" to="/profile">个人中心</router-link></li>
                <li><router-link class="dropdown-item" to="/upload">上传视频</router-link></li>
                <li><hr class="dropdown-divider"></li>
                <li><a class="dropdown-item" href="#" @click.prevent="handleLogout">退出登录</a></li>
              </ul>
            </li>
          </ul>
        </div>
      </div>
    </nav>

    <main class="main-content">
      <div class="container-fluid py-4">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </div>
    </main>

    <footer class="footer mt-auto py-3 bg-light">
      <div class="container text-center">
        <span class="text-muted">© 2026 视频服务器 Video Server. All rights reserved.</span>
      </div>
    </footer>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import * as bootstrap from 'bootstrap'

const router = useRouter()
const userStore = useUserStore()

onMounted(() => {
  userStore.checkAuth()
})

const handleLogout = () => {
  userStore.logout()
  router.push('/login')
}
</script>

<style>
.app-container {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.main-content {
  flex: 1;
  background-color: #f8f9fa;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.navbar-brand {
  font-weight: bold;
  font-size: 1.25rem;
}

.navbar-brand i {
  margin-right: 0.5rem;
}

@media (max-width: 768px) {
  .navbar-brand {
    font-size: 1.1rem;
  }
  
  .main-content {
    padding-top: 1rem;
  }
}
</style>
