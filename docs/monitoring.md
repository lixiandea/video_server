# 监控和可观测性文档

## 概述

本系统集成了完整的监控和可观测性解决方案，基于可观测性的三大支柱：
- **日志 (Logging)**: 结构化日志记录
- **指标 (Metrics)**: Prometheus指标收集
- **链路追踪 (Tracing)**: OpenTelemetry分布式追踪

## 架构组件

### 1. 日志系统 (Logging)
- **技术栈**: Zap日志库
- **特性**:
  - 结构化日志输出
  - 多级别日志记录 (DEBUG, INFO, WARN, ERROR)
  - JSON格式日志支持
  - 请求上下文关联

### 2. 指标收集 (Metrics)
- **技术栈**: Prometheus client_golang
- **收集的指标**:
  - HTTP请求指标 (请求数、响应时间、请求大小)
  - 业务指标 (活跃用户数、视频上传数、观看数等)
  - 系统指标 (Goroutines数量、内存使用)
  - 数据库指标 (连接数、查询统计、查询耗时)

### 3. 链路追踪 (Tracing)
- **技术栈**: OpenTelemetry（OTLP/HTTP；可接 Jaeger / Tempo / Collector，默认未部署采集端）
- **特性**:
  - 分布式追踪
  - 跨服务调用跟踪
  - 自动注入trace context
  - 支持 OTLP 与 stdout 导出器（`TRACING_EXPORTER=stdout` 切到标准输出）

### 4. 告警通知 (Alerting)
- **技术栈**: Prometheus Alertmanager
- **链路**: Prometheus 规则触发 → Alertmanager `:9093` → 163 SMTP → `lixiandea@163.com`
- **特性**:
  - 按 `alertname` / `severity` 分组：30s 聚合、5m 间隔、4h 重复
  - 告警恢复通知（`send_resolved`）
  - 规则与通知配置都在 `deploy/monitoring/`，随 compose 挂载

## 部署配置

### Docker Compose 配置

> 以下为示例片段；实际编排以仓库根目录 `docker-compose.yml` 为准
> （含 Prometheus / Grafana / Alertmanager，未部署 Jaeger）。

```yaml
version: '3.8'
services:
  # API Server
  api-server:
    build: .
    ports:
      - "8080:8080"
    environment:
      - JAEGER_ENDPOINT=http://jaeger:14268/api/traces
      - LOG_LEVEL=info
      - LOG_FORMAT=json
    depends_on:
      - mysql
      - jaeger
    volumes:
      - ./logs:/app/logs

  # Prometheus
  prometheus:
    image: prom/prometheus:v2.37.0
    ports:
      - "9090:9090"
    volumes:
      - ./deploy/monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'

  # Grafana
  grafana:
    image: grafana/grafana:9.1.0
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana_data:/var/lib/grafana
      - ./deploy/monitoring/grafana-dashboard.json:/var/lib/grafana/dashboards/video-server.json
    depends_on:
      - prometheus

  # Jaeger
  jaeger:
    image: jaegertracing/all-in-one:1.37
    ports:
      - "16686:16686"  # UI
      - "14268:14268"  # Collector HTTP
    environment:
      - COLLECTOR_ZIPKIN_HOST_PORT=:9411

volumes:
  prometheus_data:
  grafana_data:
```

### Prometheus 配置

```yaml
# deploy/monitoring/prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - /etc/prometheus/alerts.yml

scrape_configs:
  - job_name: 'video-api-server'
    static_configs:
      - targets: ['api-server:8080']
    metrics_path: '/metrics'

  - job_name: 'video-scheduler'
    static_configs:
      - targets: ['scheduler:8089']
    metrics_path: '/metrics'
    scrape_interval: 30s
```

## 使用指南

### 1. 启动监控系统

```bash
# 构建并启动所有服务
docker compose up -d

# 查看服务状态
docker compose ps

# 查看日志
docker compose logs -f api-server
```

### 2. 访问监控界面

- **Grafana 仪表板**: http://localhost:3301（管理员密码见 `.env` 的 `GRAFANA_ADMIN_PASSWORD`）
- **Prometheus**: http://localhost:9090（告警页面 `/alerts`，规则页面 `/rules`）
- **Alertmanager**: http://localhost:9093（告警分组、静默与通知状态）
- **Jaeger 追踪**: 未包含在 `docker-compose.yml` 中；`pkg/tracing` 仅保留接入接口

### 3. 查看指标

访问 `/metrics` 端点查看原始指标数据：
```bash
curl http://localhost:8080/metrics
```

