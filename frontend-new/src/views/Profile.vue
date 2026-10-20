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
                <div class="mb-3">
                  <label for="username" class="form-label">用户名</label>
                  <input 
                    type="text" 
                    class="form-control" 
                    id="username" 
                    v-model="profile.username"
                  >
                </div>
                
                <div class="mb-3">
                  <label for="email" class="form-label">邮箱</label>
                  <input 
                    type="email" 
                    class="form-control" 
                    id="email" 
                    v-model="profile.email"
                    placeholder="选填，用于账号找回"
                  >
                </div>
                
                <div class="mb-3">
                  <label for="bio" class="form-label">个人简介</label>
                  <textarea 
                    class="form-control" 
                    id="bio" 
                    rows="3" 
                    maxlength="200"
                    v-model="profile.bio"
                    placeholder="选填，最多 200 字"
                  ></textarea>
                </div>

                <div class="mb-3">
                  <label for="currentPassword" class="form-label">当前密码</label>
                  <input
                    type="password"
                    class="form-control"
                    id="currentPassword"
                    v-model="currentPassword"
                    placeholder="修改用户名、邮箱或密码时必填"
                  >
                </div>

                <div class="mb-3">
                  <label for="newPassword" class="form-label">新密码（选填）</label>
                  <input
                    type="password"
                    class="form-control"
                    id="newPassword"
                    v-model="newPassword"
                    minlength="6"
                    placeholder="留空表示不修改"
                  >
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
                <div v-for="video in myVideos" :key="video.video_id" class="col-6 col-md-4">
                  <div class="card video-card" @click="goToVideo(video.video_id)">
                    <div class="video-thumb position-relative">
                      <img
                        v-if="video.cover_url"
                        :src="video.cover_url + '?size=320'"
                        :srcset="`${video.cover_url}?size=320 320w, ${video.cover_url}?size=720 720w`"
                        sizes="(max-width: 768px) 50vw, 33vw"
                        :alt="video.name"
                        loading="lazy"
                      >
                      <i v-else class="bi bi-film"></i>
                      <span v-if="statusLabel(video.status)" class="badge bg-secondary position-absolute top-0 start-0 m-1">
                        {{ statusLabel(video.status) }}
                      </span>
                    </div>
                    <div class="card-body p-2">
                      <h6 class="card-title text-truncate small">{{ video.name }}</h6>
                      <p class="card-text small text-muted mb-0 d-flex justify-content-between">
                        <span><i class="bi bi-eye"></i> {{ video.view_count || 0 }}</span>
                        <span><i class="bi bi-clock"></i> {{ video.display_time }}</span>
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class="card shadow mt-4 border-danger">
            <div class="card-body">
              <h5 class="text-danger">注销账号</h5>
              <p class="small text-muted">账号注销将下架视频、评论并清理会话与推荐行为，操作不可恢复。</p>
              <input
                type="password"
                class="form-control mb-2"
                v-model="deletePassword"
                placeholder="输入当前密码确认"
              >
              <button class="btn btn-outline-danger w-100" :disabled="deleting" @click="handleDeleteAccount">
                永久注销账号
              </button>
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
              <p class="text-muted mb-1">{{ profile.email || '未填写邮箱' }}</p>
              <p class="text-muted small">{{ profile.bio || '这个人很懒，什么都没写' }}</p>
            </div>
          </div>
          
          <div class="card shadow">
            <div class="card-body">
              <h5 class="mb-3">统计信息</h5>
              <div class="row text-center">
                <div class="col-12">
                  <h3 class="text-primary">{{ myVideos.length }}</h3>
                  <p class="small text-muted">视频数</p>
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
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import userApi from '@/api/user'
import videoApi from '@/api/video'

const router = useRouter()
const userStore = useUserStore()

const profile = reactive({
  username: '',
  email: '',
  bio: ''
})

const loading = ref(false)
const deleting = ref(false)
const error = ref(null)
const success = ref(false)
const currentPassword = ref('')
const newPassword = ref('')
const deletePassword = ref('')
const myVideos = ref([])
const videosLoading = ref(false)

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
  } catch {
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
      bio: profile.bio,
      current_password: currentPassword.value,
      password: newPassword.value || undefined
    })
    success.value = true
    currentPassword.value = ''
    newPassword.value = ''
    userStore.updateUser({ username: profile.username })
  } catch (err) {
    error.value = err.response?.data?.message || err.response?.data?.error || '保存失败'
  } finally {
    loading.value = false
  }
}

async function handleDeleteAccount() {
  if (!deletePassword.value) {
    error.value = '请输入当前密码'
    return
  }
  if (!confirm('确定永久注销账号？此操作不可恢复。')) return
  deleting.value = true
  try {
    await userApi.deleteAccount(deletePassword.value)
    await userStore.logout()
    router.push('/login')
  } catch (err) {
    error.value = err.response?.data?.error || '注销账号失败'
  } finally {
    deleting.value = false
  }
}

function goToVideo(id) {
  router.push(`/video/${id}`)
}

function statusLabel(status) {
  if (status === 'processing') return '转码中'
  if (status === 'pending_review') return '待审核'
  if (status === 'rejected') return '审核未通过'
  if (status === 'failed') return '转码失败'
  return ''
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
  
  .video-thumb {
    display: flex;
    align-items: center;
    justify-content: center;
    aspect-ratio: 16/9;
    background: #e9ecef;
    color: #adb5bd;
    font-size: 2rem;
    overflow: hidden;

    img {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }
  }
}
</style>
