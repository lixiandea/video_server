# 项目分析与优化路线图

> 适用仓库：`video_server`
> 更新日期：2026-09-27
> 状态图例：⬜ 待办 · 🚧 进行中 · ✅ 已完成 · ⏸ 暂缓

## 1. 背景与目标

本文档用于追踪 `video_server` 的现状缺陷与优化落地，目标是：

- 收敛已知的安全、正确性与工程问题，形成可逐项执行的清单。
- 每项任务都有明确范围、验收标准与状态，便于增量提交与逐步验收。
- 作为后续迭代与 Code Review 的对齐基线。

## 2. 项目结构概览

| 区域 | 说明 |
| --- | --- |
| `cmd/` | 服务入口：`api-server`、`scheduler`、`worker`、`transcode-worker` |
| `internal/` | 业务代码，handler → service → model 分层，含 `config/`、`middleware/`、`utils/` |
| `pkg/` | 共享库：`auth`、`database`、`redis`、`storage`、`queue`、`metrics`、`tracing`、`logging`、`errors`、`validation` |
| `frontend-new/` | 主用前端（Vue 3 + Vite，含 Vitest / Playwright 测试） |
| `frontend-vue/` | Vue 3 SSR（webpack）前端 |
| `frontend/` | 早期静态前端 |
| `api/` | 接口契约定义 |
| `deploy/`、`docker/`、`docker-compose.yml` | 部署与监控编排 |
| `docs/` | 项目文档 |

## 3. 缺陷清单

### P0 安全与可用性

- **P0-1 凭据入库 / JWT 默认密钥**：`.env`、`config.yaml`、`docker-compose.yml`、`.github/workflows/test.yml` 含明文数据库口令；`pkg/auth/auth.go` 在未设置 `JWT_SECRET` 时回退到可预测的默认值。拿到仓库即可伪造 token。
- **P0-2 缓存永不失效**：`internal/services/video_service.go`、`user_service.go` 的更新 / 删除操作不清 `video:`、`user:` 键，删除视频或改密后最长 5 / 10 分钟仍返回旧数据。
- **P0-3 容器存储路径不一致**：各 Dockerfile 最终 `WORKDIR /root/`，默认写入 `/root/storage`；而 compose 挂载 `/app/storage`。api-server 与 transcode-worker 目录不通，转码必然找不到源文件。
- **P0-4 上传无边界**：`server.max_file_size` 定义后未使用，未设置 `MaxMultipartMemory` / `http.MaxBytesReader`，且仅按扩展名校验，可上传任意内容造成 DoS。
- **P0-5 转码 goroutine 空指针**：Redis 不可用时 `taskQueue` 为 nil，上传处理器仍在其 goroutine 中调用，导致进程 panic。
- **P0-6 CORS 配置非法**：`Access-Control-Allow-Origin: *` 与 `Allow-Credentials: true` 并用，浏览器会拒绝，且无来源白名单。
- **P0-7 数据库连接失败直接 panic**：`pkg/database/database.go` 在连接失败时直接 `panic`，无重试 / 退避与优雅退出，容器编排下会反复崩溃重启。

### P1 正确性与契约

- **P1-1 前后端接口不匹配**：不止端点缺失，而是端点、参数、字段三层都不一致。端点上：`GET /videos`、`GET /videos/:id/related` 后端未实现，`POST /videos`（应 `/videos/upload`）、`GET /videos/my`（应 `/users/videos`）、`GET /videos/search` 路径不符；参数上前端传 `page_size` 而后端读 `limit`、上传用 `title`/`video` 而后端读 `name`/`file`；字段上列表/详情用 `id`/`title`/`author`/`cover_url`/`view_count`，后端返回 `video_id`/`name`/`author_name`/`display_time`。另有 `Register.vue` 调用参数错位（`register(username, email, password)` vs 签名 `(username, password, email)`），导致注册功能实际不可用。
- **P1-2 流媒体鉴权导致 401**：`stream` 路由需要 `Authorization`，而 `VideoPlayer.vue` 通过 `<source :src>` 引用，媒体元素无法携带请求头（已按 5.14 的播放令牌方案处理）。
- **P1-3 评论分页 total 错误**：`total` 取 `len(commentList)`，恒等于当前页条数（已按 5.15 修正）。
- **P1-4 N+1 查询**：视频列表、评论列表逐条查询作者信息，无 `Preload` 或 join（已按 5.15 消除评论列表 N+1）。
- **P1-5 错误处理不统一**：not found 返回 500；`pkg/errors` 与 `internal/utils/response.go` 定义后从未使用（已按 5.16 统一）。
- **P1-6 转码链路不闭环**：转码产物路径不写回 `file_path`；无重试与 pending 回收；缺 ffmpeg 时假成功（已按 5.17 处理）。
- **P1-8 手工 DDL 与模型冲突导致迁移中断**：`docker/mysql/init.sql` 手工创建了 `comments`（`id VARCHAR(255)`、外键指向代码中已不存在的 `video_info`）等旧结构，与 `internal/models` 不一致；`AutoMigrate` 在 `comments` 上失败后**整体中断**，后续 `video_deletion_records` 从未创建。后果：删除视频报 `Error 1146 Table ... doesn't exist`，`/api/v1/transcode/queue/stats` 返回 500。
- **P1-9 多服务并发迁移**：`api-server`/`scheduler`/`worker` 启动时并行执行 `AutoMigrate`，日志出现 `Table 'x' already exists`、`Duplicate key name ...` 等噪声，并可能留下半成品表。
- **P1-10 队列统计在消费者组缺失时返回 500**：转码 worker 从未消费过队列时，`XPending` 返回 `NOGROUP`，`GetQueueStats` 将其视为失败，接口直接 500。
- **P1-7 会话机制缺失**：`Session` 表与 `session_id` 未落库校验，无法登出 / 吊销；注销账户不清理其视频与评论。

### P2 工程与可维护性

- **P2-1 `cmd/worker` 嵌套独立 module**：`cmd/worker/go.mod` 声明 `go 1.23.0`（根模块为 `go 1.21`），根目录 `go build ./...`、`go vet ./...` 覆盖不到；实测 `./build.sh` 在 worker 步骤因需要 go1.23 工具链而失败，且该脚本本身不构建 `transcode-worker`。该服务本身为空壳却进入生产编排。
- **P2-2 三套前端并存**，且 `frontend-new/dist/` 构建产物入库。
- **P2-3 限流器 map 无淘汰**（内存泄漏）；`/metrics` 公开无鉴权；Grafana 使用默认口令 `admin`。
- **P2-4 缺少 Go 测试**，CI 只跑 curl 冒烟，且不执行前端测试。
- **P2-5 启动即 `AutoMigrate`** 并全量打印 SQL，无版本化迁移。
- **P2-6 脚本重复**（多个 `start-frontend*.sh`），README 存在失效绝对路径链接，无 Makefile。
- **P2-7 前端 lint 不可用**：`frontend-new` 缺少 ESLint 配置文件，`package.json` 中的 `npm run lint` 实际报错退出。
- **P2-10 全仓库不满足 gofmt**：`gofmt -l pkg internal cmd` 报告 **24 个文件**未格式化（缩进空格、对齐、行尾空白），因此无法在 CI 强制格式化检查。属既有状态，修复会产生大范围空白 diff，建议单独一次提交处理。P1 阶段改动的文件都沿用原风格，未新增格式化债务（其中重写的 `internal/utils/response.go` 已是 gofmt 干净状态）。
- **P2-9 前端健康检查恒为 unhealthy**：`HEALTHCHECK` 探测 `http://localhost/`，容器内 `localhost` 解析为 IPv6 `::1`，而 nginx 仅监听 IPv4，容器长期标记 unhealthy。
- **P2-8 git 跟踪与忽略规则失效**：`frontend-new/dist/` 构建产物仍被跟踪；忽略规则逐个目录硬编码（`frontend-vue/node_modules/`），`frontend-new/node_modules/`、`frontend/node_modules/` 未被覆盖。实测 `node_modules` **本身未被跟踪**（`git ls-files | grep -c node_modules` = 0），问题在于忽略规则不全导致依赖目录反复出现在 `git status` 中（本轮已处理，见 5.12）。

