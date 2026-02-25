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
- **技术栈**: OpenTelemetry + Jaeger
- **特性**:
  - 分布式追踪
  - 跨服务调用跟踪
  - 自动注入trace context
  - 支持多种导出器 (Jaeger, stdout)

## 部署配置

### Docker Compose 配置

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

scrape_configs:
  - job_name: 'video-api-server'
    static_configs:
      - targets: ['api-server:8080']
    metrics_path: '/metrics'
```

## 使用指南

### 1. 启动监控系统

```bash
# 构建并启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f api-server
```

### 2. 访问监控界面

- **Grafana仪表板**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Jaeger追踪**: http://localhost:16686

### 3. 查看指标

访问 `/metrics` 端点查看原始指标数据：
```bash
curl http://localhost:8080/metrics
```

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
   - 验证Jaeger服务是否正常运行
   - 检查JAEGER_ENDPOINT环境变量配置

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

1. **告警配置**: 在Prometheus中配置告警规则
2. **日志聚合**: 集成ELK或类似日志聚合方案
3. **APM集成**: 考虑集成更完整的APM解决方案
4. **自动化运维**: 基于监控数据实现自动扩缩容

## 参考资料

- [Prometheus官方文档](https://prometheus.io/docs/)
- [OpenTelemetry文档](https://opentelemetry.io/docs/)
- [Grafana仪表板指南](https://grafana.com/docs/grafana/latest/dashboards/)
- [Jaeger使用手册](https://www.jaegertracing.io/docs/)