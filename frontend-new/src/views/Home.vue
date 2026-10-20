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

    <!-- Personalized Recommendations -->
    <section class="recommendations-section mb-5">
      <div class="container">
        <div class="d-flex justify-content-between align-items-center mb-4">
          <div>
            <h2 class="mb-1">{{ recommendationTitle }}</h2>
            <p class="text-muted mb-0">根据观看偏好、热度和更新时间排序</p>
          </div>
          <span v-if="recommendationMode === 'personalized'" class="badge text-bg-primary">
            个性化
          </span>
        </div>

        <div v-if="recommendationLoading" class="text-center py-4">
          <div class="spinner-border text-primary" role="status">
            <span class="visually-hidden">加载推荐...</span>
          </div>
        </div>

        <div v-else-if="recommendationError" class="alert alert-light border" role="alert">
          <i class="bi bi-info-circle"></i> 推荐暂时不可用，你仍可浏览最新视频。
        </div>

        <div v-else-if="recommendations.length === 0" class="text-muted py-3">
          暂无更多推荐，试试搜索或浏览视频列表。
        </div>

        <div v-else class="row g-4">
          <div
            v-for="video in recommendations"
            :key="video.video_id"
            class="col-6 col-md-4 col-lg-3"
          >
            <VideoCard :video="video" :reason="video.recommendation_reason" />
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
            <VideoCard :video="video" />
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useVideoStore } from '@/stores/video'
import { useUserStore } from '@/stores/user'
import videoApi from '@/api/video'
import VideoCard from '@/components/VideoCard.vue'

const videoStore = useVideoStore()
const userStore = useUserStore()

const loading = ref(true)
const error = ref(null)
const recommendations = ref([])
const recommendationLoading = ref(true)
const recommendationError = ref(null)
const recommendationMode = ref('popular')

const isLoggedIn = computed(() => userStore.isLoggedIn)
const videos = computed(() => videoStore.videos)
const recommendationTitle = computed(() => {
  return recommendationMode.value === 'personalized' ? '为你推荐' : '热门推荐'
})

onMounted(async () => {
  try {
    await videoStore.fetchVideos(1, 4)
  } catch (err) {
    error.value = '加载视频失败：' + err.message
  } finally {
    loading.value = false
  }
  await loadRecommendations()
})

async function loadRecommendations() {
  recommendationLoading.value = true
  recommendationError.value = null
  try {
    const excludedIds = videos.value.map(video => video.video_id)
    const response = await videoApi.getRecommendations(4, excludedIds)
    recommendations.value = response.data.videos || []
    recommendationMode.value = response.data.mode || 'popular'
  } catch (err) {
    recommendationError.value = err.message
  } finally {
    recommendationLoading.value = false
  }
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
  margin-bottom: 4rem;
}

.recommendations-section {
  .badge {
    padding: 0.5rem 0.75rem;
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
