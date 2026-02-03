import { createSSRApp } from 'vue'
import { createStore } from 'vuex'
import { createRouter, createWebHistory, createMemoryHistory } from 'vue-router'
import App from './App.vue'
import Home from '../pages/Home.vue'
import VideoDetail from '../pages/VideoDetail.vue'

export function createApp(url) {
  const store = createStore({
    state() {
      return {
        videos: [],
        currentVideo: null
      }
    },
    mutations: {
      setVideos(state, videos) {
        state.videos = videos
      },
      setCurrentVideo(state, video) {
        state.currentVideo = video
      }
    }
  })

  // 根据环境选择历史记录类型
  const history = typeof window !== 'undefined' 
    ? createWebHistory() 
    : createMemoryHistory()

  const router = createRouter({
    history,
    routes: [
      { path: '/', component: Home },
      { path: '/videos', component: Videos },
      { path: '/about', component: About },
      { path: '/video/:id', component: VideoDetail, props: true }
    ]
  })

  const app = createSSRApp(App)
  app.use(store)
  app.use(router)

  return { app, store, router }
}