<template>
  <div class="video-player-container" ref="containerRef">
    <div class="video-player-wrapper" :class="{ 'fullscreen': isFullscreen }">
      <video
        ref="videoRef"
        class="video-js vjs-default-skin vjs-big-play-centered"
        :poster="poster"
        :preload="preload"
        :autoplay="autoPlay"
        :controls="showControls"
        :muted="muted"
        :loop="loop"
        playsinline
        webkit-playsinline
      >
        <source :src="src" :type="mimeType" />
        <p class="vjs-no-js">
          您的浏览器不支持 JavaScript，请启用 JavaScript 以观看视频。
        </p>
      </video>
      
      <!-- 加载状态 -->
      <div v-if="loading && !isReady" class="video-loading">
        <div class="spinner-border text-light" role="status">
          <span class="visually-hidden">加载中...</span>
        </div>
      </div>
      
      <!-- 错误状态 -->
      <div v-if="error" class="video-error">
        <div class="error-content">
          <i class="bi bi-exclamation-triangle"></i>
          <p>{{ errorMessage }}</p>
          <button class="btn btn-outline-light btn-sm" @click="retry">
            <i class="bi bi-arrow-clockwise"></i> 重试
          </button>
        </div>
      </div>
      
      <!-- 播放速度选择 -->
      <div v-if="showControls" class="playback-speed-menu" ref="speedMenuRef">
        <button 
          class="btn btn-sm btn-outline-light playback-speed-btn"
          @click="toggleSpeedMenu"
          title="播放速度"
        >
          <i class="bi bi-speedometer"></i> {{ playbackRate }}x
        </button>
        <div v-show="speedMenuOpen" class="speed-options">
          <button 
            v-for="rate in playbackRates" 
            :key="rate"
            class="btn btn-sm"
            :class="{ 'btn-primary': playbackRate === rate, 'btn-outline-light': playbackRate !== rate }"
            @click="setPlaybackRate(rate)"
          >
            {{ rate }}x
          </button>
        </div>
      </div>
      
      <!-- 清晰度选择 -->
      <div v-if="qualities.length > 1 && showControls" class="quality-menu" ref="qualityMenuRef">
        <button 
          class="btn btn-sm btn-outline-light quality-btn"
          @click="toggleQualityMenu"
          title="清晰度"
        >
          <i class="bi bi-hd"></i> {{ currentQuality }}
        </button>
        <div v-show="qualityMenuOpen" class="quality-options">
          <button 
            v-for="quality in qualities" 
            :key="quality.label"
            class="btn btn-sm"
            :class="{ 'btn-primary': currentQuality === quality.label, 'btn-outline-light': currentQuality !== quality.label }"
            @click="setQuality(quality)"
          >
            {{ quality.label }}
          </button>
        </div>
      </div>
    </div>
    
    <!-- 视频信息 -->
    <div v-if="showInfo" class="video-info mt-2">
      <div class="row">
        <div class="col-6">
          <span class="text-muted">
            <i class="bi bi-clock"></i> {{ formattedCurrentTime }} / {{ formattedDuration }}
          </span>
        </div>
        <div class="col-6 text-end">
          <span class="text-muted" v-if="resolution">
            <i class="bi bi-aspect-ratio"></i> {{ resolution }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, watch, computed } from 'vue'
import videojs from 'video.js'
import 'video.js/dist/video-js.css'
import 'video.js/dist/lang/zh-CN.json'

const props = defineProps({
  src: {
    type: String,
    required: true
  },
  poster: {
    type: String,
    default: ''
  },
  preload: {
    type: String,
    default: 'metadata'
  },
  autoPlay: {
    type: Boolean,
    default: false
  },
  showControls: {
    type: Boolean,
    default: true
  },
  muted: {
    type: Boolean,
    default: false
  },
  loop: {
    type: Boolean,
    default: false
  },
  mimeType: {
    type: String,
    default: ''
  },
  qualities: {
    type: Array,
    default: () => []
  },
  showInfo: {
    type: Boolean,
    default: true
  }
})