## 4. 优化路线图

### 阶段一：安全与可用性（P0）

目标：堵住凭据外泄、缓存脏读、容器路径与上传边界问题。

| ID | 任务 | 主要涉及 | 验收标准 |
| --- | --- | --- | --- |
| P0-1 | 密钥外置与轮换 | `.env`、`config.yaml`、`docker-compose.yml`、CI、`pkg/auth` | 仓库无明文密钥；`JWT_SECRET` 缺失时启动失败；`.env` 被忽略 |
| P0-2 | 写操作后失效缓存 | `internal/services/video_service.go`、`user_service.go` | 删除 / 修改后立即读到新值，含回归测试 |
| P0-3 | 统一容器存储路径 | `Dockerfile*`、`docker-compose.yml`、`config.yaml` | 上传 → 转码 → 播放在 compose 中全链路可用 |
| P0-4 | 上传体积与类型限制 | `internal/handlers/video_handler.go`、`pkg/validation` | 超限请求返回 413 / 400；伪造扩展名被拒 |
| P0-5 | 修复 nil 队列崩溃 | `internal/handlers/video_handler.go` | Redis 不可用时上传仍成功且不 panic |
| P0-6 | CORS 白名单 | `internal/middleware/auth.go` | 带凭据的跨域请求通过，非白名单来源被拒 |
| P0-7 | 数据库连接重试与优雅退出 | `pkg/database/database.go`、`cmd/*/main.go` | 连接失败有限重试；无法启动时以非 0 退出码结束且不 panic |

### 阶段二：正确性与契约（P1）

目标：让前后端契约与数据语义一致。

| ID | 任务 | 主要涉及 | 验收标准 |
| --- | --- | --- | --- |
| P1-1 | 对齐接口契约 | `frontend-new/src/api/*`、`internal/handlers/*` | 前端调用的端点、参数、字段三层一致 |
| P1-2 | 流媒体鉴权方案 | `pkg/auth`、`internal/middleware`、`VideoPlayer.vue` | 视频可正常播放，未授权访问仍受控，且令牌不长期暴露在 URL |
| P1-3 | 修正评论分页 | `internal/services/comment_service.go` | `total` 返回真实总数 |
| P1-4 | 消除 N+1 | 各 service | 列表查询的数据库调用次数为常数 |
| P1-5 | 统一错误码与响应 | `pkg/errors`、`internal/utils/response.go`、handlers | 错误使用统一结构，not found 返回 404 |
| P1-6 | 转码闭环 | `internal/services/transcode_service.go`、`pkg/queue` | 产物落库、失败可重试、pending 可回收 |
| P1-7 | 会话与账户生命周期 | `pkg/auth`、`internal/services/*` | 支持登出 / 吊销，注销账户级联清理 |
| P1-8 | 统一 DDL 与模型 | `docker/mysql/init.sql`、`pkg/database` | 手工 DDL 不再与模型冲突，迁移可逐表容错 |
| P1-9 | 串行化启动迁移 | `pkg/database` | 多服务并发启动无 DDL 报错 |
| P1-10 | 队列统计容错 | `pkg/queue/task_queue.go` | 无消费者组时统计返回 200 且 pending=0 |
| P1-11 | 扩展视频/用户字段 | `internal/models`、`frontend-new/src/views/*` | 点赞、播放量、封面、时长、简介、邮箱等字段落库后再由前端消费 |

### 阶段三：工程收敛（P2）

目标：统一模块、前端与交付入口，建立质量门禁。

| ID | 任务 | 主要涉及 | 验收标准 |
| --- | --- | --- | --- |
| P2-1 | 合并 `cmd/worker` 并决策去留 | `cmd/worker`、`go.mod` | 根目录 `go build ./...` 覆盖全部服务 |
| P2-2 | 前端收敛与产物出库 | `frontend*`、`.gitignore` | 仅保留主前端，`dist/` 不再入库 |
| P2-3 | 限流器淘汰与可观测性加固 | `internal/middleware/auth.go`、`pkg/metrics` | map 有 TTL 回收；`/metrics` 受保护 |
| P2-4 | 测试与 CI 门禁 | `*_test.go`、`.github/workflows/test.yml` | CI 执行 `go vet`、`go test` 与前端测试 |
| P2-5 | 版本化数据库迁移 | `pkg/database`、`migrations/` | 迁移可回滚，生产不再 AutoMigrate |
| P2-6 | 脚本收敛与文档修正 | `*.sh`、`Makefile`、`README.md` | 统一入口命令，失效链接修复 |
| P2-7 | 修复前端 lint 配置 | `frontend-new/.eslintrc*` | `npm run lint` 可执行并纳入 CI |
| P2-8 | 修正 git 跟踪与忽略规则 | `.gitignore`、`bin/`、`frontend-new/node_modules` | `bin/`、`dist/`、`storage/` 产物出库，依赖目录被统一忽略 |
| P2-9 | 修复前端健康检查 | `frontend-new/Dockerfile.frontend.nginx` | 容器健康状态为 healthy |
| P2-10 | 全仓 gofmt 收敛 | `pkg/**`、`internal/**`、`cmd/**` | `gofmt -l` 输出为空并纳入 CI |

### 阶段四：长期演进

- **可观测性**：完善指标与告警规则，定义 SLO。
- **性能与扩展**：复核分页与索引，流媒体升级 HLS，压测队列吞吐与水平扩展。
- **安全**：依赖漏洞扫描、密钥管理集成、审计日志。

## 5. 验证记录（2026-09-27）

### 5.1 环境

- 系统 Go 为 1.20.2，低于 `go.mod` 要求的 1.21；临时安装 Go 1.21.13 到 `/tmp/go121` 用于验证，未改动系统环境。
- Node 18.15.0 / npm 9.6.1。
- 首轮验证（5.2 / 5.3）时**无 MySQL 服务、WSL 内无法使用 Docker**；该限制已在 5.11 解除：Docker Desktop 的 WSL 集成可用，全部服务以容器方式运行。

### 5.2 执行结果

| 命令 / 操作 | 结果 | 说明 |
| --- | --- | --- |
| `go build ./...`（Go 1.21.13） | ✅ | 主模块全部包编译通过 |
| `go vet ./...`（Go 1.21.13） | ✅ | 无告警 |
| `go build ./...`（系统 Go 1.20.2） | ❌ | `go.mod` 要求 go 1.21，直接拒绝 |
| `./build.sh`（Go 1.21.13） | ❌ | 在 worker 步骤失败；仅重建 api-server、scheduler，`bin/worker` 仍是旧产物 |
| `./build.sh` 是否构建 transcode-worker | ❌ | 脚本只编译 3 个服务，缺少 transcode-worker |
| 根模块是否覆盖 `cmd/worker` | ❌ | 嵌套 module，`go list ./...` 不包含 |
| `cd cmd/worker && go build .` | ❌ | 触发 go1.23.0 工具链下载 |
| `npm ci`（frontend-new） | ✅ | 安装 357 个包 |
| `npx vitest --run` | ✅ | 3 个文件 / 20 个用例全部通过 |
| `npm run build` | ✅ | 构建成功；提示 `video-js` chunk 689 KB 超过 500 KB |
| `npm run lint` | ❌ | 缺少 ESLint 配置文件，脚本报错退出 |
| 启动 Redis / api-server | ❌ | Redis 返回 `PONG`；api-server 因 MySQL 连接被拒 panic，退出码 2 |
| 端到端 上传 → 转码 → 播放 | ⏸ | 缺少 MySQL 与 Docker，无法验证 |

### 5.3 结论（P0 实施前的初始状态）

