<template>
  <div class="video-player-page">
    <div class="container">
      <div class="row justify-content-center">
        <div class="col-lg-10">
          <div class="player-wrapper">
            <VideoPlayer
              ref="playerRef"
              :src="videoUrl"
              :autoPlay="true"
              :showInfo="true"
              @ready="onPlayerReady"
              @error="onPlayerError"
            />
          </div>
          
          <div class="text-center mt-3">
            <router-link to="/videos" class="btn btn-outline-primary">
              <i class="bi bi-arrow-left"></i> 返回列表
            </router-link>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import VideoPlayer from '@/components/VideoPlayer.vue'
import videoApi from '@/api/video'

const route = useRoute()
const playerRef = ref(null)

const videoId = computed(() => route.params.id)
const videoUrl = ref('')

onMounted(async () => {
  try {
    const response = await videoApi.getStreamToken(videoId.value)
    videoUrl.value = videoApi.getVideoStream(videoId.value, response.data.token)
  } catch (err) {
    console.error('Failed to get stream token:', err)
  }
})

const onPlayerReady = (player) => {
  console.log('Player ready:', player)
}

const onPlayerError = (error) => {
  console.error('Player error:', error)
}
</script>

<style lang="scss" scoped>
.video-player-page {
  min-height: calc(100vh - 120px);
  padding: 2rem 0;
  background: #000;
}

.player-wrapper {
  border-radius: 8px;
  overflow: hidden;
}
</style>