const emit = defineEmits(['play', 'pause', 'ended', 'error', 'timeupdate', 'loadedmetadata', 'ready'])

const containerRef = ref(null)
const videoRef = ref(null)
const speedMenuRef = ref(null)
const qualityMenuRef = ref(null)
const player = ref(null)
const loading = ref(true)
const isReady = ref(false)
const error = ref(false)
const errorMessage = ref('')
const isFullscreen = ref(false)
const speedMenuOpen = ref(false)
const qualityMenuOpen = ref(false)
const playbackRate = ref(1)
const currentQuality = ref('自动')
const resolution = ref('')

const playbackRates = [0.5, 0.75, 1, 1.25, 1.5, 2]

const formattedCurrentTime = computed(() => formatTime(player.value?.currentTime() || 0))
const formattedDuration = computed(() => formatTime(player.value?.duration() || 0))

function formatTime(seconds) {
  if (!seconds || isNaN(seconds)) return '00:00'
  const mins = Math.floor(seconds / 60)
  const secs = Math.floor(seconds % 60)
  return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
}

function initPlayer() {
  if (!videoRef.value) return
  
  const options = {
    autoplay: props.autoPlay,
    preload: props.preload,
    controls: props.showControls,
    muted: props.muted,
    loop: props.loop,
    poster: props.poster,
    playbackRates: playbackRates,
    responsive: true,
    fluid: true,
    html5: {
      hls: {
        enableLowInitialPlaylist: true,
        smoothQualityChange: true,
      },
      nativeAudioTracks: false,
      nativeVideoTracks: false,
    },
    controlBar: {
      children: [
        'playToggle',
        'volumePanel',
        'currentTimeDisplay',
        'timeDivider',
        'durationDisplay',
        'progressControl',
        'liveDisplay',
        'remainingTimeDisplay',
        'customControlSpacer',
        'playbackRateMenuButton',
        'qualitySelector',
        'fullscreenToggle'
      ]
    },
    language: 'zh-CN',
    languages: {
      'zh-CN': {
        'Play': '播放',
        'Pause': '暂停',
        'Current Time': '当前时间',
        'Duration Time': '持续时间',
        'Remaining Time': '剩余时间',
        'Stream Type': '流类型',
        'LIVE': '直播',
        'SEEK': '搜索',
        'loaded': '已加载',
        'Play Video': '播放视频',
        'Video Player': '视频播放器',
        'Close': '关闭',
        'Modal Window': '模态窗口',
        'This modal window can be moved by dragging the title bar': '此模态窗口可以通过拖动标题栏移动',
        'Close Modal Dialog': '关闭模态窗口',
        'End of dialog window': '对话框结束',
        'Beginning of dialog window': '对话框开始',
        'Escape begins a window': 'Escape 开始一个窗口',
        '(Or Dom Key)': '或主键',
        'Unmute': '取消静音',
        'Mute': '静音',
        'Captions/Subtitles': '字幕/标题',
        'Off': '关闭',
        'Quality Selector': '清晰度选择',
        'Seek': '搜索',
        'Volume': '音量',
        'Playback Rate': '播放速度',
        'Picture-in-Picture': '画中画',
        'Fullscreen': '全屏',
        'Non-Fullscreen': '非全屏',
        'Play Text': '播放文本',
        'Pause Text': '暂停文本',
      }
    }
  }
  
  player.value = videojs(videoRef.value, options, function() {
    isReady.value = true
    loading.value = false
    emit('ready', this)
    
    this.on('play', () => emit('play'))
    this.on('pause', () => emit('pause'))
    this.on('ended', () => emit('ended'))
    this.on('error', () => {
      error.value = true
      const err = this.error()
      errorMessage.value = err?.message || '视频加载失败'
      loading.value = false
      emit('error', err)
    })
    this.on('timeupdate', () => emit('timeupdate', this.currentTime()))
    this.on('loadedmetadata', () => {
      const tech = this.tech({ IWillNotUseThisInPlugins: true })
      if (tech && tech.videoWidth && tech.videoHeight) {
        resolution.value = `${tech.videoWidth}x${tech.videoHeight}`
      }
      emit('loadedmetadata')
    })
    this.on('fullscreenchange', () => {
      isFullscreen.value = this.isFullscreen()
    })
  })
}

