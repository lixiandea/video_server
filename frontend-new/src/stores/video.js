import { defineStore } from 'pinia'
import { ref } from 'vue'
import videoApi from '@/api/video'

export const useVideoStore = defineStore('video', () => {
  const videos = ref([])
  const currentVideo = ref(null)
  const loading = ref(false)
  const error = ref(null)
  
  async function fetchVideos(page = 1, pageSize = 12) {
    loading.value = true
    error.value = null
    try {
      const response = await videoApi.getVideos(page, pageSize)
      videos.value = response.data.videos || []
      return response.data
    } catch (err) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }
  
  async function fetchVideoDetail(id) {
    loading.value = true
    error.value = null
    try {
      const response = await videoApi.getVideo(id)
      currentVideo.value = response.data
      return response.data
    } catch (err) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }
  
  async function uploadVideo(formData, onProgress) {
    loading.value = true
    error.value = null
    try {
      const response = await videoApi.uploadVideo(formData, onProgress)
      return response.data
    } catch (err) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }
  
  async function deleteVideo(id) {
    loading.value = true
    error.value = null
    try {
      const response = await videoApi.deleteVideo(id)
      videos.value = videos.value.filter(v => v.id !== id)
      return response.data
    } catch (err) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }
  
  function clearCurrentVideo() {
    currentVideo.value = null
  }
  
  return {
    videos,
    currentVideo,
    loading,
    error,
    fetchVideos,
    fetchVideoDetail,
    uploadVideo,
    deleteVideo,
    clearCurrentVideo
  }
})
