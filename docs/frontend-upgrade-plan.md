# 前端升级规划（2026-10-19）

## 1. 现状

| 组件 | 当前版本 | 最新版本 | 说明 |
| --- | --- | --- | --- |
| Vue | 3.5.44 | 3.5.44 | 已是最新 3.x |
| Vite | 8.3.4 | 8.3.4 | 已升级（Rolldown） |
| Vitest | 5.0.5 | 5.0.5 | 已升级 |
| @vitejs/plugin-vue | 6.0.10 | 6.0.10 | 已升级 |
| jsdom | 30.1.3 | 30.1.3 | 已升级 |
| @playwright/test | 1.64.1 | 1.64.1 | 已升级 |
| sass | 1.105.2 | 1.105.2 | 已升级 |
| hls.js | 1.7.3 | 1.7.3 | 已是最新；当前代码未直接引用 |
| eslint | 8.57.1 | **10.13.0** | 待升级（flat config） |
| eslint-plugin-vue | 10.11.1 | 10.11.1 | 已升级，兼容 eslint 8/9/10 |
| pinia | 2.3.1 | **4.1.0** | 待升级 |
| vue-router | 4.6.4 | **5.4.0** | 待升级（peer 要求 pinia 3/4） |
| video.js | 8.24.2 | **10.1.0** | 待升级（播放器核心） |

`npm audit` 已为 0；本规划只做主版本对齐，不包含安全修复。

## 2. 约束与风险

- **顺序**：pinia 4 → vue-router 5（vue-router 5 的 peer 要求 `pinia ^3.0.4 || ^4.0.2`）；
  eslint 10 独立推进；video.js 10 放最后（改动集中在播放器）。
- **Node**：CI 与前端 Dockerfile 已是 Node 22，满足 eslint 10（`^20.19 || ^22.13 || >=24`）
  与 vue-router 5（Vite 8）的要求。
- **eslint 10**：不再支持 `.eslintrc`，需迁移到 `eslint.config.js`（flat config），
  显式引入 `@eslint/js`，并使用 `eslint-plugin-vue` 的 `flat/essential`；
  `package.json` 的 `lint` 脚本去掉 `--ext`（flat config 由 `files` 匹配）。
- **video.js 10**：`VideoPlayer.vue` 依赖 `html5.hls` 选项、`controlBar.children` 自定义控件、
  `playbackRates`、中文语言包与 `dispose/src()`；升级前先读 v10 迁移说明，升级后必须手动验证
  HLS 多码率播放、mp4 回退、清晰度/倍速控件、全屏与移动端。
- **hls.js**：当前未被直接 import（video.js 8+ 自带 VHS）；若升级后仍无引用，考虑移除依赖。
- 每个主版本单独提交，失败可单独回滚；不把多个主版本混在一个提交里。

## 3. 分阶段计划

### 阶段 A：eslint 10 + flat config（低风险）

1. 新增 `eslint.config.js`：`@eslint/js` recommended + `eslint-plugin-vue/flat/essential`，
   保留现有规则（`no-unused-vars` 的 `_` 例外、`vue/multi-word-component-names: off`）与忽略目录；
2. 升级 `eslint@^10`，删除 `.eslintrc.cjs`，调整 `npm run lint`；
3. 验收：`npm run lint` 0 problem；`vitest` / `build` 不受影响；CI frontend 作业通过。

### 阶段 B：pinia 4

1. 升级 `pinia@^4`（peer：vue `^3.5.11`，当前 3.5.44 ✓）；
2. 检查 store 定义（`defineStore`、`storeToRefs`）与登录态持久化逻辑；
3. 验收：单测 22 + e2e 19 + 登录/上传/列表手动回归。

### 阶段 C：vue-router 5

1. 升级 `vue-router@^5`（peer：vue `^3.5.34`、vite `^7.3 || ^8`、pinia `^4` ✓）；
2. 检查路由守卫、`router.push`、懒加载与 `RouterView`；
3. 验收：e2e 导航用例 + 登录跳转/详情页/404 手动回归。

### 阶段 D：video.js 10（高风险）

1. 升级 `video.js@^10`，按迁移说明调整 `VideoPlayer.vue`（VHS、控件、语言包、事件）；
2. 手动覆盖：HLS 多码率播放、mp4 回退、倍速、全屏、移动端；
3. 验收：上述手动用例 + 现有 e2e；必要时补一个播放器冒烟用例。

### 阶段 E：清理与收尾

- 评估移除未使用的 `hls.js`；同步 README / AGENTS 的技术栈版本；
- 全量验收：`npm ci` / `build` / `lint` / `vitest` / `playwright` / `npm audit`，CI 全绿。

## 4. 每阶段验收基线

- `npm ci` 成功，`npm run build` 无错误；
- `npm run lint` 0 problem，`npx vitest run` 全绿（22 用例）；
- `E2E_BASE_URL=http://localhost:8000 npx playwright test --project=chromium` 19/19；
- CI 四个作业全绿；任一失败即回滚该阶段提交。

## 5. 预估

- 阶段 A/B/C 各约 0.5–1 天（含回归）；
- 阶段 D 约 1–2 天（以播放器手工验证为主）。