- 缺陷清单 **19 项全部复现**，无任何优化已落地，因此路线图状态全部保持 ⬜，文档记录与实际一致。
- 新增 3 项问题：P0-7、P2-7、P2-8；并为 P2-1 补充 `build.sh` 失败与漏构建证据。
- P1-1 的契约缺口由文档中的 5 处修正为实测 7 处。
- 运行时验证受环境限制：无 MySQL、无 Docker，无法完成端到端链路验证（限制已在 5.11 解除，P0-2 ~ P0-5 均已补做容器内实测）。
- 备注：验证在 `frontend-new/node_modules` 留下未跟踪依赖目录（`.gitignore` 未覆盖），`bin/` 与 `dist/` 的构建改动已还原。

> 说明：以上为 P0 实施前的基线。各项实施记录见 5.4–5.8。

### 5.4 P0-1 实施记录（2026-09-27）

已完成的改动：

| 文件 | 改动 |
| --- | --- |
| `.env` | 移出版本跟踪（文件保留在本地磁盘），并在 `.gitignore` 中忽略 |
| `.env.example` | 新增占位符模板，供 `docker-compose` 变量注入使用 |
| `config.yaml`、`config-scheduler.yaml` | 删除明文数据库密码，改由 `DB_PASSWORD` 注入 |
| `internal/config/config.go` | 新增 `server.jwt_secret`（绑定 `JWT_SECRET`）；数据库密码默认值改为空 |
| `pkg/auth/auth.go` | 删除内置默认密钥；新增 `InitJWT`，缺失或短于 32 字符即报错；`ParseJWT` 固定校验 HS256 |
| `cmd/api-server/main.go` | 启动时调用 `auth.InitJWT`，配置非法立即退出 |
| `docker-compose.yml` | 所有凭据改为 `${VAR:?...}` 必填注入 |
| `.github/workflows/test.yml` | 改用 `secrets.CI_MYSQL_ROOT_PASSWORD`、`secrets.CI_JWT_SECRET`，未配置时回退到 CI 专用占位值 |
| `bin/` | 移出版本跟踪（旧二进制内嵌了默认密钥字符串） |
| `README.md`、`STARTUP_GUIDE.md`、`测试指南.md`、`docs/**` | 明文密码替换为占位符或环境变量 |

验证结果：

- `go build ./...`、`go vet ./...` 通过（Go 1.21.13）。
- 未设置 `JWT_SECRET`：启动立即失败，退出码 1，提示 `JWT_SECRET is required but not set`。
- `JWT_SECRET` 过短：启动立即失败，提示至少 32 字符。
- 合法 `JWT_SECRET`：通过校验并继续进入数据库初始化。
- `git grep` 校验：tracked 文件中已无 `Cz05180921`、`video_password` 及旧 JWT 默认值。
- 4 个 YAML 文件语法校验通过。

遗留（需人工完成）：

- **必须轮换密钥**：旧口令仍在 git 历史中，需更换 MySQL 密码与 JWT 密钥；代码改动无法清除历史。
- 在仓库 Settings → Secrets 配置 `CI_MYSQL_ROOT_PASSWORD`、`CI_JWT_SECRET`（可选，未配置时 CI 使用占位值）。
- `docker compose config` 未执行（环境无 Docker），compose 变量注入仅通过 YAML 语法校验。（已在 5.11 补做：`docker compose config -q` 通过，9 个服务全部解析成功。）

### 5.5 P0-2 实施记录（2026-09-27）

已完成的改动：

| 文件 | 改动 |
| --- | --- |
| `internal/services/user_service.go` | 新增 `userCacheKey` / `invalidateUserCache`；`UpdateUser`、`DeleteUser` 写后清除 `user:{id}` |
| `internal/services/video_service.go` | 新增 `videoCacheKey` / `invalidateVideoCache`；`UpdateVideo`、`DeleteVideo`、`HardDeleteVideo` 写后清除 `video:{uuid}` |
| `pkg/redis/redis.go` | `CacheManager.Delete` 增加空键保护与失败告警日志 |

验证结果：

- `go build ./...`、`go vet ./...` 通过。
- 缓存键由同一 helper 生成，读写两侧结构上不再可能不一致。
- 使用本地 Redis 实测：`Set user:1 → Get 命中 → Delete → Get 返回零值`，且 `Delete()` 空参数返回 nil。
- 5 个写路径全部接入失效逻辑（2 处用户 + 3 处视频）。

遗留：

- 服务级回归测试（写入后立即读到新值）需要 MySQL，当前环境不可用；将在 P2-4 建立测试基础设施后补充。

### 5.6 P0-5 实施记录（2026-09-27）

- `internal/handlers/video_handler.go`：移除每个上传请求新建的 goroutine，改为同步入队；`taskQueue == nil`（Redis 不可用）时降级跳过并记录告警，响应中的 `transcode_status` 返回 `queued` 或 `skipped`。
- 验证：`go build ./...`、`go vet ./...` 通过；nil 分支已由显式判空覆盖，不再存在解引用。
- 遗留：上传路径的端到端验证需要 MySQL，当前环境不可用。

### 5.7 P0-6 实施记录（2026-09-27）

- `internal/middleware/auth.go`：`CORSMiddleware` 改为接收来源白名单，仅对白名单来源回显 `Access-Control-Allow-Origin` 并设置 `Allow-Credentials`；非白名单来源不返回 CORS 头，预检返回 403；`*` 单独处理且不携带凭据。
- `internal/config/config.go`、`config.yaml`：新增 `server.cors.allowed_origins`，默认 `http://localhost:3000`，可用 `CORS_ALLOWED_ORIGINS`（逗号分隔）覆盖。
- `cmd/api-server/main.go`、`docker-compose.yml`：传入白名单并透传环境变量。
- 验证：临时测试覆盖 5 个场景（允许来源 GET、拒绝来源 GET、拒绝来源预检 403、允许来源预检 204、无 Origin 请求）全部通过；临时测试已删除。

### 5.8 P0-3 实施记录（2026-09-27）

- 四个 Dockerfile：最终阶段 `WORKDIR /root/` 改为 `/app`，使默认 `./storage/...` 落在挂载点 `/app/storage`。
- `internal/config/config.go`、`cmd/scheduler/main.go`：新增 `STORAGE_VIDEO_DIR`、`STORAGE_TEMP_DIR` 环境变量绑定。
- `docker-compose.yml`：api-server、scheduler、transcode-worker 统一显式存储路径；**为 scheduler 补上缺失的 `./storage:/app/storage` 挂载**（此前清理任务实际删不到共享文件）。
- 验证：`go build ./...`、`go vet ./...` 通过；临时测试确认 `STORAGE_*` 能覆盖默认值（已删除）。
- 遗留：`docker compose up` 的上传 → 转码 → 播放全链路无法验证（环境无 Docker）。（已在 5.11 完成实测：转码产物 `<uuid>_high.mp4` 由 transcode-worker 写入共享卷，宿主机 `storage/videos/` 可见。）

### 5.9 P0-4 实施记录（2026-09-27）

- `pkg/validation/validation.go`：`ValidateVideoFile` 增加体积上限参数与文件头（magic number）校验，覆盖 MP4/MOV/3GP、MKV/WebM、AVI、FLV、WMV/ASF；新增 `ErrFileTooLarge` 哨兵错误与兜底上限 `MaxVideoSize`（50 MB）。
- `internal/handlers/video_handler.go`：用 `http.MaxBytesReader` 限制请求体（配置上限 +1 MB 余量），超限返回 413；体积超限返回 413，类型/内容非法返回 400。
- `cmd/api-server/main.go`：设置 `MaxMultipartMemory = 8 MB`，超出部分落盘而非常驻内存。
- 验证：临时测试覆盖 6 个场景（合法 MP4、合法 MKV、错误扩展名、伪造内容、超限、截断文件）全部通过；临时测试已删除。
- 遗留：CI 上传的是 `test_video.txt`，现有校验会拒绝；需在 P2-4 改为生成合法 MP4 并断言响应状态码。

