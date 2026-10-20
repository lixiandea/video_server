<template>
  <div class="admin-review-page">
    <div class="container">
      <div class="d-flex justify-content-between align-items-center mb-4">
        <div>
          <h1 class="mb-1">视频审核</h1>
          <p class="text-muted mb-0">新上传视频需批准后才会进入公开列表与推荐。</p>
        </div>
        <button class="btn btn-outline-secondary" :disabled="loading" @click="loadQueue">
          <i class="bi bi-arrow-clockwise"></i> 刷新
        </button>
      </div>

      <div v-if="error" class="alert alert-danger" role="alert">{{ error }}</div>
      <div v-if="loading" class="text-center py-5">
        <div class="spinner-border text-primary" role="status"></div>
      </div>
      <div v-else-if="videos.length === 0" class="alert alert-light border">
        当前没有待审核视频。
      </div>
      <div v-else class="row g-4">
        <div v-for="video in videos" :key="video.video_id" class="col-md-6 col-xl-4">
          <article class="card h-100 shadow-sm">
            <div class="card-body">
              <div class="d-flex justify-content-between gap-3">
                <h5 class="card-title text-truncate">{{ video.name }}</h5>
                <span class="badge bg-warning text-dark">待审核</span>
              </div>
              <p class="text-muted small mb-2">
                作者：{{ video.author_name || '未知' }} · {{ video.display_time }}
              </p>
              <p class="card-text">{{ video.description || '未填写简介' }}</p>
              <p class="small text-muted">
                {{ formatDuration(video.duration) }} · {{ formatSize(video.size) }}
              </p>
              <div class="mb-3">
                <label :for="`note-${video.video_id}`" class="form-label small">审核备注</label>
                <textarea
                  :id="`note-${video.video_id}`"
                  v-model="notes[video.video_id]"
                  class="form-control"
                  rows="2"
                  placeholder="拒绝时必须填写原因"
                ></textarea>
              </div>
              <div class="d-flex gap-2">
                <button class="btn btn-success" @click="review(video, 'approve')">
                  批准上线
                </button>
                <button class="btn btn-outline-danger" @click="review(video, 'reject')">
                  拒绝
                </button>
              </div>
            </div>
          </article>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import videoApi from '@/api/video'

const videos = ref([])
const notes = reactive({})
const loading = ref(false)
const error = ref('')

onMounted(loadQueue)

async function loadQueue() {
  loading.value = true
  error.value = ''
  try {
    const response = await videoApi.getReviewQueue()
    videos.value = response.data.videos || []
  } catch (err) {
    error.value = err.response?.data?.error || '加载审核队列失败'
  } finally {
    loading.value = false
  }
}

async function review(video, decision) {
  const note = (notes[video.video_id] || '').trim()
  if (decision === 'reject' && !note) {
    error.value = '拒绝视频时必须填写原因'
    return
  }
  if (decision === 'approve' && !confirm(`批准“${video.name}”上线？`)) return
  if (decision === 'reject' && !confirm(`拒绝“${video.name}”？`)) return

  error.value = ''
  try {
    await videoApi.reviewVideo(video.video_id, decision, note)
    videos.value = videos.value.filter(item => item.video_id !== video.video_id)
  } catch (err) {
    error.value = err.response?.data?.error || '审核操作失败'
  }
}

function formatDuration(seconds) {
  const total = Math.round(Number(seconds) || 0)
  const minutes = Math.floor(total / 60)
  const rest = total % 60
  return `${minutes}:${String(rest).padStart(2, '0')}`
}

function formatSize(bytes) {
  const value = Number(bytes) || 0
  return value > 1024 * 1024 ? `${(value / 1024 / 1024).toFixed(1)} MB` : `${Math.round(value / 1024)} KB`
}
</script>

<style scoped>
.admin-review-page {
  min-height: calc(100vh - 120px);
  padding: 2rem 0;
  background: #f8f9fa;
}
</style>