### 4. 查看告警

```bash
# 当前处于 pending / firing 的告警
curl -s http://localhost:9090/api/v1/alerts | jq

# 已加载的规则与上次求值结果
curl -s http://localhost:9090/api/v1/rules | jq
```

Prometheus 页面：http://localhost:9090/alerts 与 http://localhost:9090/rules

## 告警规则与 SLO

规则文件：[`deploy/monitoring/alerts.yml`](../deploy/monitoring/alerts.yml)，由 `prometheus.yml` 的
`rule_files` 加载，`docker-compose.yml` 将其只读挂载到 `/etc/prometheus/alerts.yml`。

| 告警 | 级别 | 条件 | 持续 |
| --- | --- | --- | --- |
| `ApiServerDown` | critical | `up{job="video-api-server"} == 0` | 1m |
| `SchedulerDown` | warning | `up{job="video-scheduler"} == 0` | 2m |
| `HighHttpErrorRate` | critical | 5xx 比例 > 5% | 5m |
| `HighHttpLatencyP95` | warning | p95 > 1s | 10m |
| `TranscodeQueueBacklog` | warning | `transcode_queue_pending > 50` | 10m |
| `DatabaseConnectionsHigh` | warning | `db_connections_open > 80` | 10m |

同时提供两条记录规则，便于在 Grafana 与告警中复用：`job:http_request_error_ratio:5m`、
`job:http_request_duration_p95:5m`。

**SLO 起步基线**

| SLI | 目标 | 说明 |
| --- | --- | --- |
| API 可用性 | 99.5% / 月 | 以 `up{job="video-api-server"}` 计算，约允许 3.6 小时不可用 |
| API 5xx 比例 | < 1% | 5 分钟窗口；告警阈值 5% 留出缓冲 |
| API p95 延迟 | < 1s | 5 分钟窗口 |
| 转码积压 | pending < 50 | 持续 10 分钟视为异常 |

阈值是单机 compose 的起步值，接入真实流量后再收紧。修改规则后先校验：

```bash
docker run --rm --entrypoint promtool \
  -v "$PWD/deploy/monitoring:/rules:ro" \
  prom/prometheus:v2.40.0 check rules /rules/alerts.yml
```

> `transcode_queue_*` 由 api-server 每 30 秒从 Redis 采集后上报（worker 不单独暴露 `/metrics`）；
> `db_connections_open` 由数据库连接池每 30 秒刷新。

## 邮件通知配置（163 SMTP）

Prometheus 触发告警后转发给 Alertmanager（`:9093`），由它按 `route` 把邮件发给
`lixiandea@163.com`。配置见
[`deploy/monitoring/alertmanager.yml`](../deploy/monitoring/alertmanager.yml)。

1. 163 邮箱 → 设置 → POP3/SMTP/IMAP → 开启 **SMTP 服务** → 获取 **授权码**（不是登录密码）。
2. 把授权码写入 `.env`：

   ```bash
   ALERTMANAGER_SMTP_PASSWORD=<163 授权码>
   ```

3. 重建 Alertmanager（改了 `.env` 必须重建，`restart` 不会更新容器环境变量）：

   ```bash
   docker compose up -d alertmanager
   ```

   > 只修改 `alertmanager.yml` 时，`docker compose restart alertmanager` 即可
   > （启动脚本会重新生成临时配置）。

4. 发一条测试告警验证投递：

   ```bash
   docker exec video_server_alertmanager \
     amtool alert add alertname=EmailPipelineTest severity=warning \
     --annotation='summary="Alertmanager 邮件通道测试"' \
     --alertmanager.url=http://localhost:9093
   ```

   Alertmanager 在 `group_wait`（30s）后尝试发送；失败时看
   `docker compose logs --tail=100 alertmanager`。常见错误 `535 authentication failed`
   表示授权码未填或已失效。

   > 测试告警默认 5 分钟后自动过期；也可以等它在 Alertmanager 页面消失。

> 当前 `.env` 中 `ALERTMANAGER_SMTP_PASSWORD` 为空，**邮件投递尚未验证**（该步骤已按计划跳过）。
> 需要启用时，按上面步骤填入 163 授权码并重建 Alertmanager 即可。

告警邮件默认包含 `alertname`、`severity`、`summary`、`description`；恢复时也会发送
`[RESOLVED]` 通知（`send_resolved: true`）。

## 告警 Runbook

