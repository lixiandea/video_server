<template>
  <div class="video-player-container">
    <video
      ref="videoRef"
      :src="src"
      :poster="poster"
      :controls="showControls"
      :autoplay="autoPlay"
      :preload="preload"
      @play="onPlay"
      @pause="onPause"
      @ended="onEnded"
      @error="onError"
      @timeupdate="onTimeUpdate"
      @loadedmetadata="onLoadedMetadata"
      class="video-element"
    >
      您的浏览器不支持视频播放。
    </video>
    
    <div v-if="showOverlay" class="video-overlay">
      <div class="play-button" @click="togglePlay" v-if="!isPlaying">
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="white" width="64px" height="64px">
          <path d="M8 5v14l11-7z"/>
        </svg>
      </div>
    </div>
    
    <div v-if="showLoading" class="loading-indicator">
      <div class="spinner"></div>
    </div>
  </div>
</template>

<script>
import { ref, onMounted, onUnmounted } from 'vue'

export default {
  name: 'VideoPlayer',
  props: {
    src: {
      type: String,
      required: true
    },
    poster: {
      type: String,
      default: ''
    },
    autoPlay: {
      type: Boolean,
      default: false
    },
    showControls: {
      type: Boolean,
      default: true
    },
    preload: {
      type: String,
      default: 'metadata'
    }
  },
  emits: ['play', 'pause', 'ended', 'error', 'timeupdate', 'loadedmetadata'],
  setup(props, { emit }) {
    const videoRef = ref(null)
    const isPlaying = ref(false)
    const showOverlay = ref(true)
    const showLoading = ref(false)
    
    const togglePlay = () => {
      if (videoRef.value.paused) {
        videoRef.value.play()
      } else {
        videoRef.value.pause()
      }
    }
    
    const onPlay = (event) => {
      isPlaying.value = true
      showOverlay.value = false
      emit('play', event)
    }
    
    const onPause = (event) => {
      isPlaying.value = false
      showOverlay.value = true
      emit('pause', event)
    }
    
    const onEnded = (event) => {
      isPlaying.value = false
      showOverlay.value = true
      emit('ended', event)
    }
    
    const onError = (event) => {
      showLoading.value = false
      emit('error', event)
    }
    
    const onTimeUpdate = (event) => {
      emit('timeupdate', event)
    }
    
    const onLoadedMetadata = (event) => {
      emit('loadedmetadata', event)
    }
    
    // 监听加载事件
    const handleLoadStart = () => {
      showLoading.value = true
    }
    
    const handleCanPlay = () => {
      showLoading.value = false
    }
    
    onMounted(() => {
      if (videoRef.value) {
        videoRef.value.addEventListener('loadstart', handleLoadStart)
        videoRef.value.addEventListener('canplay', handleCanPlay)
      }
    })
    
    onUnmounted(() => {
      if (videoRef.value) {
        videoRef.value.removeEventListener('loadstart', handleLoadStart)
        videoRef.value.removeEventListener('canplay', handleCanPlay)
      }
    })
    
    return {
      videoRef,
      isPlaying,
      showOverlay,
      showLoading,
      togglePlay,
      onPlay,
      onPause,
      onEnded,
      onError,
      onTimeUpdate,
      onLoadedMetadata
    }
  }
}
</script>

<style scoped>
.video-player-container {
  position: relative;
  width: 100%;
  background: #000;
  border-radius: 8px;
  overflow: hidden;
}

.video-element {
  width: 100%;
  display: block;
  max-height: 70vh;
}

.video-overlay {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  display: flex;
  justify-content: center;
  align-items: center;
  background: rgba(0, 0, 0, 0.4);
  z-index: 10;
}

.play-button {
  cursor: pointer;
  transition: transform 0.2s;
}

.play-button:hover {
  transform: scale(1.1);
}

.loading-indicator {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  z-index: 20;
}

.spinner {
  width: 40px;
  height: 40px;
  border: 4px solid rgba(255, 255, 255, 0.3);
  border-radius: 50%;
  border-top-color: white;
  animation: spin 1s ease-in-out infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>