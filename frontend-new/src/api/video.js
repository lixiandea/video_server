import apiClient from './client'

// 后端视频列表项的统一字段：video_id / name / author_name / display_time / size / status / created_at
const videoApi = {
  // 视频列表：GET /videos?page=&limit=&q=
  getVideos(page = 1, pageSize = 12, query = '') {
    return apiClient.get('/videos', {
      params: { page, limit: pageSize, q: query || undefined }
    })
  },

  // 推荐：登录时个性化，匿名时返回热门/新上架内容。
  getRecommendations(limit = 8, excludeIds = []) {
    return apiClient.get('/recommendations', {
      params: {
        limit,
        exclude_ids: excludeIds.length ? excludeIds.join(',') : undefined,
      }
    })
  },

  // 搜索视频（同一端点，带 q 参数）
  searchVideos(query, page = 1, pageSize = 12) {
    return apiClient.get('/videos', {
      params: { page, limit: pageSize, q: query }
    })
  },

  getVideo(id) {
    return apiClient.get(`/videos/${id}`)
  },

  // 点赞 / 取消点赞，两个接口都幂等，返回 {liked, like_count}
  likeVideo(id) {
    return apiClient.post(`/videos/${id}/like`)
  },

  unlikeVideo(id) {
    return apiClient.delete(`/videos/${id}/like`)
  },

  // 换取短时效播放令牌（<video src> 无法携带 Authorization 头）
  getStreamToken(id) {
    return apiClient.get(`/videos/${id}/stream-token`)
  },

  // 拼接可直接绑到 <video src> 的地址
  getVideoStream(id, token, quality = 'original') {
    const query = new URLSearchParams({ token, quality })
    return `${apiClient.defaults.baseURL}/videos/${id}/stream?${query.toString()}`
  },

  // 多码率 HLS 地址：后端会把 token 补到 playlist 里的每个相对分片 URI
  getVideoHls(id, token) {
    const query = new URLSearchParams({ token })
    return `${apiClient.defaults.baseURL}/videos/${id}/hls/index.m3u8?${query.toString()}`
  },

  uploadVideo(formData, onProgress) {
    return apiClient.post('/videos/upload', formData, {
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

  // 我的视频：后端端点为 /users/videos（需登录）
  getMyVideos(page = 1, pageSize = 12) {
    return apiClient.get('/users/videos', {
      params: { page, limit: pageSize }
    })
  },

  getRelatedVideos(id, limit = 6) {
    return apiClient.get(`/videos/${id}/related`, {
      params: { limit }
    })
  }
}

export default videoApi