### 5.10 P0-7 实施记录（2026-09-27）

- `pkg/database/database.go`：`InitDatabase` 改为返回 `(*gorm.DB, error)`；连接失败按 `DB_CONNECT_ATTEMPTS`（默认 5）重试、间隔 `DB_CONNECT_DELAY`（默认 3s），并执行 `Ping` 探活；移除三处 `panic`。
- `cmd/api-server/main.go`、`cmd/scheduler/main.go`：改为结构化日志 + `log.Fatal`，以退出码 1 结束。
- 验证：MySQL 不可用时设置 `DB_CONNECT_ATTEMPTS=2 DB_CONNECT_DELAY=500ms`，实测重试 2 次、耗时 519 ms、退出码 1、输出无 panic（`grep -c panic` = 0）。

### 5.11 Docker 部署与端到端验证记录（2026-09-27）

本轮目标：按仓库既有的 Docker 方案把整套环境跑起来，并验证 P0 改动在真实容器链路中生效。

**环境与准备**

- WSL2 Debian + Docker Desktop（WSL 集成）；`docker compose` 9 个服务全部启动。
- Docker Hub 直连不可用：在 Docker Desktop `daemon.json` 配置国内镜像源后，`mysql:8.0`、`redis:alpine`、`golang:1.21-alpine`、`node:20-alpine`、`nginx:alpine` 均可正常拉取。
- 本地 `.env` 由 `.env.example` 生成（随机口令、权限 600，已被 `.gitignore` 忽略），`docker compose config -q` 通过。

**构建加速（本轮一并落地）**

| 改动 | 效果 |
| --- | --- |
| 新增 `.dockerignore`、`frontend-new/.dockerignore` | 构建上下文由 273 MB 降至约 1 MB，`node_modules` 不再进入镜像层 |
| `Dockerfile.*` 增加 apk 国内镜像 | `apk add ffmpeg` 由 **>25 分钟未完成**降至 **36 秒** |
| `Dockerfile.transcode-worker` builder 阶段移除 ffmpeg | 该依赖仅运行期需要，构建阶段减负 |
| `frontend-new/Dockerfile.frontend.nginx` 改用 `registry.npmmirror.com` | 前端依赖安装加速，镜像构建成功 |
| `docker-compose.yml` 端口可配置 | `FRONTEND_PORT` / `GRAFANA_PORT` 可覆盖；本机 3000/3001 落在 Windows 保留端口段（2952-3051），改用 8000/3301 |

**本轮修复的缺陷**：P1-8（`init.sql` 只保留 `users` 引导表，其余交逐表 `AutoMigrate`，单表失败不再中断）、P1-9（迁移前用 MySQL `GET_LOCK` 命名锁串行化）、P1-10（`NOGROUP` 视为 pending=0）、P2-9（健康检查改用 `127.0.0.1`）。

**验收结果**（全新数据卷 `docker compose down -v && up -d` 后执行）

| 场景 | 结果 |
| --- | --- |
| `docker compose ps` | ✅ 9 个容器全部 Up，frontend `healthy` |
| 三个服务并发启动的迁移日志 | ✅ 无 `already exists` / `Duplicate key name` / `Error 1146` |
| `GET /health` | ✅ 200 |
| 注册 / 登录 | ✅ 201，返回可用 JWT |
| 无 token / 有效 token 访问 `/users/profile` | ✅ 401 / 200 |
| 伪造内容上传 | ✅ 400（magic number 校验，P0-4） |
| 55 MB 上传 | ✅ 413（`MaxBytesReader`，P0-4） |
| 合法 MP4 上传 | ✅ 201，`transcode_status: queued` |
| 转码任务 | ✅ `status: completed`，`progress: 100` |
| 转码产物落盘 | ✅ 宿主机 `storage/videos/<uuid>_high.mp4`（774 KB，P0-3） |
| `GET /transcode/queue/stats` | ✅ 200（修复前 500，P1-10） |
| 删除视频后又立即读取 | ✅ 404（缓存失效，P0-2） |
| `video_deletion_records` 落库 | ✅ 1 行（修复前报 1146，P1-8） |
| CORS 白名单 | ✅ 回显允许来源并带凭据；非白名单无 CORS 头，预检 403（P0-6） |
| 前端 `:8000` 首页 / `/api` 代理 | ✅ 200 / 401（代理生效） |
| `go build ./...`、`go vet ./...`、`go test ./...` | ✅ 全部通过（Go 1.21.13） |
| `cd frontend-new && npx vitest --run` | ✅ 3 文件 / 20 用例通过 |

**遗留**

- `go.mod` 中 `go-redis/redis/v8` 由 indirect 提升为直接依赖（`pkg/queue` 直接 import），属正确的 tidy 结果。
- 需要人工执行：轮换历史泄露的旧口令与 JWT 密钥（见 P0-1）。
- 版本控制产物清理已在 5.12 处理完毕。

### 5.12 版本控制跟踪与忽略规则修正（2026-09-27）

处理 P2-8：把构建产物与运行时产物移出版本控制，并让忽略规则覆盖所有层级。

**审计结果**：全仓 172 个被跟踪文件中，13 个属产物类——`frontend-new/dist/**`（12 个，且文件名哈希 `index-CQRXtVaz.js` 已落后于当前构建产物）、`storage/videos/b0becc9e-...`（34 字节测试残留）。`node_modules`、`.env`、`bin/` 均未被跟踪（后两者已随 P0-1 出库）。

**改动**

| 项 | 内容 |
| --- | --- |
| `.gitignore` | 三处硬编码依赖路径合并为通用 `node_modules/`；新增 `frontend-new/dist/`、`storage/videos/*`、`storage/temp/*` |
| `git rm -r --cached frontend-new/dist` | 12 个构建产物退出索引，磁盘文件保留 |
| `git rm --cached storage/videos/b0becc9e-...` | 测试残留退出索引，磁盘文件保留 |

**核验**

- 被跟踪文件数 172 → 159。
- `git ls-files | grep -cE 'node_modules|/dist/|^bin/|\.env$'` = 0。
- `git check-ignore -v frontend-new/node_modules` 命中 `.gitignore` 的 `node_modules/` 规则。
- `git status` 中不再出现产物类未跟踪条目；`frontend-new/dist/`、`storage/` 的磁盘文件未被删除，`docker compose` 前端镜像仍在容器内自行构建。

**遗留**：删除记录已在索引中暂存，需随下次提交落库；P2-2 的“三套前端并存”仍未处理。

### 5.13 P1-1 前后端契约对齐（2026-09-27）

**改动（后端）**

| 文件 | 改动 |
| --- | --- |
| `internal/services/video_service.go` | 新增 `GetVideos`（分页 + 名称模糊搜索）、`GetRelatedVideos`（同作者其他视频）、`GetAuthorNames`（批量取作者名，避免新接口引入 N+1） |
| `internal/handlers/video_handler.go` | 新增 `ListVideos`、`GetRelatedVideos`；抽出 `parsePagination` / `videoListItem` / `buildVideoList`，`GetUserVideos` 改为复用，列表类接口字段完全一致 |
| `cmd/api-server/main.go` | 注册 `GET /api/v1/videos`（公开组：首页与列表页对未登录用户可见）与 `GET /api/v1/videos/:video_id/related`（受保护组，与详情接口一致） |

**改动（前端）**

