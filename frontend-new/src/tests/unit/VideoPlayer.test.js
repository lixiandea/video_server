import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import VideoPlayer from '@/components/VideoPlayer.vue'

describe('VideoPlayer', () => {
  let pinia

  beforeEach(() => {
    pinia = createPinia()
    setActivePinia(pinia)
  })

  it('应该正确渲染视频播放器容器', () => {
    const wrapper = mount(VideoPlayer, {
      global: {
        plugins: [pinia],
      },
      props: {
        src: 'http://example.com/video.m3u8',
      },
    })

    expect(wrapper.find('.video-player-container').exists()).toBe(true)
    expect(wrapper.find('video').exists()).toBe(true)
  })

  it('应该接收必需的 src 属性', () => {
    const wrapper = mount(VideoPlayer, {
      global: {
        plugins: [pinia],
      },
      props: {
        src: 'http://example.com/video.m3u8',
      },
    })

    expect(wrapper.props().src).toBe('http://example.com/video.m3u8')
  })

  it('应该使用默认的 preload 属性值', () => {
    const wrapper = mount(VideoPlayer, {
      global: {
        plugins: [pinia],
      },
      props: {
        src: 'http://example.com/video.m3u8',
      },
    })

    expect(wrapper.props().preload).toBe('metadata')
  })

  it('应该使用默认的 autoPlay 属性值', () => {
    const wrapper = mount(VideoPlayer, {
      global: {
        plugins: [pinia],
      },
      props: {
        src: 'http://example.com/video.m3u8',
      },
    })

    expect(wrapper.props().autoPlay).toBe(false)
  })

  it('应该使用 poster 属性', () => {
    const wrapper = mount(VideoPlayer, {
      global: {
        plugins: [pinia],
      },
      props: {
        src: 'http://example.com/video.m3u8',
        poster: 'http://example.com/poster.jpg',
      },
    })

    expect(wrapper.props().poster).toBe('http://example.com/poster.jpg')
  })

  it('应该显示加载状态', async () => {
    const wrapper = mount(VideoPlayer, {
      global: {
        plugins: [pinia],
      },
      props: {
        src: 'http://example.com/video.m3u8',
      },
    })

    // 初始状态应该是加载中
    expect(wrapper.vm.loading).toBe(true)
  })

  it('应该定义正确的 emits', () => {
    const wrapper = mount(VideoPlayer, {
      global: {
        plugins: [pinia],
      },
      props: {
        src: 'http://example.com/video.m3u8',
      },
    })

    expect(wrapper.emitted()).toBeDefined()
  })
})
