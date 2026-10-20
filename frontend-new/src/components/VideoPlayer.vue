<template>
  <div class="video-player-container" ref="containerRef">
    <div class="video-player-wrapper">
      <video
        ref="videoRef"
        class="native-video"
        :poster="poster"
        :preload="preload"
        :autoplay="autoPlay"
        :controls="showControls"
        :muted="muted"
        :loop="loop"
        playsinline
        webkit-playsinline
      >
        您的浏览器不支持 video 标签。
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

      <!-- 回退提示：HLS 不支持/失败时自动切到 MP4 -->
      <div v-if="usingFallback" class="video-fallback-hint">
        <i class="bi bi-info-circle"></i> 已回退到 MP4 播放
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

      <!-- 清晰度选择（来自 HLS 码率档位） -->
      <div v-if="availableQualities.length > 1 && showControls" class="quality-menu" ref="qualityMenuRef">
        <button
          class="btn btn-sm btn-outline-light quality-btn"
          @click="toggleQualityMenu"
          title="清晰度"
        >
          <i class="bi bi-hd"></i> {{ currentQuality }}
        </button>
        <div v-show="qualityMenuOpen" class="quality-options">
          <button
            v-for="quality in availableQualities"
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
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import Hls from 'hls.js'

const props = defineProps({
  src: {
    type: String,
    required: true,
  },
  // HLS 不支持或播放失败时回退的 MP4 地址（可选）
  fallbackSrc: {
    type: String,
    default: '',
  },
  poster: {
    type: String,
    default: '',
  },
  preload: {
    type: String,
    default: 'metadata',
  },
  autoPlay: {
    type: Boolean,
    default: false,
  },
  showControls: {
    type: Boolean,
    default: true,
  },
  muted: {
    type: Boolean,
    default: false,
  },
  loop: {
    type: Boolean,
    default: false,
  },
  showInfo: {
    type: Boolean,
    default: true,
  },
})

const emit = defineEmits(['play', 'pause', 'ended', 'error', 'timeupdate', 'loadedmetadata', 'ready'])

const containerRef = ref(null)
const videoRef = ref(null)
const speedMenuRef = ref(null)
const qualityMenuRef = ref(null)

const loading = ref(true)
const isReady = ref(false)
const error = ref(false)
const errorMessage = ref('')
const usingFallback = ref(false)
const speedMenuOpen = ref(false)
const qualityMenuOpen = ref(false)
const playbackRate = ref(1)
const currentQuality = ref('自动')
const availableQualities = ref([])
const currentTime = ref(0)
const duration = ref(0)
const resolution = ref('')

const playbackRates = [0.5, 0.75, 1, 1.25, 1.5, 2]

let hls = null

const formattedCurrentTime = computed(() => formatTime(currentTime.value))
const formattedDuration = computed(() => formatTime(duration.value))

function formatTime(seconds) {
  if (!seconds || isNaN(seconds)) return '00:00'
  const mins = Math.floor(seconds / 60)
  const secs = Math.floor(seconds % 60)
  return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
}

function isHlsSource(url) {
  return /\.m3u8(\?|$)/i.test(url || '')
}

function destroyHls() {
  if (hls) {
    hls.destroy()
    hls = null
  }
}

function fail(message) {
  error.value = true
  errorMessage.value = message
  loading.value = false
  emit('error', new Error(message))
}

// 回退到 MP4：HLS 不受支持、HLS 致命错误或视频源不可用时调用
function switchToFallback() {
  const video = videoRef.value
  if (!video || !props.fallbackSrc) {
    fail('视频播放失败')
    return
  }

  destroyHls()
  usingFallback.value = true
  availableQualities.value = []
  currentQuality.value = '自动'
  loading.value = true
  isReady.value = false
  video.src = props.fallbackSrc
}

