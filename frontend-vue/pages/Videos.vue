<template>
  <div class="videos-page">
    <h1>视频列表</h1>
    <div class="video-grid">
      <div 
        v-for="video in videos" 
        :key="video.id" 
        class="video-card"
        @click="goToVideo(video.id)"
      >
        <div class="thumbnail">
          <img :src="video.thumbnail || '/static/default-thumbnail.jpg'" :alt="video.title" />
          <div class="duration">{{ video.duration }}</div>
        </div>
        <div class="video-info">
          <h3>{{ video.title }}</h3>
          <p class="description">{{ video.description }}</p>
          <div class="meta">
            <span class="views">{{ video.views }} 次观看</span>
            <span class="date">{{ video.uploadDate }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'Videos',
  asyncData({ store }) {
    // 模拟从API获取视频数据
    const videos = [
      {
        id: 1,
        title: 'Vue 3 入门教程',
        description: '学习 Vue 3 的基础知识和新特性',
        thumbnail: '/static/vue-tutorial.jpg',
        duration: '12:34',
        views: '12K',
        uploadDate: '2023-08-15'
      },
      {
        id: 2,
        title: '前端工程化实践',
        description: '现代前端开发的最佳实践',
        thumbnail: '/static/frontend-practice.jpg',
        duration: '25:42',
        views: '8.5K',
        uploadDate: '2023-08-10'
      },
      {
        id: 3,
        title: 'CSS Grid 布局详解',
        description: '掌握现代 CSS 布局技术',
        thumbnail: '/static/css-grid.jpg',
        duration: '18:21',
        views: '15K',
        uploadDate: '2023-08-05'
      },
      {
        id: 4,
        title: 'JavaScript ES6+ 新特性',
        description: '探索现代 JavaScript 的强大功能',
        thumbnail: '/static/js-es6.jpg',
        duration: '32:15',
        views: '22K',
        uploadDate: '2023-08-01'
      }
    ];
    
    store.commit('setVideos', videos);
  },
  computed: {
    videos() {
      return this.$store.state.videos;
    }
  },
  methods: {
    goToVideo(id) {
      this.$router.push(`/video/${id}`);
    }
  }
}
</script>

<style scoped>
.videos-page {
  padding: 2rem 0;
}

.video-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 1.5rem;
  margin-top: 2rem;
}

.video-card {
  background: white;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
  cursor: pointer;
  transition: transform 0.3s, box-shadow 0.3s;
}

.video-card:hover {
  transform: translateY(-5px);
  box-shadow: 0 10px 20px rgba(0, 0, 0, 0.15);
}

.thumbnail {
  position: relative;
  height: 180px;
  overflow: hidden;
}

.thumbnail img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.duration {
  position: absolute;
  bottom: 8px;
  right: 8px;
  background: rgba(0, 0, 0, 0.8);
  color: white;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 0.8rem;
}

.video-info {
  padding: 1rem;
}

.video-info h3 {
  margin-bottom: 0.5rem;
  color: #2c3e50;
  font-size: 1.1rem;
}

.description {
  color: #666;
  font-size: 0.9rem;
  margin-bottom: 0.5rem;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.meta {
  display: flex;
  justify-content: space-between;
  font-size: 0.8rem;
  color: #999;
}
</style>