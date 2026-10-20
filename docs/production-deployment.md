# 海外生产部署指南

目标拓扑：DigitalOcean NYC3 单台应用主机、托管 MySQL/Redis、NYC3 主媒体桶、
SFO3 备份桶、Cloudflare WAF/TLS、Caddy 源站 TLS。

## 1. 上线前准备

- 准备正式域名，并在 Cloudflare 接管 DNS。
- 生成 SSH 密钥，只允许固定运维 IP 访问 22 端口。
- 在 DigitalOcean 创建 Spaces access key；主桶与备份桶使用不同凭据。
- 在 Cloudflare 创建 Origin Certificate，保存证书和私钥。
- 准备告警邮箱的 SMTP 授权码，不使用邮箱登录密码。
- 明确服务条款、隐私政策、版权投诉、滥用举报和内容审核责任人。

## 2. 基础设施

```bash
cd infra/terraform
export TF_VAR_digitalocean_token=...
export TF_VAR_cloudflare_api_token=...
terraform init -backend-config=backend.hcl
terraform apply \
  -var='cloudflare_zone=example.com' \
  -var='app_domain=video.example.com' \
  -var='ssh_key_fingerprint=aa:bb:...' \
  -var='deploy_ssh_public_key=ssh-ed25519 AAAA...' \
  -var='ssh_allowed_ips=["203.0.113.10/32"]'
```

`backend.hcl` 使用 DigitalOcean Spaces 的加密远端状态桶，不得提交到仓库。
Terraform 输出只用于定位资源；数据库 URI 和 CA 不写入仓库。

## 3. 服务器初始化

在 `/opt/video-server` 创建：

```text
.env.production
docker-compose.prod.yml
scripts/
deploy/
secrets/
```

`secrets/` 必须为 `0700`，文件为 `0600`：

```text
db_password
db_ca.pem
redis_password
redis_ca.pem
jwt_secret
metrics_token
admin_password
ghcr_token
s3_access_key
s3_secret_key
s3_backup_access_key
s3_backup_secret_key
smtp_password
grafana_admin_password
origin_cert.pem
origin_key.pem
```

从 `.env.production.example` 复制 `.env.production`，填入资源地址与域名。
为 rclone 创建 `backup-spaces` remote，生产 `.env.production` 中设置
`RCLONE_REMOTE` 与 `RCLONE_LOG_REMOTE`。

## 4. 首次发布

1. 在 GitHub 配置 `production` Environment，并限制审批人。
2. 配置 Secrets：`PROD_HOST`、`PROD_USER=deploy`、`PROD_SSH_KEY`、`PROD_KNOWN_HOSTS`、
   `PRODUCTION_URL`（供每 15 分钟外部冒烟检查）。
3. 手动触发 `Production Release`，输入已通过 CI 的提交 SHA。
4. 发布流程会备份数据库、拉取镜像、执行迁移、启动服务并检查 `/ready`。
5. 首次发布后执行管理员初始化：

```bash
cd /opt/video-server
export DB_HOST=... DB_USER=... DB_NAME=video_server
docker compose --env-file .env.production -f docker-compose.prod.yml \
  run --rm admin ensure --username ops
```

通过 Docker Secret 文件提供 `DB_PASSWORD_FILE` 与 `DB_TLS_CA_PATH`，并将
`ADMIN_PASSWORD` 注入一次性容器环境；不要写入 shell 历史或仓库。

## 5. Cloudflare

- SSL/TLS 模式：Full (strict)。
- 开启 WAF、DDoS 防护、Bot Fight Mode 和登录/上传路径限速。
- `/admin/*`、Grafana、Prometheus、Alertmanager 通过 Cloudflare Access 或
  SSH 隧道访问，不直接映射公网端口。
- 源站防火墙只允许 Cloudflare CIDR 访问 80/443。
- 启用 HSTS 前先验证域名、证书和回源链路。

## 6. 备份与恢复

- 托管 MySQL：开启自动备份与 PITR，保留至少 7 天；迁移前额外执行
  `scripts/backup-database.sh`。
- 媒体：转码后立即复制到备份桶，Scheduler 每 5 分钟补传；备份桶禁止公共 ACL。
- 日志：Docker 本地轮转保留 5 × 50 MB；每日归档到备份 Spaces，保留 30 天。
- 每月做一次恢复演练，验收标准为数据库 RPO ≤ 5 分钟、RTO ≤ 2 小时。

恢复数据库：

```bash
gunzip -c /var/backups/video-server/video-server-<timestamp>.sql.gz | \
  MYSQL_PWD="$(cat secrets/db_password)" mysql \
    --ssl-ca=secrets/db_ca.pem -h "$DB_HOST" -P "$DB_PORT" -u "$DB_USER" "$DB_NAME"
```

恢复后先跑 `migrate check`，再启动单独 staging 实例验证播放与审核流程。

## 7. 发布与回退

- 镜像以提交 SHA 为不可变标签，发布使用 GitHub Environment 人工审批。
- 应用回退：`scripts/rollback-production.sh <previous-image-tag>`。
- 数据库迁移采用 expand/contract；应用回退不自动执行破坏性 down 迁移。
- 涉及不可逆 SQL 时停止自动发布，使用数据库快照恢复并向前修复。

## 8. 上线检查

- [ ] CI 四个作业成功，镜像漏洞扫描无 HIGH/CRITICAL。
- [ ] `/ready` 返回 READY，外部可用性检测通过。
- [ ] Cloudflare 证书有效，源站拒绝非 Cloudflare 访问。
- [ ] 注册、登录、刷新、登出、改密和账号注销通过。
- [ ] 上传 → 病毒扫描 → 转码 → 待审 → 批准 → 播放通过。
- [ ] 拒绝视频不可公开访问，作者能看到拒绝原因。
- [ ] 配额、每日上传限制和管理员权限测试通过。
- [ ] MySQL PITR、Spaces 跨区域复制和恢复演练通过。
- [ ] 审计日志、告警邮件、日志归档和备份失败告警通过。
- [ ] 服务条款、隐私政策、版权投诉与滥用举报入口已发布。