function attachSource() {
  const video = videoRef.value
  if (!video) return

  destroyHls()
  error.value = false
  errorMessage.value = ''
  loading.value = true
  isReady.value = false
  usingFallback.value = false
  availableQualities.value = []
  currentQuality.value = '自动'

  if (!isHlsSource(props.src)) {
    video.src = props.src
    return
  }

  if (Hls.isSupported()) {
    const instance = new Hls({ enableWorker: true })
    hls = instance

    instance.on(Hls.Events.MANIFEST_PARSED, () => {
      const levels = instance.levels || []
      availableQualities.value = [
        { label: '自动', level: -1 },
        ...levels.map((level, index) => ({
          label: level.height ? `${level.height}p` : `码率 ${index + 1}`,
          level: index,
        })),
      ]
    })

    instance.on(Hls.Events.LEVEL_SWITCHED, (_event, data) => {
      const level = instance.levels[data.level]
      if (level?.height) {
        resolution.value = `${level.width || '?'}x${level.height}`
      }
    })

    instance.on(Hls.Events.ERROR, (_event, data) => {
      if (!data.fatal) return
      console.warn('HLS 播放失败:', data.type, data.details)
      switchToFallback()
    })

    instance.loadSource(props.src)
    instance.attachMedia(video)
    return
  }

  // Safari 等支持原生 HLS 的浏览器
  if (video.canPlayType('application/vnd.apple.mpegurl')) {
    video.src = props.src
    return
  }

  switchToFallback()
}

function onLoadedMetadata() {
  const video = videoRef.value
  if (!video) return
  duration.value = Number.isFinite(video.duration) ? video.duration : 0
  if (video.videoWidth && video.videoHeight) {
    resolution.value = `${video.videoWidth}x${video.videoHeight}`
  }
  loading.value = false
  if (!isReady.value) {
    isReady.value = true
    emit('ready', video)
  }
  emit('loadedmetadata')
}

function onTimeUpdate() {
  currentTime.value = videoRef.value?.currentTime || 0
  emit('timeupdate', currentTime.value)
}

function onPlay() {
  emit('play')
}

function onPause() {
  emit('pause')
}

function onEnded() {
  emit('ended')
}

function onVideoError() {
  // HLS 源出错时优先回退到 MP4；MP4 也失败则显示错误
  if (!usingFallback.value && props.fallbackSrc) {
    switchToFallback()
    return
  }
  fail('视频加载失败')
}

function bindVideoEvents() {
  const video = videoRef.value
  if (!video) return
  video.addEventListener('loadedmetadata', onLoadedMetadata)
  video.addEventListener('timeupdate', onTimeUpdate)
  video.addEventListener('play', onPlay)
  video.addEventListener('pause', onPause)
  video.addEventListener('ended', onEnded)
  video.addEventListener('error', onVideoError)
}

function unbindVideoEvents() {
  const video = videoRef.value
  if (!video) return
  video.removeEventListener('loadedmetadata', onLoadedMetadata)
  video.removeEventListener('timeupdate', onTimeUpdate)
  video.removeEventListener('play', onPlay)
  video.removeEventListener('pause', onPause)
  video.removeEventListener('ended', onEnded)
  video.removeEventListener('error', onVideoError)
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
  if (videoRef.value) {
    videoRef.value.playbackRate = rate
  }
  speedMenuOpen.value = false
}

function setQuality(quality) {
  currentQuality.value = quality.label
  qualityMenuOpen.value = false
  if (hls && typeof quality.level === 'number') {
    // -1 表示自动码率
    hls.currentLevel = quality.level
  }
}

function retry() {
  attachSource()
}

onMounted(() => {
  bindVideoEvents()
  attachSource()
})

onBeforeUnmount(() => {
  unbindVideoEvents()
  destroyHls()
})

watch(() => props.src, () => attachSource())

defineExpose({
  player: videoRef,
  play: () => videoRef.value?.play(),
  pause: () => videoRef.value?.pause(),
  togglePlay: () => {
    const video = videoRef.value
    if (!video) return
    return video.paused ? video.play() : video.pause()
  },
  seek: (time) => {
    if (videoRef.value) videoRef.value.currentTime = time
  },
  setVolume: (volume) => {
    if (videoRef.value) videoRef.value.volume = volume
  },
  toggleFullscreen: () => {
    const video = videoRef.value
    if (video?.requestFullscreen) return video.requestFullscreen()
  },
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
  aspect-ratio: 16 / 9;
  background: #000;
}

.native-video {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
  background: #000;
  font-family: inherit;
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

.video-fallback-hint {
  position: absolute;
  top: 12px;
  left: 12px;
  z-index: 20;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.7);
  color: #fff;
  font-size: 0.8rem;
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
</style>
