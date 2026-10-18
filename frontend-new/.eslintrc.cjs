/* eslint-env node */
// ESLint 8 的经典配置（package.json 依赖 eslint ^8.57）。
// 构建产物与依赖目录必须显式忽略，否则会扫描 dist 下的打包结果。
module.exports = {
  root: true,
  env: {
    browser: true,
    es2022: true,
    node: true
  },
  parserOptions: {
    ecmaVersion: 'latest',
    sourceType: 'module'
  },
  // eslint-plugin-vue 10 起，Vue 3 的配置名不再带 vue3- 前缀
  extends: ['eslint:recommended', 'plugin:vue/essential'],
  rules: {
    // 以 _ 开头的参数/变量视为「有意未使用」（如回调占位）
    'no-unused-vars': ['error', { argsIgnorePattern: '^_', varsIgnorePattern: '^_' }],
    // 路由页面组件名不必强制多单词（Home.vue / Videos.vue 等）
    'vue/multi-word-component-names': 'off'
  },
  overrides: [
    {
      // Vitest 的全局 API（setup.js 使用 beforeEach，测试文件多用显式 import）
      files: ['src/tests/**/*.{js,vue}'],
      globals: {
        describe: 'readonly',
        it: 'readonly',
        test: 'readonly',
        expect: 'readonly',
        beforeEach: 'readonly',
        afterEach: 'readonly',
        vi: 'readonly'
      }
    }
  ],
  ignorePatterns: [
    'dist/',
    'node_modules/',
    'coverage/',
    'playwright-report/',
    'test-results/'
  ]
}
