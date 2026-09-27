<template>
  <div class="home-page">
    <!-- Hero Section -->
    <section class="hero-section bg-primary text-white py-5 mb-4">
      <div class="container">
        <div class="row align-items-center">
          <div class="col-lg-6">
            <h1 class="display-4 fw-bold mb-3">欢迎使用视频服务器</h1>
            <p class="lead mb-4">
              高性能视频转码与流媒体服务平台，支持多种视频格式，
              提供流畅的在线播放体验。
            </p>
            <div class="d-flex gap-3">
              <router-link to="/videos" class="btn btn-light btn-lg">
                <i class="bi bi-play-circle"></i> 浏览视频
              </router-link>
              <router-link to="/upload" class="btn btn-outline-light btn-lg" v-if="isLoggedIn">
                <i class="bi bi-upload"></i> 上传视频
              </router-link>
            </div>
          </div>
          <div class="col-lg-6 text-center d-none d-lg-block">
            <i class="bi bi-camera-video display-1"></i>
          </div>
        </div>
      </div>
    </section>

    <!-- Features Section -->
    <section class="features-section mb-4">
      <div class="container">
        <h2 class="text-center mb-4">平台特性</h2>
        <div class="row g-4">
          <div class="col-md-4">
            <div class="card h-100 border-0 shadow-sm">
              <div class="card-body text-center">
                <i class="bi bi-lightning-charge display-4 text-primary mb-3"></i>
                <h5 class="card-title">快速转码</h5>
                <p class="card-text">
                  基于 FFmpeg 的高效转码，支持多种输出格式和分辨率
                </p>
              </div>
            </div>
          </div>
          <div class="col-md-4">
            <div class="card h-100 border-0 shadow-sm">
              <div class="card-body text-center">
                <i class="bi bi-phone display-4 text-primary mb-3"></i>
                <h5 class="card-title">响应式设计</h5>
                <p class="card-text">
                  完美适配手机、平板和电脑，随时随地观看视频
                </p>
              </div>
            </div>
          </div>
          <div class="col-md-4">
            <div class="card h-100 border-0 shadow-sm">
              <div class="card-body text-center">
                <i class="bi bi-hdd-stack display-4 text-primary mb-3"></i>
                <h5 class="card-title">云端存储</h5>
                <p class="card-text">
                  安全的视频存储和管理，支持批量上传和下载
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- Latest Videos Section -->
    <section class="videos-section">
      <div class="container">
        <div class="d-flex justify-content-between align-items-center mb-4">
          <h2 class="mb-0">最新视频</h2>
          <router-link to="/videos" class="btn btn-outline-primary">
            查看全部 <i class="bi bi-arrow-right"></i>
          </router-link>
        </div>
        
        <div v-if="loading" class="text-center py-5">
          <div class="spinner-border text-primary" role="status">
            <span class="visually-hidden">加载中...</span>
          </div>
        </div>
        
        <div v-else-if="error" class="alert alert-danger" role="alert">
          <i class="bi bi-exclamation-triangle"></i> {{ error }}
        </div>
        
        <div v-else-if="videos.length === 0" class="text-center py-5">
          <i class="bi bi-inbox display-4 text-muted"></i>
          <p class="text-muted mt-3">暂无视频</p>
          <router-link to="/upload" class="btn btn-primary" v-if="isLoggedIn">
            <i class="bi bi-upload"></i> 上传第一个视频
          </router-link>
        </div>
        
        <div v-else class="row g-4">
          <div 
            v-for="video in videos" 
            :key="video.video_id" 
            class="col-6 col-md-4 col-lg-3"
          >
            <div class="card h-100 video-card" @click="goToVideo(video.video_id)">
              <div class="video-thumb">
                <i class="bi bi-film"></i>
              </div>
              <div class="card-body">
                <h6 class="card-title text-truncate">{{ video.name }}</h6>
                <p class="card-text small text-muted mb-1">
                  <i class="bi bi-person"></i> {{ video.author_name || '未知' }}
                </p>
                <p class="card-text small text-muted mb-0">
                  <i class="bi bi-clock"></i> {{ video.display_time }}
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useVideoStore } from '@/stores/video'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const videoStore = useVideoStore()
const userStore = useUserStore()

const loading = ref(true)
const error = ref(null)

const isLoggedIn = computed(() => userStore.isLoggedIn)
const videos = computed(() => videoStore.videos)

onMounted(async () => {
  try {
    await videoStore.fetchVideos(1, 8)
  } catch (err) {
    error.value = '加载视频失败：' + err.message
  } finally {
    loading.value = false
  }
})

const goToVideo = (id) => {
  router.push(`/video/${id}`)
}
</script>

<style lang="scss" scoped>
.home-page {
  min-height: calc(100vh - 120px);
}

.hero-section {
  background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
}

.features-section {
  .card {
    transition: transform 0.3s ease, box-shadow 0.3s ease;
    
    &:hover {
      transform: translateY(-5px);
      box-shadow: 0 10px 25px rgba(0, 0, 0, 0.1);
    }
  }
}

.videos-section {
  .video-card {
    cursor: pointer;
    transition: transform 0.3s ease, box-shadow 0.3s ease;
    
    &:hover {
      transform: translateY(-5px);
      box-shadow: 0 10px 25px rgba(0, 0, 0, 0.15);
    }
    
    .video-thumb {
      display: flex;
      align-items: center;
      justify-content: center;
      aspect-ratio: 16/9;
      background: #e9ecef;
      color: #adb5bd;
      font-size: 2rem;
    }
  }
}

@media (max-width: 768px) {
  .hero-section {
    h1 {
      font-size: 1.75rem;
    }
    
    p.lead {
      font-size: 1rem;
    }
  }
}
</style>
