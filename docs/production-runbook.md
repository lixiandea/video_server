# 生产运行手册

## 媒体备份

检查 Scheduler 指标：

```promql
media_replication_pending{job="video-scheduler"}
media_replication_lag_seconds{job="video-scheduler"}
```

- 延迟超过 10 分钟：检查 Spaces 两类凭据、备份桶权限、Scheduler 日志。
- Scheduler 正常但积压不降：检查对象大小、跨区域带宽与每分钟处理上限。
- 禁止直接用 `rclone sync --delete` 镜像备份桶，避免误删历史对象。

## 内容审核

- 管理员从 `/admin/review` 处理待审视频。
- 批准后视频才进入公开列表和推荐；拒绝必须填写原因。
- 对版权或违法内容投诉，先立即拒绝/下架，再保留审计和申诉记录。
- 待审积压超过 50 个并持续 24 小时会触发告警。

## 审计日志

- `audit_logs_dropped_total` 增长表示异步队列满，检查 MySQL 延迟和表大小。
- 审计保留期默认 30 天，由 Scheduler 每小时清理。
- 审计日志只记录请求元数据，不记录密码、Cookie 或请求体。

## 数据库故障

1. 检查 DigitalOcean Managed Database 状态与连接数。
2. API `/ready` 会返回 503，Caddy 和应用健康检查会失败。
3. 禁止在数据库不可用时执行迁移。
4. 从 PITR 恢复到新集群，验证后切换 `DB_HOST`。

## Redis / 队列故障

- Redis 丢失时，Worker 和 Scheduler 会从 `transcode_task_records` 恢复未完成任务。
- 检查 `transcode_queue_pending`、`transcode_queue_processing` 和持久任务状态。
- 不直接删除 Redis Stream；先确认数据库任务状态与备份。

## 证书与告警

- Cloudflare Origin Certificate 到期前 30 天轮换，更新 Docker Secret 后重建 Caddy。
- 告警邮件恢复通知必须真实送达；授权码泄露后立即在邮箱后台重新生成。
- 所有监控页面只通过 Cloudflare Access 或 SSH 隧道访问。

## 事件响应

1. 先保护数据与用户：必要时关闭上传或切换维护页。
2. 保留审计日志、容器日志和对象版本，不覆盖现场。
3. 轮换可能泄露的数据库、Redis、S3、JWT、SMTP 和 Cloudflare 凭据。
4. 修复后执行恢复演练和最小回归，再逐步恢复流量。
