# 安全与密钥管理

> 覆盖依赖漏洞巡检、密钥轮换与配置检查。巡检命令：`make security-check`。

## 1. 依赖漏洞巡检

### 后端（Go）

```bash
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

2026-10-18 基线（本机 Go 1.26.5）：

- `golang.org/x/net@v0.10.0`：HTTP/2 与解析相关公告（GO-2026-6610/6611/6612/6617 等）；
- `go.opentelemetry.io/otel/sdk@v1.12.0`：GO-2026-6616；
- `net/http`（随 Go 工具链）：受所用 Go 版本影响。

结论：需要升级 Go 工具链与上述依赖。建议顺序：

1. 把 CI 的 `setup-go` 与各 Dockerfile 的 `golang` 基础镜像升到同一个受支持版本（如 1.24/1.25）；
2. `go get -u golang.org/x/net go.opentelemetry.io/otel/...` 后 `go mod tidy`；
3. 运行 `go build ./... && go vet ./... && go test ./...`，再做一次上传→转码→播放的集成回归。

在完成升级前，`govulncheck` 作为**巡检项**（`make security-check` 不因此失败），暂不设为 CI 阻断门禁。

### 前端（npm）

```bash
cd frontend-new
npm audit --omit=dev --audit-level=high   # 生产依赖：CI 门禁
npm audit                                  # 含 dev 依赖：巡检
```

- 生产依赖：2026-10-18 执行 `npm audit fix`（非破坏性）后为 **0** 个漏洞。
- dev 依赖：`esbuild`、`postcss-selector-parser`、`tinypool` 仍有公告；修复需要
  `npm audit fix --force`（会带来 vite/vitest 主版本升级），单独安排并回归。

## 2. 密钥轮换

仓库历史里出现过明文数据库口令与 JWT 默认密钥，**一律视为已泄露**。

| 密钥 | 存放位置 | 轮换方式 |
| --- | --- | --- |
| MySQL root 口令 | 数据卷（首次初始化写入） | `ALTER USER 'root'@'%' IDENTIFIED BY '<新口令>';` |
| MySQL 应用口令 | 数据卷（首次初始化写入） | `ALTER USER 'video_user'@'%' IDENTIFIED BY '<新口令>';` |
| `JWT_SECRET` | `.env`（≥32 字符） | 生成新随机值并重建 api-server |
| `GRAFANA_ADMIN_PASSWORD` | Grafana 数据卷（首次初始化写入） | `grafana-cli admin reset-admin-password <新口令>` |
| `ALERTMANAGER_SMTP_PASSWORD` | `.env` | 163 SMTP 授权码，泄露后重新生成 |

本地 compose 的轮换步骤：

```bash
# 1. 修改 MySQL 账号口令（在容器内执行）
docker exec -i video_server_mysql mysql -uroot -p"<旧 root 口令>" <<'SQL'
ALTER USER 'root'@'%' IDENTIFIED BY '<新 root 口令>';
ALTER USER 'video_user'@'%' IDENTIFIED BY '<新应用口令>';
FLUSH PRIVILEGES;
SQL

# 2. 更新 .env 的 MYSQL_ROOT_PASSWORD / DB_PASSWORD / JWT_SECRET
# 3. 重建业务服务（数据卷不会被重建，口令以 ALTER USER 为准）
docker compose up -d --force-recreate api-server scheduler transcode-worker

# 4. 验证
curl -fs http://localhost:8080/health
docker compose run --rm migrate check
```

注意：

- 本机数据卷目前仍在沿用历史口令，建议尽快按上表轮换；轮换后旧口令只存在于 git 历史中。
- 生产环境通过部署平台注入 Secret，不要提交 `.env`；建议每 90 天轮换一次，人员变动后立即轮换。

## 3. 配置检查清单

- [ ] `.env` 未被跟踪：`git ls-files | grep -E '^\.env$'` 输出为空
- [ ] `JWT_SECRET` ≥ 32 字符且不是默认值
- [ ] 对外暴露 `/metrics` 时配置 `METRICS_TOKEN`
- [ ] 生产环境 CORS 白名单不包含 `*`
- [ ] 业务数据库账号不使用 root，并遵循最小权限
