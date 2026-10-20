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
          <VideoCard :video="video" />
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
import { useVideoStore } from '@/stores/video'
import VideoCard from '@/components/VideoCard.vue'

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