| 文件 | 改动 |
| --- | --- |
| `src/api/video.js` | `getVideos` → `GET /videos`（`limit` 替代 `page_size`，支持 `q`）；新增 `searchVideos`；`uploadVideo` → `POST /videos/upload`；`getMyVideos` → `GET /users/videos`；删除无后端支撑的 `updateVideo` |
| `src/api/user.js` | `register(username, password)` 去掉后端不存在的邮箱参数；删除无后端支撑的 `changePassword` |
| `src/stores/video.js` | 新增 `searchVideos` 动作；删除后按 `video_id` 过滤（原按 `id` 过滤无效） |
| `views/*.vue` | 列表/详情/个人中心改用后端字段；`Videos.vue` 搜索框接上 `q`（此前点击无效）；封面 `<img>` 改为占位块（后端无封面字段，原地址 `/static/placeholder.jpg` 也不存在）；`VideoDetail.vue` 相关推荐与播放地址改走 `videoApi`、`isOwner` 改用 `author_id`；`Upload.vue` 改 `file`/`name` 字段并按 50MB 校验；`Register.vue`/`Profile.vue` 去掉邮箱与简介输入 |
| `tests/unit/videoStore.test.js` | 用例字段对齐真实契约，新增搜索用例（21 通过） |
| `tests/e2e/auth.spec.js` | 去掉已移除的 `#email` 断言 |

**验收（容器内实测）**

| 场景 | 结果 |
| --- | --- |
| 未登录 `GET /api/v1/videos` | ✅ 200（公开可读） |
| 上传（字段 `file` + `name`） | ✅ 201，`name` 正确落库 |
| 列表项字段 | ✅ `video_id`/`name`/`author_name`/`display_time`/`created_at` 齐全 |
| `?q=` 搜索 | ✅ 命中 1 条；不匹配 `total: 0` |
| `GET /users/videos` | ✅ 200，2 条，字段与列表一致 |
| `GET /videos/:id/related` | ✅ 双向返回同作者其他视频，不含自身 |
| `PUT /users/profile` 改名 | ✅ 200，`profile.username` 更新，新用户名可登录 |
| 前端构建产物 | ✅ 含 `videos/upload`、`users/videos`、`/videos/${id}/related` 路径 |
| `npx vitest --run` | ✅ 3 文件 / 21 用例通过 |

**遗留**

- 点赞、播放量、封面图、时长、简介、邮箱等字段后端数据模型不存在，前端已停止引用，需先扩展 `internal/models` 与数据库（记录为 P1-11）。
- Playwright e2e 依赖 `:3000` 开发端口与浏览器二进制，本机端口 3000 落在 Windows 保留段且未安装浏览器，本轮未执行。
- `VideoDetail.vue` 的点赞按钮仍是本地状态（代码内 TODO），随 P1-11 一并实现。

### 5.14 P1-2 流媒体鉴权方案（2026-09-27）

**问题**：`GET /videos/:id/stream` 位于受保护路由组，需要 `Authorization: Bearer`，而 `<video src>` / Video.js 的 `<source>` 无法设置请求头，播放必然 401。直接把主访问令牌放进 URL 又不可接受——它是 24 小时有效，会落在访问日志、浏览器历史与 `Referer` 中。

**方案**：新增独立的「播放令牌」——短时效（默认 2 小时，`STREAM_TOKEN_TTL` 可配）、绑定单个视频、独立 JWT Subject，只能用于播放。

| 层 | 改动 |
| --- | --- |
| `pkg/auth/auth.go` | 新增 `SubjectAccessToken` / `SubjectStreamToken`、`GenerateStreamToken` / `ParseStreamToken`；解析统一走 `parse(token, expectedSubject)`，**访问令牌解析现在会拒绝播放令牌**（否则 2 小时内的播放令牌可当完整访问令牌使用） |
| `internal/middleware/auth.go` | 新增 `StreamAuthMiddleware`：仅用于播放路由，接受 Bearer 头或 `?token=` 播放令牌 |
| `internal/handlers/video_handler.go` | 新增 `CreateStreamToken`（`GET /videos/:video_id/stream-token`），返回 `{token, expires_in}` |
| `cmd/api-server/main.go` | 新增 `stream` 路由组并挂 `StreamAuthMiddleware`；`stream-token` 放在受保护组 |
| `internal/config/config.go`、`config.yaml` | 新增 `server.stream_token_ttl`（默认 7200 秒），支持 `STREAM_TOKEN_TTL` 环境变量 |
| `frontend-new/src/api/video.js` | 新增 `getStreamToken(id)`；`getVideoStream(id, token, quality)` 拼接 `?token=` |
| `VideoDetail.vue`、`VideoPlayer.vue` | 播放前先换取令牌；失败时给出提示而不是静默黑屏 |

**验收（容器内实测）**

| 场景 | 结果 |
| --- | --- |
| 无凭据换取播放令牌 | ✅ 401 |
| 带访问令牌换取 | ✅ 200，载荷 `sub=stream-token`，含 `video_id` 与 `exp` |
| `?token=` 播放目标视频 | ✅ 200（11507 字节，与源文件一致） |
| 用同一令牌播放**其他**视频 | ✅ 401（令牌绑定单个视频） |
| 无凭据 / 伪造令牌播放 | ✅ 401 / 401 |
| Bearer 头播放（兼容既有脚本与调用方） | ✅ 200 |
| 用播放令牌调用 `/users/profile` | ✅ 401（不能顶替访问令牌） |
| 访问令牌调用 `/users/profile` | ✅ 200 |
| 前端构建产物 | ✅ 含 `stream-token` 与 `/stream?` |

**遗留**

- 播放令牌 2 小时有效：超时后拖动进度条会 401，需重新打开页面；后续可做自动续期或按需刷新。
- 浏览器内真实播放未验证（本机未安装 Playwright 浏览器），本轮以「模拟前端两步调用」的方式覆盖。

### 5.15 P1-3 / P1-4 实施记录（2026-09-27）

**P1-3 评论分页 total 错误**：`GetComments` 原先用 `len(commentList)` 作为 `total`，无论第几页都只等于当前页条数，前端永远算不出总页数。

- `internal/services/comment_service.go`：`GetCommentsByVideoID` 改为 `GetCommentsByVideoIDWithTotal(videoID, limit, offset) ([]*models.Comment, int64, error)`，先 `Count` 取真实总数再分页查询。
- `internal/handlers/comment_handler.go`：改用新方法返回真实 `total`；分页参数改用全站统一的 `parsePagination(c)`，因此评论默认 `limit` 从 10 调整为 12，并顺带获得 `limit` 上下界保护（1–100）。

**P1-4 消除 N+1**：评论列表原先对每条评论单独查一次作者名，返回 n 条评论即 n 次 `users` 查询。

- `internal/handlers/comment_handler.go`：先收集本页 `author_id`，再调用 `h.videoService.GetAuthorNames(authorIDs)` 一次 `WHERE id IN (...)` 批量取回并组装。
- 复核结论：`video_handler.go` 的 `buildVideoList`（列表 / 搜索 / 我的视频 / 相关视频）已使用批量 `GetAuthorNames`，本轮改动后评论链路是最后一处 N+1；`transcode_handler.go ListTasks` 为简化实现（未分页），不构成 N+1。

**验收（容器内实测，`docker compose build api-server && up -d` 后）**

| 场景 | 结果 |
| --- | --- |
| 5 条评论 + `?page=1&limit=2` | ✅ `total=5`、返回 2 条（旧实现 `total=2`） |
| 第 2 页 `?page=2&limit=2` | ✅ 返回 2 条，`total=5` |
| 越界页 `?page=99&limit=2` | ✅ 返回 0 条，`total=5` |
| `author_name` 字段 | ✅ 正确（`p13_30173`） |
| 6 条评论单次请求的 `users` 查询次数 | ✅ 1 次（旧实现为 6 次），`comments` 查询 2 次（count + 分页） |

**遗留**：评论 `created_at` 与模型 `Ctime` 并存（`Ctime` 为字符串格式时间戳），后续统一时间字段时一并清理。

### 5.16 P1-5 统一错误码与响应（2026-09-27）

**问题**：错误响应由各 handler 手写 `c.JSON(http.StatusX, gin.H{"error": "..."})`，共 90 处，存在三类不一致：

1. **状态码错**：`if err != nil` 一律 500，导致「资源不存在」也返回 500（如 `UpdateProfile`、`UpdateComment` 对已删除记录）；反过来，部分分支把所有错误（含数据库故障）都压成 404。
2. **响应体不统一**：`pkg/errors`（`AppError`）与 `internal/utils/response.go`（`Response`）定义后从未被引用。
3. **消息全靠字符串约定**，无法按错误类型分支处理。

