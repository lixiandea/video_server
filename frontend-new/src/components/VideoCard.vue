<template>
  <router-link
    :to="`/video/${video.video_id}`"
    class="video-card-link text-decoration-none text-reset"
  >
    <article class="card h-100 video-card">
      <div class="video-thumb">
        <i class="bi bi-film"></i>
        <img
          v-if="video.cover_url"
          :src="video.cover_url + '?size=320'"
          :srcset="`${video.cover_url}?size=320 320w, ${video.cover_url}?size=720 720w`"
          sizes="(max-width: 768px) 50vw, 25vw"
          :alt="video.name"
          loading="lazy"
          @error="event => event.currentTarget.hidden = true"
        >
        <span v-if="video.duration" class="video-duration">{{ formatDuration(video.duration) }}</span>
      </div>
      <div class="card-body">
        <h6 class="card-title text-truncate">{{ video.name }}</h6>
        <p class="card-text small text-muted mb-1">
          <i class="bi bi-person"></i> {{ video.author_name || '未知' }}
        </p>
        <p class="card-text small text-muted mb-0 d-flex justify-content-between">
          <span><i class="bi bi-eye"></i> {{ video.view_count || 0 }}</span>
          <span class="text-truncate ms-2"><i class="bi bi-clock"></i> {{ video.display_time }}</span>
        </p>
        <small v-if="reason" class="recommendation-reason">{{ reason }}</small>
      </div>
    </article>
  </router-link>
</template>

<script setup>
defineProps({
  video: {
    type: Object,
    required: true,
  },
  reason: {
    type: String,
    default: '',
  },
})

function formatDuration(seconds) {
  const total = Math.round(Number(seconds) || 0)
  const hours = Math.floor(total / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  const secs = total % 60
  const pad = (value) => String(value).padStart(2, '0')

  return hours > 0 ? `${hours}:${pad(minutes)}:${pad(secs)}` : `${minutes}:${pad(secs)}`
}
</script>

<style scoped>
.video-card-link {
  display: block;
  height: 100%;
}

.video-card {
  cursor: pointer;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.video-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.15);
}

.video-thumb {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  aspect-ratio: 16 / 9;
  overflow: hidden;
  background: #e9ecef;
  color: #adb5bd;
  font-size: 2rem;
}

.video-thumb img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
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

.recommendation-reason {
  display: inline-block;
  margin-top: 0.5rem;
  color: #6366f1;
}

@media (prefers-reduced-motion: reduce) {
  .video-card {
    transition: none;
  }
}
</style>
