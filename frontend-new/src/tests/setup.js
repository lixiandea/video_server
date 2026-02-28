import { config } from '@vue/test-utils'

// 全局配置
config.global.mocks = {
  $router: {
    push: () => {},
    replace: () => {},
    go: () => {},
  },
  $route: {
    params: {},
    query: {},
    path: '/',
  },
}

// 全局组件
config.global.components = {}

// 全局插件
config.global.plugins = []

// 配置 localStorage mock
beforeEach(() => {
  const localStorageMock = {
    store: {},
    getItem: function(key) {
      return this.store[key] || null
    },
    setItem: function(key, value) {
      this.store[key] = value.toString()
    },
    removeItem: function(key) {
      delete this.store[key]
    },
    clear: function() {
      this.store = {}
    }
  }
  
  Object.defineProperty(global, 'localStorage', {
    value: localStorageMock,
    writable: true,
    configurable: true
  })
})