**改动**

| 层 | 改动 |
| --- | --- |
| `pkg/errors` | 新增 `NotFoundf` / `BadRequestf` / `Unauthorizedf` / `Forbiddenf` / `Conflictf` / `Internalf` 构造器与 `StatusCode` / `Message` 提取函数；`AppError.Is` 改为按状态码判定，`fmt.Errorf("%w")` 包装后仍可识别 |
| `internal/utils/response.go` | 重写为 Gin 版统一响应：`Fail(c, status, msg)` 与 `FailFrom(c, err, fallback)`，统一输出 `{"success":false,"code":<status>,"error":"..."}`；删除从未使用的 `http.ResponseWriter` 版本 |
| `internal/services/*` | 领域错误改为返回 `AppError`：视频 / 评论 / 用户 `not found` → 404，登录密码错误 → 401；数据库等内部错误保持普通 error，由 `FailFrom` 兜底为 500 且只暴露通用消息 |
| `pkg/queue/task_queue.go` | `GetTaskStatus` 的缺失任务返回 `NotFoundf("task not found")`，Redis 故障仍为 500 |
| handlers + `middleware/auth.go` | 90 处错误分支全部改走 `utils.Fail` / `FailFrom`；中间件不再需要额外的 `c.Abort()`（`AbortWithStatusJSON` 已终止） |

**响应体兼容性**：保留 `error` 字段（前端仅读取该字段），新增 `success` / `code` 便于调用方按状态码分支。

**验收（容器内实测，19/19 通过）**

| 场景 | 结果 |
| --- | --- |
| 无 token / 非法格式 / 伪造 token 访问 `/users/profile` | ✅ 401，统一响应体 |
| 注册密码过短 / 重复用户名 / 登录密码错误 | ✅ 400 / 409 / 401 |
| `GET /videos/:id`、`/related`、`/stream-token`、`DELETE /videos/:id` 视频不存在 | ✅ 404 `video not found` |
| 评论读 / 改 / 删（`/comments/:id`）不存在 | ✅ 404 `comment not found` |
| 转码任务不存在 | ✅ 404 `task not found` |
| 关联账户已删除的令牌访问 / 修改 profile | ✅ 404 `user not found`（旧实现为 500） |
| 上传缺少文件 | ✅ 400 |
| `GET /videos`、`/health` | ✅ 200（无回归） |

### 5.17 P1-6 转码闭环（2026-09-27）

**问题**（四个独立缺陷叠加，导致转码链路整条不可用）

1. **产物不落库**：`executeTranscode` 写出 `<video_id>_<quality>.mp4` 后不更新 `videos.file_path`，而播放侧固定按 `storage/videos/<uuid>`（原始上传文件）取文件 —— 转码结果从未被使用，白跑一遍 CPU。
2. **缺 ffmpeg 假成功**：`exec.LookPath("ffmpeg")` 失败时走 `simulateTranscode`，把任务标记为 completed 但没有任何产物。
3. **失败不重试**：任务失败直接置 failed，只能靠人工调 retry 接口；`MaxRetries` 定义了却从未使用。
4. **pending 不回收**：每条消息的消费者名是随机 UUID，worker 崩溃后消息永久滞留在 PEL，既不被重投也不计为失败。

**改动**

| 层 | 改动 |
| --- | --- |
| `pkg/queue/task_queue.go` | `TranscodeTask` 新增 `retries`；`TaskQueue` 固定消费者名 `worker-<pid>`（原来每次读取都用随机名，pending 会散落到无数消费者名下，无法回收）；新增 `ReclaimStaleTasks`；`EnqueueTranscodeTask` 不再覆盖调用方设置的重试次数与创建时间 |
| `internal/services/transcode_service.go` | `executeTranscode` 返回产物路径，删除 `simulateTranscode`，ffmpeg 缺失即失败；成功后 `persistResult` 回写 `file_path`/`size` 并失效 Redis 缓存；失败按 `TRANSCODE_MAX_RETRIES` 重新入队，超限才置 failed；`StartWorker` 增加 `reclaimLoop` 周期回收 |
| `cmd/transcode-worker/main.go` | 初始化数据库（回写 `file_path` 的前提，此前 worker 只连 Redis） |
| `internal/handlers/video_handler.go` | 新增 `resolveVideoPath`：优先使用 `video.file_path`，产物缺失或字段为空时回退到原始上传文件 |
| `internal/handlers/transcode_handler.go` | 任务详情返回 `retries`；人工重试时重置重试计数与时间戳 |
| `docker-compose.yml` | 新增 `TRANSCODE_MAX_RETRIES` / `TRANSCODE_VISIBILITY_TIMEOUT` / `TRANSCODE_RECLAIM_INTERVAL` 可配置项 |

**踩坑记录**：go-redis v8.11.5 的 `XAutoClaim` 解析器硬编码要求 2 元素回复，而 Redis 7 的 `XAUTOCLAIM` 返回 3 元素（多出「已删除消息 ID」列表），报错 `got 3, wanted 2`。更隐蔽的是**命令已在服务端执行成功**（消息已被转移给新消费者），只有客户端解析失败，表现为「回收一直失败但 pending 归属已变化」。最终改用 `XPENDING`（找空闲超时消息）+ `XCLAIM`（接管）实现。

**验收（容器内实测）**

| 场景 | 结果 |
| --- | --- |
| 上传 → 转码 | ✅ 约 2 秒生成 `storage/videos/<uuid>_high.mp4`（源 15051 B → 产物 757373 B） |
| 播放路径 | ✅ `/videos/:id/stream` 返回 757373 字节，与产物一致且不等于源文件（证明 `file_path` 已回写并生效） |
| 消息确认 | ✅ `XPENDING` = 0 |
| 失败重试 → 永久失败 | ✅ 源文件不存在时重试 3 次后置 `failed`，`error` 保留原因（4 次尝试 = 首次 + 3 次重试） |
| pending 回收 | ✅ 用 `dead-worker` 消费后不确认并重启 worker（可见性超时设为 5s），消息被回收执行，`XPENDING` 归零 |
| 失败任务的 API 详情 | ✅ 返回 `status`/`retries`/`error`/时间戳 |

**遗留**：`videos.status` 未区分「转码中 / 可播放」；多码率 HLS 与封面截图仍属阶段四演进项。

### 5.18 全量回归快照（2026-09-27）

全部镜像 `docker compose build && up -d` 重建后（9 个容器全部 Up，`frontend` healthy），一次性执行：

| 验证 | 结果 |
| --- | --- |
| P1-5 错误契约脚本（19 项） | ✅ 19 PASS / 0 FAIL |
| P1-3 / P1-4 评论分页（total=5、分页、越界页） | ✅ 通过 |
| P1-6 上传 → 转码 → 播放（产物 749435 B 与 stream 字节数一致） | ✅ 通过，`XPENDING` = 0 |
| `:8080/health` / `:8000/` / `:9090/-/healthy` | ✅ 200 / 200 / 200 |

### 5.19 P1-7 实施记录（2026-09-27）

**改动**

| 文件 | 改动 |
| --- | --- |
| `internal/models/models.go` | `Session.UserID` 增加索引；`TTL` 重命名为 `ExpiresAt`（列名仍为 `ttl`，不产生重命名 DDL），语义明确为 Unix 秒 |
| `pkg/auth/auth.go` | 导出 `AccessTokenTTL`（24h），令牌过期时间与会话过期时间共用同一常量 |
| `internal/services/session_service.go`（新增） | 会话落地 / 校验 / 吊销 / 过期清理；登出幂等，会话不存在或已过期一律 401 |
| `internal/middleware/auth.go` | 新增 `SessionValidator`；`AuthMiddleware` 与 `StreamAuthMiddleware` 的 Bearer 分支在签名校验后追加会话校验（会话错误 401、DB 故障 500） |
| `internal/handlers/user_handler.go` | 注册 / 登录后落库会话；新增 `Logout`、`LogoutAll`；改密后吊销其他设备会话 |
| `internal/services/user_service.go` | `DeleteUser` 改为事务内级联：会话物理删除、评论软删除、视频标记 `deleted` 并软删除、写入删除台账、最后软删除用户 |
| `cmd/api-server/main.go` | 注册 `POST /users/logout`、`POST /users/logout-all`；装配会话中间件；每小时清理过期会话 |
| `frontend-new/src/api/user.js`、`stores/user.js`、`App.vue` | 登出先本地清理再调用后端吊销接口（显式携带登出前的令牌，避免清理后请求缺头） |
| `test-api.sh` | 新增第 13 项用例「Logout Revokes Session」 |

