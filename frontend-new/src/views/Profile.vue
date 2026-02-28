<template>
  <div class="profile-page">
    <div class="container">
      <div class="row">
        <div class="col-lg-8">
          <div class="card shadow mb-4">
            <div class="card-body">
              <h2 class="mb-4">个人信息</h2>
              
              <div v-if="error" class="alert alert-danger" role="alert">
                <i class="bi bi-exclamation-triangle"></i> {{ error }}
              </div>
              
              <div v-if="success" class="alert alert-success" role="alert">
                <i class="bi bi-check-circle"></i> 保存成功
              </div>
              
              <form @submit.prevent="handleUpdate">
                <div class="row">
                  <div class="col-md-6">
                    <div class="mb-3">
                      <label for="username" class="form-label">用户名</label>
                      <input 
                        type="text" 
                        class="form-control" 
                        id="username" 
                        v-model="profile.username"
                      >
                    </div>
                  </div>
                  
                  <div class="col-md-6">
                    <div class="mb-3">
                      <label for="email" class="form-label">邮箱</label>
                      <input 
                        type="email" 
                        class="form-control" 
                        id="email" 
                        v-model="profile.email"
                      >
                    </div>
                  </div>
                </div>
                
                <div class="mb-3">
                  <label for="bio" class="form-label">个人简介</label>
                  <textarea 
                    class="form-control" 
                    id="bio" 
                    v-model="profile.bio"
                    rows="3"
                    placeholder="介绍一下自己..."
                  ></textarea>
                </div>
                
                <div class="d-flex gap-2">
                  <button type="submit" class="btn btn-primary" :disabled="loading">
                    <span v-if="!loading">保存修改</span>
                    <span v-else>
                      <span class="spinner-border spinner-border-sm" role="status"></span>
                      保存中...
                    </span>
                  </button>
                  <button type="button" class="btn btn-outline-secondary" @click="loadProfile">
                    <i class="bi bi-arrow-clockwise"></i> 重置
                  </button>
                </div>
              </form>
            </div>
          </div>
          
          <div class="card shadow">
            <div class="card-body">
              <h4 class="mb-3">我的视频</h4>
              
              <div v-if="videosLoading" class="text-center py-3">
                <div class="spinner-border text-primary" role="status">
                  <span class="visually-hidden">加载中...</span>
                </div>
              </div>
              
              <div v-else-if="myVideos.length === 0" class="text-muted text-center py-3">
                <i class="bi bi-inbox"></i> 还没有上传视频
                <router-link to="/upload" class="d-block mt-2">
                  <button class="btn btn-primary btn-sm">上传第一个视频</button>
                </router-link>
              </div>
              
              <div v-else class="row g-3">
                <div v-for="video in myVideos" :key="video.id" class="col-6 col-md-4">
                  <div class="card video-card" @click="goToVideo(video.id)">
                    <img 
                      :src="video.cover_url || '/static/placeholder.jpg'" 
                      class="card-img-top" 
                      :alt="video.title"
                    >
                    <div class="card-body p-2">
                      <h6 class="card-title text-truncate small">{{ video.title }}</h6>
                      <p class="card-text small text-muted mb-0">
                        <i class="bi bi-eye"></i> {{ video.view_count || 0 }}
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
        
        <div class="col-lg-4">
          <div class="card shadow mb-4">
            <div class="card-body text-center">
              <div class="avatar bg-primary text-white rounded-circle d-inline-flex align-items-center justify-content-center mb-3">
                <span class="display-4">{{ profile.username?.charAt(0).toUpperCase() }}</span>
              </div>
              <h4>{{ profile.username }}</h4>
              <p class="text-muted">{{ profile.email }}</p>
            </div>
          </div>
          
          <div class="card shadow">
            <div class="card-body">
              <h5 class="mb-3">统计信息</h5>
              <div class="row text-center">
                <div class="col-4">
                  <h3 class="text-primary">{{ myVideos.length }}</h3>
                  <p class="small text-muted">视频数</p>
                </div>
                <div class="col-4">
                  <h3 class="text-primary">{{ totalViews }}</h3>
                  <p class="small text-muted">总观看</p>
                </div>
                <div class="col-4">
                  <h3 class="text-primary">0</h3>
                  <p class="small text-muted">粉丝</p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import userApi from '@/api/user'
import videoApi from '@/api/video'
import apiClient from '@/api/client'

const router = useRouter()
const userStore = useUserStore()

const profile = reactive({
  username: '',
  email: '',
  bio: ''
})

const loading = ref(false)
const error = ref(null)
const success = ref(false)
const myVideos = ref([])
const videosLoading = ref(false)

const totalViews = computed(() => {
  return myVideos.value.reduce((sum, video) => sum + (video.view_count || 0), 0)
})

onMounted(async () => {
  await loadProfile()
  await loadMyVideos()
})

async function loadProfile() {
  try {
    const response = await userApi.getProfile()
    const data = response.data
    profile.username = data.username || ''
    profile.email = data.email || ''
    profile.bio = data.bio || ''
  } catch (err) {
    error.value = '加载个人信息失败'
  }
}

async function loadMyVideos() {
  videosLoading.value = true
  try {
    const response = await videoApi.getMyVideos()
    myVideos.value = response.data.videos || []
  } catch (err) {
    console.error('Failed to load videos:', err)
  } finally {
    videosLoading.value = false
  }
}

async function handleUpdate() {
  loading.value = true
  error.value = null
  success.value = false
  
  try {
    await userApi.updateProfile({
      username: profile.username,
      email: profile.email,
      bio: profile.bio
    })
    success.value = true
    userStore.user = { ...userStore.user, username: profile.username }
    localStorage.setItem('user', JSON.stringify(userStore.user))
  } catch (err) {
    error.value = err.response?.data?.message || '保存失败'
  } finally {
    loading.value = false
  }
}

function goToVideo(id) {
  router.push(`/video/${id}`)
}
</script>

<style lang="scss" scoped>
.profile-page {
  min-height: calc(100vh - 120px);
  padding: 2rem 0;
  background: #f8f9fa;
}

.avatar {
  width: 100px;
  height: 100px;
}

.video-card {
  cursor: pointer;
  transition: transform 0.2s ease;
  
  &:hover {
    transform: scale(1.05);
  }
  
  .card-img-top {
    aspect-ratio: 16/9;
    object-fit: cover;
  }
}
</style>
