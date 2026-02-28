import { createRouter, createWebHistory } from 'vue-router'
import Home from '@/views/Home.vue'
import Videos from '@/views/Videos.vue'
import VideoDetail from '@/views/VideoDetail.vue'
import VideoPlayer from '@/views/VideoPlayer.vue'
import Login from '@/views/Login.vue'
import Register from '@/views/Register.vue'
import Profile from '@/views/Profile.vue'
import Upload from '@/views/Upload.vue'

const routes = [
  {
    path: '/',
    name: 'Home',
    component: Home,
    meta: { title: '首页' }
  },
  {
    path: '/videos',
    name: 'Videos',
    component: Videos,
    meta: { title: '视频列表' }
  },
  {
    path: '/video/:id',
    name: 'VideoDetail',
    component: VideoDetail,
    meta: { title: '视频详情' }
  },
  {
    path: '/watch/:id',
    name: 'VideoPlayer',
    component: VideoPlayer,
    meta: { title: '播放视频' }
  },
  {
    path: '/login',
    name: 'Login',
    component: Login,
    meta: { title: '登录', requiresGuest: true }
  },
  {
    path: '/register',
    name: 'Register',
    component: Register,
    meta: { title: '注册', requiresGuest: true }
  },
  {
    path: '/profile',
    name: 'Profile',
    component: Profile,
    meta: { title: '个人中心', requiresAuth: true }
  },
  {
    path: '/upload',
    name: 'Upload',
    component: Upload,
    meta: { title: '上传视频', requiresAuth: true }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior(to, from, savedPosition) {
    if (savedPosition) {
      return savedPosition
    } else {
      return { top: 0 }
    }
  }
})

router.beforeEach((to, from, next) => {
  document.title = `${to.meta.title} - 视频服务器`
  
  const token = localStorage.getItem('token')
  
  if (to.meta.requiresAuth && !token) {
    next('/login')
  } else if (to.meta.requiresGuest && token) {
    next('/')
  } else {
    next()
  }
})

export default router