**验收结果**（重建 api-server 镜像后执行）

| 场景 | 结果 |
| --- | --- |
| `test-api.sh` | ✅ 13/13（含新增登出用例） |
| 注册 / 登录后 `sessions` 落库 | ✅ 每次登录 1 行 |
| 登出后原令牌访问 `/users/profile` | ✅ 401，其余会话仍 200 |
| 改密后其他设备会话 | ✅ 401，当前会话保留 200 |
| 注销账户级联清理 | ✅ 会话清空、视频 `deleted` 且软删除、评论下架、`video_deletion_records` 写入 `Account deleted` |
| `go build ./...` / `go vet ./...` | ✅ 通过 |
| `cd frontend-new && npx vitest run` | ✅ 3 文件 / 22 用例（新增登出吊销断言） |

**遗留**：访问令牌仍为固定 24h 有效期，未引入刷新令牌；注销账户使用软删除，用户名受 `login_name` 唯一索引限制不可立即复用。

### 5.20 P1-11a 用户字段（email / bio）实施记录（2026-09-27）

P1-11 涉及用户字段、视频字段与点赞三块，按「一次一个可验收单元」拆成三批推进，本节只覆盖用户字段。

**改动**

| 文件 | 改动 |
| --- | --- |
| `internal/models/models.go` | `User` 新增 `Email`（varchar(255)）与 `Bio`（varchar(500)），`default:''` 保证既有行迁移后为空串 |
| `pkg/validation/validation.go` | 新增 `ValidateEmail`（空串视为未填写）、`ValidateBio`（≤200 字） |
| `internal/services/user_service.go` | `CreateUser` 接收 `email`；新增 `IsEmailTaken`（支持排除自己，供资料更新使用） |
| `internal/handlers/user_handler.go` | 注册接受可选 `email` 并做格式与占用校验；`GetProfile` 返回 `email`/`bio`；`UpdateProfile` 支持 `email`/`bio`（指针区分「未提交」与「清空」） |
| `frontend-new/src/api/user.js` | `register(username, password, email)`，邮箱为空时不提交该键 |
| `frontend-new/src/views/Register.vue` | 新增可选邮箱输入与前端格式校验 |
| `frontend-new/src/views/Profile.vue` | 恢复邮箱 / 简介输入与展示，资料卡显示邮箱与简介 |
| `frontend-new/src/tests/e2e/auth.spec.js` | 注册表单用例恢复 `#email` 可见性断言 |

**验收结果**

| 场景 | 结果 |
| --- | --- |
| 注册携带邮箱 → `/users/profile` 与 DB | ✅ 一致回显 |
| 非法邮箱 / 重复邮箱 | ✅ 400 / 409 |
| 不提交或提交空邮箱 | ✅ 201，资料中为空串 |
| 更新 email + bio | ✅ 200，DB 落库 |
| 更新为他人邮箱 / 非法邮箱 / 超长 bio | ✅ 409 / 400 / 400 |
| 清空 bio（`{"bio":""}`） | ✅ 200，回显为空 |
| 专项验收脚本（17 项） | ✅ 17 PASS / 0 FAIL |
| `test-api.sh` 回归 | ✅ 13/13 |
| `go build ./...` / `go vet ./...` | ✅ 通过 |
| `cd frontend-new && npx vitest run` | ✅ 3 文件 / 22 用例 |

**遗留**：`email` 只有普通索引而非唯一索引（历史行均为空串，唯一索引会互相冲突），唯一性由 service 层在写入前校验，并发下存在极小概率竞态。

### 5.21 P1-11b 视频字段（简介 / 封面 / 播放量 / 时长）实施记录（2026-09-27）

**改动**

| 文件 | 改动 |
| --- | --- |
| `internal/models/models.go` | `Video` 新增 `Description`（varchar(500)）、`CoverPath`、`ViewCount`（`Duration` 字段早已存在，此前从未写入） |
| `pkg/validation/validation.go` | 新增 `ValidateVideoDescription`（≤500 字） |
| `internal/services/video_service.go` | 新增 `IncrementViewCount`：SQL 自增 + 失效缓存，避免读改写丢更新 |
| `internal/services/transcode_service.go` | 转码成功后用 ffprobe 取时长、ffmpeg 抽帧生成封面（`<产物名>_cover.jpg`）；两项失败只告警，不影响任务成功状态 |
| `internal/handlers/video_handler.go` | 上传接受 `description`；列表项与详情统一返回 `description`/`cover_url`/`duration`/`view_count`；打开详情页计一次观看；新增 `GetVideoCover` |
| `cmd/api-server/main.go` | 新增公开路由 `GET /videos/:video_id/cover`（`<img>` 无法携带 Authorization 头） |
| `frontend-new/src/views/Videos.vue`、`VideoDetail.vue`、`Profile.vue` | 封面图 + 时长角标 + 播放量，详情页展示简介 |
| `frontend-new/src/views/Upload.vue` | 新增可选简介输入并随表单提交 |

**验收结果**（重建 api-server 与 transcode-worker 后执行）

| 场景 | 结果 |
| --- | --- |
| 上传携带简介 | ✅ 201 且落库；>500 字返回 400 |
| 转码后 `duration` / `cover_path` | ✅ 均写入，封面文件落在共享卷 |
| 详情 / 列表字段 | ✅ `description`、`cover_url`、`duration`、`view_count` 齐全 |
| `GET /videos/:id/cover` | ✅ 200 且 `image/jpeg`；无封面或视频不存在返回 404 |
| 连续访问详情页 | ✅ `view_count` 递增且与 DB 一致 |
| 专项验收脚本（17 项） | ✅ 17 PASS / 0 FAIL |
| `test-api.sh` 回归 | ✅ 13/13 |
| `go build ./...` / `go vet ./...` | ✅ 通过 |
| `cd frontend-new && npx vitest run` | ✅ 3 文件 / 22 用例 |

**遗留**：封面为单张 JPEG，未生成多分辨率缩略图，且封面接口对外公开；`view_count` 在打开详情页时累加（需登录），未按用户去重。

### 5.22 P1-11c 点赞实施记录（2026-09-27）

**改动**

| 文件 | 改动 |
| --- | --- |
| `internal/models/models.go` | 新增 `VideoLike`（`(user_id, video_uuid)` 组合唯一索引，重复点赞天然幂等）；`Video` 新增冗余计数 `LikeCount` |
| `pkg/database/database.go` | 迁移表新增 `video_likes` |
| `internal/services/like_service.go`（新增） | `Like` / `Unlike` / `IsLiked` / `LikeCount`：关系表与计数在同一事务内更新，先校验视频存在，全部幂等；写后失效视频缓存 |
| `internal/handlers/video_handler.go` | 详情返回 `like_count`/`liked`，列表项返回 `like_count`；新增 `LikeVideo` / `UnlikeVideo` |
| `cmd/api-server/main.go` | 新增 `POST /videos/:video_id/like` 与 `DELETE /videos/:video_id/like` |
| `internal/services/user_service.go` | 注销级联追加：删除该用户的点赞关系并修正计数，同时清理其视频上他人的点赞 |
| `frontend-new/src/api/video.js`、`views/VideoDetail.vue` | `toggleLike` 改调真实接口，计数以服务端返回为准（原为本地自增的 TODO） |

