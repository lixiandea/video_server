<template>
  <div class="video-detail-page">
    <div class="container">
      <div class="row">
        <div class="col-lg-8">
          <!-- Video Player -->
          <div class="video-player-section mb-4">
            <VideoPlayer
              v-if="videoUrl"
              ref="playerRef"
              :src="videoUrl"
              :poster="currentVideo?.cover_url"
              :autoPlay="true"
              :qualities="availableQualities"
              @ready="onPlayerReady"
              @error="onPlayerError"
            />
          </div>
          
          <!-- Video Info -->
          <div class="video-info-section mb-4">
            <h1 class="video-title mb-3">{{ currentVideo?.title || '加载中...' }}</h1>
            
            <div class="video-meta d-flex flex-wrap gap-3 text-muted mb-3">
              <span><i class="bi bi-eye"></i> {{ currentVideo?.view_count || 0 }} 次观看</span>
              <span><i class="bi bi-calendar"></i> {{ formatDate(currentVideo?.created_at) }}</span>
              <span v-if="currentVideo?.duration">
                <i class="bi bi-clock"></i> {{ formatDuration(currentVideo.duration) }}
              </span>
            </div>
            
            <div class="video-actions d-flex gap-2 mb-3">
              <button class="btn btn-primary" @click="toggleLike">
                <i :class="isLiked ? 'bi bi-hand-thumbs-up-fill' : 'bi bi-hand-thumbs-up'"></i>
                {{ likeCount }}
              </button>
              <button class="btn btn-outline-primary">
                <i class="bi bi-share"></i> 分享
              </button>
              <button class="btn btn-outline-danger" v-if="isOwner" @click="handleDelete">
                <i class="bi bi-trash"></i> 删除
              </button>
            </div>
            
            <div class="video-description" v-if="currentVideo?.description">
              <h5>简介</h5>
              <p class="text-muted">{{ currentVideo.description }}</p>
            </div>
          </div>
          
          <!-- Comments Section -->
          <div class="comments-section">
            <h4 class="mb-3">评论</h4>
            
            <div v-if="isLoggedIn" class="comment-form mb-4">
              <div class="mb-3">
                <textarea 
                  class="form-control" 
                  rows="3" 
                  placeholder="写下你的评论..."
                  v-model="newComment"
                ></textarea>
              </div>
              <button class="btn btn-primary" @click="submitComment" :disabled="submitting">
                <i class="bi bi-send"></i> 发表评论
              </button>
            </div>
            
            <div v-else class="alert alert-info">
              <i class="bi bi-info-circle"></i> 请 <router-link :to="`/login?redirect=/video/${videoId}`">登录</router-link> 后发表评论
            </div>
            
            <div v-if="commentsLoading" class="text-center py-3">
              <div class="spinner-border text-primary" role="status">
                <span class="visually-hidden">加载中...</span>
              </div>
            </div>
            
            <div v-else-if="comments.length === 0" class="text-center py-3 text-muted">
              <i class="bi bi-chat-square-text"></i> 暂无评论，快来抢沙发吧
            </div>
            
            <div v-else class="comments-list">
              <div v-for="comment in comments" :key="comment.id" class="comment-item mb-3">
                <div class="d-flex gap-3">
                  <div class="avatar bg-primary text-white rounded-circle d-flex align-items-center justify-content-center">
                    {{ comment.username?.charAt(0).toUpperCase() }}
                  </div>
                  <div class="flex-grow-1">
                    <div class="d-flex justify-content-between align-items-center">
                      <h6 class="mb-0">{{ comment.username }}</h6>
                      <small class="text-muted">{{ formatDate(comment.created_at) }}</small>
                    </div>
                    <p class="mb-0 mt-1">{{ comment.content }}</p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
        
        <!-- Sidebar: Related Videos -->
        <div class="col-lg-4">
          <div class="related-videos">
            <h4 class="mb-3">相关推荐</h4>
            
            <div v-if="relatedLoading" class="text-center py-3">
              <div class="spinner-border text-primary" role="status">
                <span class="visually-hidden">加载中...</span>
              </div>
            </div>
            
            <div v-else-if="relatedVideos.length === 0" class="text-muted">
              暂无相关视频
            </div>
            
            <div v-else class="related-list">
              <div 
                v-for="video in relatedVideos" 
                :key="video.id"
                class="related-item mb-3"
                @click="goToVideo(video.id)"
              >
                <div class="row g-2">
                  <div class="col-5">
                    <img 
                      :src="video.cover_url || '/static/placeholder.jpg'" 
                      class="img-fluid rounded" 
                      :alt="video.title"
                    >
                  </div>
                  <div class="col-7">
                    <h6 class="text-truncate mb-1">{{ video.title }}</h6>
                    <p class="small text-muted mb-0">
                      <i class="bi bi-eye"></i> {{ video.view_count || 0 }}
                    </p>
                  </div>
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
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useVideoStore } from '@/stores/video'
import { useUserStore } from '@/stores/user'
import VideoPlayer from '@/components/VideoPlayer.vue'
import apiClient from '@/api/client'

