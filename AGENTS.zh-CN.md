# 仓库贡献指南

## 项目结构与模块划分
- `cmd/` — 各服务入口：`api-server`、`scheduler`、`transcode-worker`。
- `internal/` — 应用代码：`handlers/`（Gin HTTP 层）、`services/`（业务逻辑）、`models/`（GORM 模型）、`middleware/`、`config/`、`utils/`。
- `pkg/` — 可复用库：`auth`、`database`、`redis`、`storage`、`queue`、`metrics`、`tracing`、`logging`、`errors`、`validation`。
- `api/` — 接口契约；`frontend-new/` — 唯一前端（Vue 3 + Vite），容器内由 Nginx 托管。
- 前端测试位于 `frontend-new/src/tests/{unit,e2e}`。脚本测试套件：`test-api.sh`、`test-monitoring.sh`。配置见 `config.yaml`、`docker/`、`deploy/`，文档见 `docs/`。

## 构建、测试与开发命令
- `make help` — 列出全部可用任务，优先使用它而不是零散命令。
- `./build.sh`（或 `make build`）— 构建 `api-server`、`scheduler`、`transcode-worker` 到 `bin/`。
- `go build ./...` — 编译全部 Go 包；提交前运行 `go vet ./...` 与 `go test ./...`。
- `go run ./cmd/api-server` — 本地启动 API（需 MySQL/Redis）。
- `make up` / `make down` — 启动 / 停止 compose 服务栈（会先执行一次性 `migrate` 服务）。
- `make migrate-up` / `make migrate-status` / `make migrate-down` — 数据库结构变更与回滚。
- `make test-api`（`./test-api.sh`）— 针对运行中服务的 curl 集成测试。
- `cd frontend-new && npm install && npm run dev` — 在 `:3000` 启动开发服务（等价 `make frontend-dev`）。
- `npm run build`、`npm run lint`、`npm test`（单元 + e2e）、`npm run test:unit`、`npm run test:e2e`。

## Docker 部署
- 先复制 `.env.example` 为 `.env`；`docker-compose.yml` 使用 `${VAR:?}`，缺失密钥会直接中止。用 `docker compose config -q` 校验配置。
- `make up`（`docker compose up -d`）启动全部 8 个服务（mysql、redis、api-server、scheduler、transcode-worker、frontend、prometheus、grafana）。前端 `:8000`（`FRONTEND_PORT`）、API `:8080`、Prometheus `:9090`、Grafana `:3301`（`GRAFANA_PORT`）。
- 主机保留端口时可覆盖 `FRONTEND_PORT` / `GRAFANA_PORT`（Windows/WSL2 常见，如 2952-3051 段）。
- 结构变更统一走版本化迁移：新增 `migrations/<4位序号>_<描述>.{up,down}.sql`（两者必须成对）并执行 `make migrate-up`。一次性 `migrate` 服务会在 api-server / scheduler / transcode-worker 之前跑完 `up`。服务不再自动迁移：`DB_AUTO_MIGRATE=true` 仅供本地/CI 起库，release 模式强制忽略。`docker/mysql/init.sql` 只负责建库。
- Alpine 基础镜像使用国内 apk 镜像，`frontend-new` 使用 `registry.npmmirror.com`；构建因网络失败时先检查这两处。
- `.dockerignore` 会排除 `node_modules/`、`dist/`、`.env` 与文档；新增构建输入时确认未被误排除。

## 编码风格与命名约定
- Go：使用 `gofmt`/`goimports` 格式化；包名小写单词；导出标识符用 PascalCase；文件名用 `snake_case.go`（如 `video_handler.go`）。错误用 `fmt.Errorf("...: %w", err)` 包装。
- HTTP 参数解析放在 `handlers/`，业务逻辑放在 `services/`；配置经 Viper 读取并支持环境变量覆盖（`DB_HOST`、`REDIS_ADDR`）。
- Vue：缩进 2 空格；组件用 PascalCase（`VideoDetail.vue`），store/composable 用 camelCase；通过 `@` 别名导入。使用 `npm run lint` 检查。

## 测试规范
- 前端单元测试：Vitest，位于 `src/tests/unit/*.test.js`；e2e：Playwright，位于 `src/tests/e2e/*.spec.js`。覆盖率：`npm run test:unit:coverage`。
- 后端单元测试与代码同目录（`internal/middleware/ratelimit_test.go`、`pkg/metrics/metrics_endpoint_test.go`），运行 `go test ./...`。CI（`.github/workflows/test.yml`）会构建服务并针对 MySQL/Redis 运行 API 脚本。
- e2e 需要 API 监听 `:8080`、前端开发服务监听 `:3000`。

## 提交与 Pull Request 规范
- 沿用仓库历史中的 Conventional Commits 前缀：`feat:`、`fix:`、`docs:`、`chore:`、`refactor:`、`perf:`、`test:`。允许中文描述（如 `fix: 修复转码队列`）。
- PR 目标分支为 `main`/`develop`，必须通过 CI，并说明改动内容、关联 issue、列出执行的测试命令；UI 改动需附截图。

## 安全与配置提示
- 不要提交真实凭据；泄露的旧口令与 JWT 密钥需人工轮换。`config.yaml`/`.env` 中的是开发密钥，部署时应通过环境变量注入。切勿提交 `bin/`、`node_modules/` 或 `storage/` 产物。