function toggleSpeedMenu() {
  speedMenuOpen.value = !speedMenuOpen.value
  qualityMenuOpen.value = false
}

function toggleQualityMenu() {
  qualityMenuOpen.value = !qualityMenuOpen.value
  speedMenuOpen.value = false
}

function setPlaybackRate(rate) {
  playbackRate.value = rate
  if (player.value) {
    player.value.playbackRate(rate)
  }
  speedMenuOpen.value = false
}

function setQuality(quality) {
  currentQuality.value = quality.label
  qualityMenuOpen.value = false
}

function retry() {
  error.value = false
  loading.value = true
  if (player.value) {
    player.value.src({ src: props.src, type: props.mimeType || 'application/x-mpegURL' })
    player.value.load()
  }
}

function disposePlayer() {
  if (player.value) {
    player.value.dispose()
    player.value = null
  }
}

watch(() => props.src, (newSrc) => {
  if (player.value && newSrc) {
    player.value.src({ src: newSrc, type: props.mimeType || 'application/x-mpegURL' })
  }
})

onMounted(() => {
  initPlayer()
})

onBeforeUnmount(() => {
  disposePlayer()
})

defineExpose({
  player,
  play: () => player.value?.play(),
  pause: () => player.value?.pause(),
  togglePlay: () => player.value?.paused() ? player.value?.play() : player.value?.pause(),
  seek: (time) => player.value?.currentTime(time),
  setVolume: (volume) => player.value?.volume(volume),
  toggleFullscreen: () => player.value?.requestFullscreen(),
})
</script>

<style lang="scss" scoped>
.video-player-container {
  position: relative;
  width: 100%;
  background: #000;
  border-radius: 8px;
  overflow: hidden;
}

.video-player-wrapper {
  position: relative;
  
  &.fullscreen {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    z-index: 9999;
  }
}

:deep(.video-js) {
  width: 100%;
  height: 100%;
  font-family: inherit;
  
  .vjs-big-play-button {
    background-color: rgba(99, 102, 241, 0.8);
    border: none;
    border-radius: 50%;
    width: 80px;
    height: 80px;
    line-height: 80px;
    
    &:hover {
      background-color: rgba(99, 102, 241, 1);
    }
    
    .vjs-icon-placeholder:before {
      font-size: 40px;
    }
  }
  
  .vjs-control-bar {
    background-color: rgba(0, 0, 0, 0.7);
  }
  
  .vjs-play-progress,
  .vjs-volume-level {
    background-color: #6366f1;
  }
  
  .vjs-slider {
    background-color: rgba(255, 255, 255, 0.3);
  }
}

.video-loading {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.5);
  z-index: 10;
}

.video-error {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.8);
  z-index: 10;
  
  .error-content {
    text-align: center;
    color: white;
    
    i {
      font-size: 3rem;
      margin-bottom: 1rem;
    }
    
    p {
      margin-bottom: 1rem;
    }
  }
}

.playback-speed-menu,
.quality-menu {
  position: absolute;
  bottom: 60px;
  right: 60px;
  z-index: 20;
  
  .speed-options,
  .quality-options {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    margin-top: 0.5rem;
    background: rgba(0, 0, 0, 0.8);
    padding: 0.5rem;
    border-radius: 4px;
  }
}

.quality-menu {
  right: 120px;
}

.video-info {
  background: #1a1a1a;
  padding: 0.75rem;
}

@media (max-width: 768px) {
  .playback-speed-menu,
  .quality-menu {
    bottom: 50px;
    right: 50px;
    
    .speed-options,
    .quality-options {
      font-size: 0.875rem;
    }
  }
  
  .quality-menu {
    right: 100px;
  }
}
</style>
