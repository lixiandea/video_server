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
| eslint | 10.13.0 | 10.13.0 | 已升级（flat config） |
| eslint-plugin-vue | 10.11.1 | 10.11.1 | 已升级，兼容 eslint 8/9/10 |
| pinia | 4.1.0 | 4.1.0 | 已升级 |
| vue-router | 5.4.0 | 5.4.0 | 已升级 |
| video.js | 8.24.2 | 10.1.0（新架构） | 8.x 是最后一条经典 API 线；v10 为 `@videojs/html` 重写，见阶段 D |

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

### 阶段 D：video.js 8 → 10（重新评估，不是版本升级）

**调研结论（2026-10-19）**：

- video.js **没有 9.x**；`video.js@10` 是兼容外壳，对 v8 的 `videojs()` / `registerPlugin` /
  `getPlayer` 等用法直接抛出 `VJS8_LEGACY_*` 错误。
- v10 的实际实现是 `@videojs/html`（Web Components，支持 HTML / Vue / Svelte）或
  `@videojs/react`；HLS 等播放引擎通过 `@videojs/hlsjs-video` 之类的适配包接入。
- 因此"升级到 10"等价于**重写 `VideoPlayer.vue`**，而不是改 package.json。

三个选项：

| 选项 | 做法 | 代价 / 风险 | 建议 |
| --- | --- | --- | --- |
| A. 保持 8.24.2 | 播放器不动 | 无；8.24 是最新经典版，`npm audit` 为 0 | **短期推荐** |
| B. 迁移 `@videojs/html` | 自定义元素 + hlsjs 适配包 | 新 API / 皮肤，需重写控件、事件与语言；未知较多 | 长期跟随官方 |
| C. 原生 `<video>` + `hls.js` | 去掉 video.js，用已安装的 hls.js（1.7.3） | 需自己实现倍速/清晰度/全屏 UI；控件外观变化 | 轻量替代，可做原型 |

**当前决策**：本次保持 8.24.2（A）；B/C 作为独立项目评估，不在本轮升级里重写播放器。

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

## 6. 进度

- 2026-10-19：**阶段 A 完成**——eslint 10 + flat config（`eslint.config.js` + `@eslint/js` + `globals`），
  删除 `.eslintrc.cjs`；`npm run lint` 0 problem，build / Vitest 22 / e2e 19 全部通过。
- 2026-10-19：**阶段 B 完成**——pinia 2.3.1 → 4.1.0；store 为 setup 风格，无需改代码；
  已用新镜像重跑 e2e 19/19。
- 2026-10-19：**阶段 C 完成**——vue-router 4.6.4 → 5.4.0；路由 API 与守卫无需改代码；
  已用新镜像重跑 e2e 19/19。
- 2026-10-19：**阶段 D 重新评估**——video.js 10 是新架构（`@videojs/html`）的兼容外壳，
  升级等价于重写播放器；本轮保持 8.24.2，B/C 另立项目。
