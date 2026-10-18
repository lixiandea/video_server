# 安全与密钥管理

> 覆盖依赖漏洞巡检、密钥轮换与配置检查。巡检命令：`make security-check`。

## 1. 依赖漏洞巡检

### 后端（Go）

```bash
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

2026-10-18 升级后基线（Go 1.27.2）：

- 被调用代码：**0** 个漏洞（升级前 22 个）；
- 导入包：0 个；
- 仅剩 2 个位于依赖模块、当前代码未调用的公告（`govulncheck -show verbose` 可见）。

本次升级：

- 工具链：CI `setup-go` 与各 Dockerfile 统一到 Go 1.27（`go.mod` 最低要求 1.26）；
- `golang.org/x/net v0.10.0 → v0.61.0`、`golang.org/x/crypto v0.12.0 → v0.58.0`；
- `go.opentelemetry.io/otel* v1.12.0 → v1.47.0`；
- `github.com/golang-jwt/jwt/v5 v5.0.0 → v5.3.1`（GO-2025-3553）。

`govulncheck@v1.9.0` 现已作为 CI 阻断门禁（`make security-check` 同样会失败退出）。

仍未处理：

- dev 依赖（vite/vitest 体系）需要 `npm audit fix --force` 做主版本升级；
- tracing 已从弃用的 Jaeger exporter 切换到 OTLP/HTTP（`OTEL_EXPORTER_OTLP_ENDPOINT`，
  默认 `localhost:4318`）；`InitTracer` 目前仍未在服务启动时调用。

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
