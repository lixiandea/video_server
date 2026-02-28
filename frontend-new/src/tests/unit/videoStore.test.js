import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useVideoStore } from '@/stores/video'
import videoApi from '@/api/video'

// Mock videoApi
vi.mock('@/api/video', () => ({
  default: {
    getVideos: vi.fn(),
    getVideo: vi.fn(),
    uploadVideo: vi.fn(),
    deleteVideo: vi.fn(),
  },
}))

describe('Video Store', () => {
  let videoStore

  beforeEach(() => {
    setActivePinia(createPinia())
    videoStore = useVideoStore()
    vi.clearAllMocks()
  })

  it('应该初始化时为空状态', () => {
    expect(videoStore.videos).toEqual([])
    expect(videoStore.currentVideo).toBe(null)
    expect(videoStore.loading).toBe(false)
    expect(videoStore.error).toBe(null)
  })

  it('应该获取视频列表', async () => {
    const mockVideos = [
      { id: 1, title: 'Video 1' },
      { id: 2, title: 'Video 2' },
    ]
    const mockResponse = { data: { videos: mockVideos, total: 2 } }
    
    videoApi.getVideos.mockResolvedValue(mockResponse)

    const result = await videoStore.fetchVideos(1, 12)

    expect(videoApi.getVideos).toHaveBeenCalledWith(1, 12)
    expect(videoStore.videos).toEqual(mockVideos)
    expect(result).toEqual({ videos: mockVideos, total: 2 })
  })

  it('应该在获取视频列表时设置 loading 状态', async () => {
    const mockResponse = { data: { videos: [], total: 0 } }
    videoApi.getVideos.mockResolvedValue(mockResponse)

    const promise = videoStore.fetchVideos()
    
    expect(videoStore.loading).toBe(true)
    
    await promise
    
    expect(videoStore.loading).toBe(false)
  })

  it('应该在获取视频失败时设置 error 状态', async () => {
    const errorMessage = 'Network error'
    videoApi.getVideos.mockRejectedValue(new Error(errorMessage))

    await expect(videoStore.fetchVideos()).rejects.toThrow(errorMessage)
    expect(videoStore.error).toBe(errorMessage)
  })

  it('应该获取视频详情', async () => {
    const mockVideo = { id: 1, title: 'Video 1', description: 'Test' }
    videoApi.getVideo.mockResolvedValue({ data: mockVideo })

    const result = await videoStore.fetchVideoDetail(1)

    expect(videoApi.getVideo).toHaveBeenCalledWith(1)
    expect(videoStore.currentVideo).toEqual(mockVideo)
    expect(result).toEqual(mockVideo)
  })

  it('应该清除当前视频', () => {
    videoStore.currentVideo = { id: 1, title: 'Video 1' }
    
    videoStore.clearCurrentVideo()
    
    expect(videoStore.currentVideo).toBe(null)
  })

  it('应该删除视频', async () => {
    const mockVideos = [
      { id: 1, title: 'Video 1' },
      { id: 2, title: 'Video 2' },
    ]
    videoStore.videos = mockVideos
    
    videoApi.deleteVideo.mockResolvedValue({ data: { success: true } })

    await videoStore.deleteVideo(1)

    expect(videoApi.deleteVideo).toHaveBeenCalledWith(1)
    expect(videoStore.videos).toEqual([{ id: 2, title: 'Video 2' }])
  })
})
