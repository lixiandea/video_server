import apiClient from './client'

const videoApi = {
  getVideos(page = 1, pageSize = 12) {
    return apiClient.get('/videos', {
      params: { page, page_size: pageSize }
    })
  },
  
  getVideo(id) {
    return apiClient.get(`/videos/${id}`)
  },
  
  getVideoStream(id, quality = 'original') {
    return `${apiClient.defaults.baseURL}/videos/${id}/stream?quality=${quality}`
  },
  
  uploadVideo(formData, onProgress) {
    return apiClient.post('/videos', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
      onUploadProgress: (progressEvent) => {
        if (onProgress) {
          const percentCompleted = Math.round(
            (progressEvent.loaded * 100) / progressEvent.total
          )
          onProgress(percentCompleted)
        }
      }
    })
  },
  
  deleteVideo(id) {
    return apiClient.delete(`/videos/${id}`)
  },
  
  getMyVideos(page = 1, pageSize = 12) {
    return apiClient.get('/videos/my', {
      params: { page, page_size: pageSize }
    })
  },
  
  updateVideo(id, data) {
    return apiClient.put(`/videos/${id}`, data)
  },
  
  searchVideos(query, page = 1, pageSize = 12) {
    return apiClient.get('/videos/search', {
      params: { q: query, page, page_size: pageSize }
    })
  },
  
  getRelatedVideos(id, limit = 6) {
    return apiClient.get(`/videos/${id}/related`, {
      params: { limit }
    })
  }
}

export default videoApi