**验收结果**（重建 api-server 后执行）

| 场景 | 结果 |
| --- | --- |
| 点赞 / 取消点赞 | ✅ 200，`{liked, like_count}` |
| 重复点赞、重复取消 | ✅ 幂等，计数不漂移 |
| 详情 `liked` / `like_count`、列表 `like_count` | ✅ 与 DB 一致 |
| 未登录点赞 / 视频不存在 | ✅ 401 / 404 |
| 两人点赞 | ✅ `like_count = 2` |
| 注销作者 / 注销点赞者 | ✅ 关系与计数均正确清理 |
| 专项验收脚本（22 项） | ✅ 22 PASS / 0 FAIL |
| P1-7 级联验收回归 | ✅ 22 PASS / 0 FAIL |
| `test-api.sh` 回归 | ✅ 13/13 |
| `go build ./...` / `go vet ./...` | ✅ 通过 |
| `cd frontend-new && npx vitest run` | ✅ 3 文件 / 22 用例 |

**遗留**：点赞接口未做频率限制与防刷（仅依赖全局 IP 限流）；历史视频的 `like_count` 无对账任务，若有漂移只能靠 `video_likes` 重算。

### 5.23 P2-10 全仓 gofmt 收敛（2026-09-27）

**改动**：对 `cmd/`、`internal/`、`pkg/` 下 `gofmt -l` 报出的 26 个文件执行 `gofmt -w`，无手工改动。

**核验**

| 项 | 结果 |
| --- | --- |
| `gofmt -l ./cmd ./internal ./pkg` | ✅ 输出为空（收敛前 26 个文件） |
| `git diff -w --stat` | ✅ 仅剩 2 处 import 重排与 1 处长行折行（gofmt 的语义等价变换） |
| `go build ./...` / `go vet ./...` | ✅ 通过 |

**遗留**：`gofmt -l` 尚未纳入 CI 门禁，随 P2-4 一并落地。

### 5.24 P2-3 限流器淘汰与 `/metrics` 加固（2026-09-27）

**改动**

| 文件 | 改动 |
| --- | --- |
| `internal/middleware/auth.go` | 限流器条目改为 `limiterEntry{limiter, lastSeen atomic.Int64}`；新增 `Size`/`Cleanup(ttl)`；后台协程由「空实现」改为每 5 分钟回收空闲超过 10 分钟的条目，并上报 `rate_limiter_entries` |
| `pkg/metrics/metrics.go` | 新增 `rate_limiter_entries` gauge；`RegisterMetricsEndpoint(r, token)` 支持两种保护：配置令牌时要求 `Authorization: Bearer <token>`；未配置时只放行回环 / 私有网段，其余 403 |
| `internal/config/config.go` | 新增 `metrics_token`（绑定 `METRICS_TOKEN`） |
| `cmd/api-server/main.go` | 传入 `cfg.Server.MetricsToken` |
| `.env.example`、`docker-compose.yml` | 新增可选的 `METRICS_TOKEN` |
| `internal/middleware/ratelimit_test.go`、`pkg/metrics/metrics_endpoint_test.go`（新增） | 仓库首批 Go 单元测试：验证回收只清理空闲条目、回收后桶重新填满、令牌与网段两种保护分支 |

**验收结果**（重建 api-server 后执行）

| 场景 | 结果 |
| --- | --- |
| `go test ./...` | ✅ `internal/middleware`、`pkg/metrics` 通过（其余包无测试文件） |
| 未配置令牌 + 本机抓取 `/metrics` | ✅ 200（经 docker 网关，属私有网段） |
| 未配置令牌 + 公网来源 | ✅ 403（单元测试用 `203.0.113.7` 验证） |
| 配置令牌 | ✅ 无头 401 / 错误令牌 401 / 正确令牌 200 |
| `rate_limiter_entries` 指标 | ✅ 已在 `/metrics` 暴露 |
| Prometheus 抓取 `video-api-server` | ✅ `up`（默认内网放行路径未破坏监控） |
| `test-api.sh` 回归 | ✅ 13/13 |

**遗留**：`X-Forwarded-For` 不被信任（判定用 TCP 源地址），因此若未来在反向代理后启用 `METRICS_TOKEN` 更稳妥；Prometheus 中 `video-scheduler`（404）与 `video-worker`（连接被拒）两个抓取目标本就无 `/metrics`，属既有问题，随 P2-1 处理 worker 时一并收敛。

## 6. 实施约定

- 一次只推进一个任务，独立提交，保持可回滚。
- 完成后更新本文档状态列，并在 PR 中列出执行的验证命令。
- 破坏性变更需同步更新 `README.md` 与 `AGENTS.md`。
- 常用验证命令：`go build ./...`、`go vet ./...`、`./build.sh`、`docker compose config`、`cd frontend-new && npm test`、`./test-api.sh`。

## 7. 进度追踪

- [x] P0-1 密钥外置与轮换（代码侧完成；待人工轮换密钥与配置 CI Secret）
- [x] P0-2 写操作后失效缓存（代码侧完成；回归测试待 P2-4）
- [x] P0-3 统一容器存储路径（代码侧完成；需 Docker 端到端验证）
- [x] P0-4 上传体积与类型限制
- [x] P0-5 修复 nil 队列崩溃
- [x] P0-6 CORS 白名单
- [x] P0-7 数据库连接重试与优雅退出
- [x] P1-1 对齐接口契约（新增 2 个后端端点，修正路径/参数/表单字段，统一列表字段；前后端契约三层一致）
- [x] P1-2 流媒体鉴权方案（新增短时效、绑定视频的播放令牌 + `StreamAuthMiddleware`，访问令牌拒绝播放令牌）
- [x] P1-3 修正评论分页（`total` 改为真实总数，分页参数统一走 `parsePagination`）
- [x] P1-4 消除 N+1（评论作者名改批量 `IN` 查询，6 条评论 1 次 `users` 查询）
- [x] P1-5 统一错误码与响应（`utils.Fail`/`FailFrom` 覆盖 90 处；领域错误按 404/401 等返回，DB 错误保持 500）
- [x] P1-6 转码闭环（产物写回 `file_path` 并用于播放；失败重试上限 3 次；XPENDING+XCLAIM 回收 pending；ffmpeg 缺失不再假成功）
- [x] P1-7 会话与账户生命周期（会话落库并逐请求校验；登出 / 改密可吊销；注销账户事务级联清理）
- [x] P1-8 统一 DDL 与模型（`init.sql` 仅引导 `users`，逐表迁移容错）
- [x] P1-9 串行化启动迁移（MySQL `GET_LOCK`）
- [x] P1-10 队列统计容错（`NOGROUP` → pending=0）
- [x] P1-11 扩展视频/用户字段（分三批完成：P1-11a 用户 email/bio、P1-11b 视频封面/播放量/时长/简介、P1-11c 点赞关系与计数）
- [ ] P2-1 合并 cmd/worker
- [ ] P2-2 前端收敛与产物出库
- [x] P2-3 限流器淘汰与可观测性加固（空闲 10 分钟的条目被回收并上报 `rate_limiter_entries`；`/metrics` 支持令牌或内网限制）
- [ ] P2-4 测试与 CI 门禁
- [ ] P2-5 版本化数据库迁移
- [ ] P2-6 脚本收敛与文档修正
- [ ] P2-7 修复前端 lint 配置
- [x] P2-8 修正 git 跟踪与忽略规则（`bin/`、`frontend-new/dist/`、`storage/` 产物已出库，跟踪文件 172 → 159；依赖目录由通用 `node_modules/` 规则忽略；删除记录已暂存待提交）
- [x] P2-9 修复前端健康检查（改用 `127.0.0.1`，容器状态 healthy）
- [x] P2-10 全仓 gofmt 收敛（26 个文件已格式化，`gofmt -l` 输出为空；CI 门禁随 P2-4 落地）
