<template>
  <div class="videos-page">
    <div class="container">
      <div class="d-flex justify-content-between align-items-center mb-4">
        <h1 class="mb-0">视频列表</h1>
        <div class="search-box">
          <div class="input-group">
            <input 
              type="text" 
              class="form-control" 
              placeholder="搜索视频..."
              v-model="searchQuery"
              @keyup.enter="handleSearch"
            >
            <button class="btn btn-primary" @click="handleSearch">
              <i class="bi bi-search"></i> 搜索
            </button>
          </div>
        </div>
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
      </div>
      
      <div v-else class="row g-4">
        <div 
          v-for="video in videos" 
          :key="video.video_id" 
          class="col-6 col-md-4 col-lg-3"
        >
          <div class="card h-100 video-card" @click="goToVideo(video.video_id)">
            <div class="video-thumb">
              <img
                v-if="video.cover_url"
                :src="video.cover_url + '?size=320'"
                :srcset="`${video.cover_url}?size=320 320w, ${video.cover_url}?size=720 720w`"
                sizes="(max-width: 768px) 50vw, 33vw"
                :alt="video.name"
                loading="lazy"
              >
              <i v-else class="bi bi-film"></i>
              <span v-if="video.duration" class="video-duration">{{ formatDuration(video.duration) }}</span>
            </div>
            <div class="card-body">
              <h6 class="card-title text-truncate">{{ video.name }}</h6>
              <p class="card-text small text-muted mb-1">
                <i class="bi bi-person"></i> {{ video.author_name || '未知' }}
              </p>
              <p class="card-text small text-muted mb-0 d-flex justify-content-between">
                <span><i class="bi bi-eye"></i> {{ video.view_count || 0 }}</span>
                <span><i class="bi bi-clock"></i> {{ video.display_time }}</span>
              </p>
            </div>
          </div>
        </div>
      </div>
      
      <!-- Pagination -->
      <nav v-if="totalPages > 1" class="mt-4">
        <ul class="pagination justify-content-center">
          <li class="page-item" :class="{ disabled: currentPage === 1 }">
            <a class="page-link" href="#" @click.prevent="changePage(currentPage - 1)">
              <i class="bi bi-chevron-left"></i>
            </a>
          </li>
          <li 
            v-for="page in totalPages" 
            :key="page"
            class="page-item" 
            :class="{ active: currentPage === page }"
          >
            <a class="page-link" href="#" @click.prevent="changePage(page)">{{ page }}</a>
          </li>
          <li class="page-item" :class="{ disabled: currentPage === totalPages }">
            <a class="page-link" href="#" @click.prevent="changePage(currentPage + 1)">
              <i class="bi bi-chevron-right"></i>
            </a>
          </li>
        </ul>
      </nav>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useVideoStore } from '@/stores/video'

const router = useRouter()
const videoStore = useVideoStore()

const loading = ref(true)
const error = ref(null)
const searchQuery = ref('')
const currentPage = ref(1)
const pageSize = ref(12)
const total = ref(0)

const videos = computed(() => videoStore.videos)
const totalPages = computed(() => Math.ceil(total.value / pageSize.value))

onMounted(async () => {
  await loadVideos()
})

async function loadVideos() {
  loading.value = true
  error.value = null
  try {
    const query = searchQuery.value.trim()
    const result = query
      ? await videoStore.searchVideos(query, currentPage.value, pageSize.value)
      : await videoStore.fetchVideos(currentPage.value, pageSize.value)
    total.value = result.total || videos.value.length
  } catch (err) {
    error.value = '加载视频失败：' + err.message
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  currentPage.value = 1
  loadVideos()
}

function changePage(page) {
  if (page < 1 || page > totalPages.value) return
  currentPage.value = page
  loadVideos()
}

function goToVideo(id) {
  router.push(`/video/${id}`)
}

// 后端返回的是秒数，列表卡片只需要粗略的 mm:ss / h:mm:ss
function formatDuration(seconds) {
  const total = Math.round(Number(seconds) || 0)
  const hours = Math.floor(total / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  const secs = total % 60
  const pad = (value) => String(value).padStart(2, '0')

  return hours > 0 ? `${hours}:${pad(minutes)}:${pad(secs)}` : `${minutes}:${pad(secs)}`
}

</script>

<style lang="scss" scoped>
.videos-page {
  min-height: calc(100vh - 120px);
  padding: 2rem 0;
  background: #f8f9fa;
}

.search-box {
  max-width: 300px;
}

.video-card {
  cursor: pointer;
  transition: transform 0.3s ease, box-shadow 0.3s ease;
  
  &:hover {
    transform: translateY(-5px);
    box-shadow: 0 10px 25px rgba(0, 0, 0, 0.15);
  }
  
  .video-thumb {
    position: relative;
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

  .video-duration {
    position: absolute;
    right: 0.5rem;
    bottom: 0.5rem;
    padding: 0.1rem 0.35rem;
    border-radius: 0.25rem;
    background: rgba(0, 0, 0, 0.75);
    color: #fff;
    font-size: 0.75rem;
  }
}

@media (max-width: 768px) {
  .search-box {
    max-width: 100%;
    margin-top: 1rem;
  }
  
  .d-flex {
    flex-direction: column;
    align-items: flex-start !important;
    gap: 1rem;
  }
}
</style>
