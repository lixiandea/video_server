<template>
  <div class="video-detail">
    <div class="video-player-section">
      <div class="video-player">
        <!-- 视频播放器 -->
        <video 
          ref="videoPlayer" 
          :src="currentVideo?.videoUrl" 
          :poster="currentVideo?.thumbnail" 
          controls
          preload="metadata"
          class="player"
        >
          您的浏览器不支持视频播放。
        </video>
        
        <div class="video-info">
          <h1>{{ currentVideo?.title }}</h1>
          <div class="video-meta">
            <span class="views">{{ currentVideo?.views }} 次观看</span>
            <span class="date">{{ currentVideo?.uploadDate }}</span>
          </div>
        </div>
      </div>
      
      <div class="video-description">
        <p>{{ currentVideo?.description }}</p>
      </div>
    </div>
    
    <div class="video-sidebar">
      <h3>推荐视频</h3>
      <div 
        v-for="video in relatedVideos" 
        :key="video.id" 
        class="related-video"
        @click="switchVideo(video.id)"
      >
        <img :src="video.thumbnail" :alt="video.title" class="thumb" />
        <div class="info">
          <h4>{{ video.title }}</h4>
          <p>{{ video.channel }}</p>
          <p>{{ video.views }} 次观看 • {{ video.uploadDate }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import VideoPlayer from '../components/VideoPlayer.vue';

export default {
  name: 'VideoDetail',
  components: {
    VideoPlayer
  },
  props: ['id'],
  asyncData({ store, route }) {
    // 模拟从API获取视频详情
    const videoId = parseInt(route.params.id);
    const videoDetail = {
      id: videoId,
      title: `视频 ${videoId} - 详细内容`,
      description: `这是视频 ${videoId} 的详细描述。在这里您可以找到关于此视频的所有信息和相关内容。这个视频包含了丰富的内容，值得您花时间观看。`,
      thumbnail: `/static/video-${videoId}.jpg`,
      videoUrl: `/api/videos/${videoId}/stream`, // 实际使用时应替换为真实API
      duration: '10:25',
      views: '125K',
      uploadDate: '2023-08-15',
      channel: '频道名称'
    };
    
    store.commit('setCurrentVideo', videoDetail);
  },
  data() {
    return {
      relatedVideos: [
        {
          id: 101,
          title: '相关视频 1',
          thumbnail: '/static/related1.jpg',
          channel: '频道 1',
          views: '24K',
          uploadDate: '2023-08-10'
        },
        {
          id: 102,
          title: '相关视频 2',
          thumbnail: '/static/related2.jpg',
          channel: '频道 2',
          views: '18K',
          uploadDate: '2023-08-08'
        },
        {
          id: 103,
          title: '相关视频 3',
          thumbnail: '/static/related3.jpg',
          channel: '频道 3',
          views: '31K',
          uploadDate: '2023-08-05'
        },
        {
          id: 104,
          title: '相关视频 4',
          thumbnail: '/static/related4.jpg',
          channel: '频道 4',
          views: '15K',
          uploadDate: '2023-08-01'
        }
      ]
    }
  },
  computed: {
    currentVideo() {
      return this.$store.state.currentVideo;
    }
  },
  methods: {
    switchVideo(videoId) {
      this.$router.push(`/video/${videoId}`);
      // 在实际实现中，这里会调用API获取新视频数据
      this.loadVideoDetail(videoId);
    },
    loadVideoDetail(videoId) {
      // 模拟加载新视频数据
      const videoDetail = {
        id: videoId,
        title: `视频 ${videoId} - 详细内容`,
        description: `这是视频 ${videoId} 的详细描述。在这个视频中，您将了解到很多有趣的信息。`,
        thumbnail: `/static/video-${videoId}.jpg`,
        videoUrl: `/api/videos/${videoId}/stream`,
        duration: '8:42',
        views: '89K',
        uploadDate: '2023-08-12',
        channel: '频道名称'
      };
      
      this.$store.commit('setCurrentVideo', videoDetail);
    },
    onVideoPlay() {
      console.log('视频开始播放');
    },
    onVideoPause() {
      console.log('视频暂停');
    },
    onVideoEnded() {
      console.log('视频播放结束');
    }
  }
}
</script>

<style scoped>
.video-detail {
  display: flex;
  gap: 2rem;
  margin-top: 1rem;
}

.video-player-section {
  flex: 1;
}

.video-player {
  background: #000;
  border-radius: 8px;
  overflow: hidden;
  margin-bottom: 1rem;
}

.player {
  width: 100%;
  display: block;
  max-height: 60vh;
}

.video-info {
  padding: 1rem 0;
}

.video-info h1 {
  font-size: 1.5rem;
  margin-bottom: 0.5rem;
  color: #2c3e50;
}

.video-meta {
  display: flex;
  gap: 1rem;
  color: #666;
  font-size: 0.9rem;
}

.video-description {
  padding: 1rem 0;
  border-top: 1px solid #eee;
}

.video-description p {
  color: #555;
  line-height: 1.6;
}

.video-sidebar {
  width: 350px;
  flex-shrink: 0;
}

.video-sidebar h3 {
  font-size: 1.2rem;
  margin-bottom: 1rem;
  color: #2c3e50;
}

.related-video {
  display: flex;
  gap: 0.8rem;
  margin-bottom: 1rem;
  cursor: pointer;
  padding: 0.5rem;
  border-radius: 4px;
  transition: background-color 0.3s;
}

.related-video:hover {
  background-color: #f5f5f5;
}

.thumb {
  width: 120px;
  height: 68px;
  object-fit: cover;
  border-radius: 4px;
}

.info {
  flex: 1;
}

.info h4 {
  font-size: 0.95rem;
  margin-bottom: 0.3rem;
  color: #2c3e50;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.info p {
  font-size: 0.8rem;
  color: #666;
  margin-bottom: 0.2rem;
}

@media (max-width: 768px) {
  .video-detail {
    flex-direction: column;
  }
  
  .video-sidebar {
    width: 100%;
  }
}
</style>