const route = useRoute()
const router = useRouter()
const videoStore = useVideoStore()
const userStore = useUserStore()

const videoId = computed(() => route.params.id)
const currentVideo = computed(() => videoStore.currentVideo)
const isLoggedIn = computed(() => userStore.isLoggedIn)
const isOwner = computed(() => {
  return isLoggedIn.value && currentVideo.value?.user_id === userStore.user?.id
})

const videoUrl = ref('')
const availableQualities = ref([])
const likeCount = ref(0)
const isLiked = ref(false)
const newComment = ref('')
const submitting = ref(false)
const comments = ref([])
const commentsLoading = ref(false)
const relatedVideos = ref([])
const relatedLoading = ref(false)

const playerRef = ref(null)

onMounted(async () => {
  await loadVideoDetail()
  await loadComments()
  await loadRelatedVideos()
})

async function loadVideoDetail() {
  try {
    const data = await videoStore.fetchVideoDetail(videoId.value)
    videoUrl.value = `/api/v1/videos/${videoId.value}/stream`
    availableQualities.value = data.qualities || []
    likeCount.value = data.like_count || 0
  } catch (err) {
    console.error('Failed to load video:', err)
  }
}

async function loadComments() {
  commentsLoading.value = true
  try {
    const response = await apiClient.get(`/videos/${videoId.value}/comments`)
    comments.value = response.data.comments || []
  } catch (err) {
    console.error('Failed to load comments:', err)
  } finally {
    commentsLoading.value = false
  }
}

async function loadRelatedVideos() {
  relatedLoading.value = true
  try {
    const response = await apiClient.get(`/videos/${videoId.value}/related`)
    relatedVideos.value = response.data.videos || []
  } catch (err) {
    console.error('Failed to load related videos:', err)
  } finally {
    relatedLoading.value = false
  }
}

function onPlayerReady(player) {
  console.log('Player ready:', player)
}

function onPlayerError(error) {
  console.error('Player error:', error)
}

function toggleLike() {
  if (!isLoggedIn.value) {
    router.push(`/login?redirect=/video/${videoId.value}`)
    return
  }
  // TODO: Implement like API
  isLiked.value = !isLiked.value
  likeCount.value += isLiked.value ? 1 : -1
}

async function submitComment() {
  if (!newComment.value.trim() || submitting.value) return
  
  submitting.value = true
  try {
    await apiClient.post(`/videos/${videoId.value}/comments`, {
      content: newComment.value
    })
    newComment.value = ''
    await loadComments()
  } catch (err) {
    console.error('Failed to submit comment:', err)
    alert('评论失败，请重试')
  } finally {
    submitting.value = false
  }
}

async function handleDelete() {
  if (!confirm('确定要删除这个视频吗？')) return
  
  try {
    await videoStore.deleteVideo(videoId.value)
    router.push('/videos')
  } catch (err) {
    console.error('Failed to delete video:', err)
    alert('删除失败，请重试')
  }
}

function goToVideo(id) {
  router.push(`/video/${id}`)
}

function formatDate(dateStr) {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN')
}

function formatDuration(seconds) {
  if (!seconds) return ''
  const mins = Math.floor(seconds / 60)
  const secs = Math.floor(seconds % 60)
  return `${mins}:${secs.toString().padStart(2, '0')}`
}
</script>

<style lang="scss" scoped>
.video-detail-page {
  background: #f8f9fa;
  min-height: calc(100vh - 120px);
  padding: 2rem 0;
}

.video-player-section {
  background: #000;
  border-radius: 8px;
  overflow: hidden;
}

.video-info-section {
  background: white;
  padding: 1.5rem;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  
  .video-title {
    font-size: 1.5rem;
    font-weight: 600;
  }
}

.comments-section {
  background: white;
  padding: 1.5rem;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  
  .comment-item {
    .avatar {
      width: 40px;
      height: 40px;
      font-weight: 600;
    }
  }
}

.related-videos {
  background: white;
  padding: 1.5rem;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  
  .related-item {
    cursor: pointer;
    transition: transform 0.2s ease;
    
    &:hover {
      transform: translateX(5px);
    }
    
    img {
      aspect-ratio: 16/9;
      object-fit: cover;
    }
  }
}

@media (max-width: 991px) {
  .video-detail-page {
    padding: 1rem 0;
  }
}
</style>