- **ApiServerDown**：`docker compose ps` 看容器状态 → `docker compose logs --tail=200 api-server` →
  检查 MySQL/Redis 是否健康 → 确认 8080 端口未被其他进程占用。
- **SchedulerDown**：`docker compose logs --tail=200 scheduler`；确认 8089 端口与 `/metrics` 正常；
  清理任务停摆期间 API 读写不受影响。
- **HighHttpErrorRate**：按 `endpoint` / `status` 拆分 `http_requests_total` 定位 5xx 集中的接口；
  再查 api-server 日志中的 `level=error` 与数据库连接/慢查询。
- **HighHttpLatencyP95**：先看 `db_query_duration_seconds` 的 p95 与 Redis 延迟；再看是否被大文件
  上传/播放占满带宽。
- **TranscodeQueueBacklog**：`docker compose ps transcode-worker` 与
  `docker logs --tail=200 video_server_transcode_worker`；检查 `XPENDING` 与
  `TRANSCODE_CONCURRENCY`。
- **DatabaseConnectionsHigh**：`SHOW PROCESSLIST` 看长事务；确认连接池大小与 `max_connections`
  的余量；必要时重启发生泄漏的服务。

## 代码集成示例

### 添加日志记录

```go
import "github.com/lixiande/video_server/pkg/logging"

// 基本日志记录
logger := logging.GetLogger()
logger.Info("User registered successfully", 
    "user_id", userID, 
    "username", username)

// 错误日志记录
logger.Error("Database connection failed", "error", err)

// 上下文日志记录
ctx := logging.WithContext(context.Background(), logger)
logging.FromContext(ctx).Info("Processing request")
```

### 添加指标收集

```go
import "github.com/lixiande/video_server/pkg/metrics"

// 递增计数器
metrics.IncVideoUploads()
metrics.IncVideoViews()

// 设置Gauge值
metrics.SetDBConnections(float64(connectionCount))

// 观察直方图
metrics.ObserveDBQueryDuration("users", "SELECT", duration)
```

### 添加链路追踪

```go
import "github.com/lixiande/video_server/pkg/tracing"

// 开始新的span
ctx, span := tracing.StartSpan(ctx, "user_registration")
defer span.End()

// 添加属性
tracing.AddSpanAttribute(ctx, "user.id", userID)

// 记录事件
tracing.AddSpanEvent(ctx, "validation_completed", map[string]interface{}{
    "validation_time": time.Since(startTime),
})

// 记录错误
tracing.RecordError(ctx, err)
```

## 监控最佳实践

### 1. 日志级别使用
- **DEBUG**: 详细的调试信息，仅在开发环境中使用
- **INFO**: 重要的业务事件和系统状态变更
- **WARN**: 潜在的问题但不影响系统正常运行
- **ERROR**: 错误事件，需要关注和处理

### 2. 指标命名规范
```
<namespace>_<subsystem>_<metric_name>_<unit>
例如: http_requests_total, db_query_duration_seconds
```

### 3. 追踪Span设计
- 每个重要业务操作都应该有对应的span
- 合理设置span的粒度
- 添加有意义的属性和事件

## 故障排除

### 常见问题

1. **指标未显示**
   - 检查Prometheus是否能访问/metrics端点
   - 确认防火墙规则允许相应端口

2. **追踪数据缺失**
   - 确认已按需调用 `tracing.InitTracer`（默认未启用）
   - 检查 OTLP collector 是否可达（`OTEL_EXPORTER_OTLP_ENDPOINT`，默认 `localhost:4318`）

3. **日志文件过大**
   - 配置日志轮转策略
   - 调整日志级别减少输出

### 性能监控

通过以下方式监控系统性能：

```bash
# 查看实时指标
watch -n 1 'curl -s http://localhost:8080/metrics | grep http_requests_total'

# 查看系统资源使用
docker stats

# 查看日志增长情况
du -sh logs/
```

## 扩展建议

1. **告警通知**: 已接入 Alertmanager + 163 邮件；下一步可增加飞书/企业微信/Webhook 与静默流程
2. **日志聚合**: 集成ELK或类似日志聚合方案
3. **APM集成**: 考虑集成更完整的APM解决方案
4. **自动化运维**: 基于监控数据实现自动扩缩容

## 参考资料

- [Prometheus官方文档](https://prometheus.io/docs/)
- [OpenTelemetry文档](https://opentelemetry.io/docs/)
- [Grafana仪表板指南](https://grafana.com/docs/grafana/latest/dashboards/)
- [Jaeger使用手册](https://www.jaegertracing.io/docs/)
