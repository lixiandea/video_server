# 监控系统部署文档

## 概述

视频服务项目采用完整的可观测性解决方案，包含指标收集、日志管理和分布式追踪三个核心组件。

## 系统架构

```
应用服务 → Prometheus (指标收集)
        → Loki (日志收集)  
        → Tempo/Jaeger (追踪收集)
        ↓
Grafana ← 各种数据源 (数据可视化)
```

## 组件介绍

### 1. Prometheus - 指标收集
- 时间序列数据库
- 多维数据模型
- 强大的查询语言(PromQL)

### 2. Grafana - 数据可视化
- 丰富的仪表板
- 多种数据源支持
- 告警功能

### 3. Jaeger - 分布式追踪
- 端到端请求追踪
- 性能瓶颈分析
- 服务依赖关系可视化

## 部署配置

### Docker Compose 配置

```yaml
# docker-compose.monitoring.yml
version: '3.8'

services:
  # Prometheus - 指标收集
  prometheus:
    image: prom/prometheus:v2.40.0
    container_name: video_server_prometheus
    restart: always
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
      - '--storage.tsdb.retention.time=30d'
      - '--web.enable-lifecycle'
    networks:
      - monitoring_network

  # Grafana - 数据可视化
  grafana:
    image: grafana/grafana:9.3.0
    container_name: video_server_grafana
    restart: always
    ports:
      - "3001:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
      - GF_USERS_ALLOW_SIGN_UP=false
      - GF_SERVER_ROOT_URL=http://localhost:3001
    volumes:
      - grafana_data:/var/lib/grafana
      - ./deploy/monitoring/grafana-datasources.yml:/etc/grafana/provisioning/datasources/datasources.yml
      - ./deploy/monitoring/grafana-dashboard.yml:/etc/grafana/provisioning/dashboards/dashboard.yml
      - ./deploy/monitoring/dashboards:/var/lib/grafana/dashboards
    depends_on:
      - prometheus
    networks:
      - monitoring_network

  # Jaeger - 分布式追踪
  jaeger:
    image: jaegertracing/all-in-one:1.40
    container_name: video_server_jaeger
    restart: always
    ports:
      - "16686:16686"  # UI界面
      - "14268:14268"  # Collector HTTP
      - "6831:6831/udp" # Jaeger Thrift Compact
    environment:
      - SPAN_STORAGE_TYPE=badger
      - BADGER_EPHEMERAL=false
      - BADGER_DIRECTORY_VALUE=/badger/data
      - BADGER_DIRECTORY_KEY=/badger/key
      - COLLECTOR_ZIPKIN_HOST_PORT=:9411
      - COLLECTOR_OTLP_ENABLED=true
    volumes:
      - jaeger_data:/badger
    networks:
      - monitoring_network

  # Loki - 日志收集 (可选)
  loki:
    image: grafana/loki:2.7.0
    container_name: video_server_loki
    restart: always
    ports:
      - "3100:3100"
    command: -config.file=/etc/loki/local-config.yaml
    volumes:
      - loki_data:/loki
    networks:
      - monitoring_network

  # Promtail - 日志收集代理 (配合Loki使用)
  promtail:
    image: grafana/promtail:2.7.0
    container_name: video_server_promtail
    restart: always
    volumes:
      - ./logs:/var/log/app:ro
      - ./deploy/monitoring/promtail.yml:/etc/promtail/config.yml
    command: -config.file=/etc/promtail/config.yml
    networks:
      - monitoring_network

volumes:
  prometheus_data:
  grafana_data:
  jaeger_data:
  loki_data:

networks:
  monitoring_network:
    driver: bridge
```

### Prometheus 配置

创建 `deploy/monitoring/prometheus.yml`：

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "alert.rules"

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093

scrape_configs:
  # API Server监控
  - job_name: 'api-server'
    static_configs:
      - targets: ['api-server:8080']
    metrics_path: '/metrics'
    relabel_configs:
      - source_labels: [__address__]
        target_label: instance
        replacement: 'api-server'

  # Scheduler监控
  - job_name: 'scheduler'
    static_configs:
      - targets: ['scheduler:8089']
    metrics_path: '/metrics'

  # Node Exporter (系统指标)
  - job_name: 'node-exporter'
    static_configs:
      - targets: ['node-exporter:9100']

  # MySQL Exporter
  - job_name: 'mysql-exporter'
    static_configs:
      - targets: ['mysql-exporter:9104']

  # Redis Exporter
  - job_name: 'redis-exporter'
    static_configs:
      - targets: ['redis-exporter:9121']
```

### Grafana 数据源配置

创建 `deploy/monitoring/grafana-datasources.yml`：

```yaml
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    
  - name: Jaeger
    type: jaeger
    access: proxy
    url: http://jaeger:16686
    
  - name: Loki
    type: loki
    access: proxy
    url: http://loki:3100
```

### Grafana 仪表板配置

创建 `deploy/monitoring/grafana-dashboard.yml`：

```yaml
apiVersion: 1

providers:
  - name: 'default'
    orgId: 1
    folder: ''
    type: file
    disableDeletion: false
    editable: true
    options:
      path: /var/lib/grafana/dashboards
```

## 应用集成

### 1. Prometheus指标集成

```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "net/http"
)

// 预定义指标
var (
    // HTTP请求指标
    httpRequestTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "endpoint", "status"},
    )

    httpRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "endpoint"},
    )

    // 业务指标
    activeUsers = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "active_users_count",
            Help: "Number of active users",
        },
    )

    videoUploadsTotal = promauto.NewCounter(
        prometheus.CounterOpts{
            Name: "video_uploads_total",
            Help: "Total number of video uploads",
        },
    )

    // 系统指标
    dbConnections = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "database_connections",
            Help: "Number of active database connections",
        },
    )
)

// 注册指标端点
func RegisterMetricsEndpoint(r *gin.Engine) {
    r.GET("/metrics", gin.WrapH(promhttp.Handler()))
}

// 中间件集成
func MetricsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        
        c.Next()
        
        // 记录请求指标
        httpRequestTotal.WithLabelValues(
            c.Request.Method,
            c.FullPath(),
            strconv.Itoa(c.Writer.Status()),
        ).Inc()
        
        httpRequestDuration.WithLabelValues(
            c.Request.Method,
            c.FullPath(),
        ).Observe(time.Since(start).Seconds())
    }
}

// 业务指标更新
func IncVideoUploads() {
    videoUploadsTotal.Inc()
}

func SetActiveUsers(count float64) {
    activeUsers.Set(count)
}

func SetDBConnections(count float64) {
    dbConnections.Set(count)
}
```

### 2. Jaeger追踪集成

```go
package tracing

import (
    "context"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/jaeger"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
    "go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

func InitTracer(serviceName, collectorEndpoint string) error {
    // 创建Jaeger导出器
    exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(collectorEndpoint)))
    if err != nil {
        return err
    }

    // 创建追踪器提供者
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exp),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String(serviceName),
        )),
    )

    // 设置全局追踪器
    otel.SetTracerProvider(tp)
    tracer = tp.Tracer(serviceName)
    
    return nil
}

// 开始新的span
func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
    return tracer.Start(ctx, name)
}

// 添加span属性
func AddSpanAttribute(ctx context.Context, key string, value interface{}) {
    span := trace.SpanFromContext(ctx)
    span.SetAttributes(attribute.Any(key, value))
}

// 记录事件
func AddSpanEvent(ctx context.Context, name string, attrs map[string]interface{}) {
    span := trace.SpanFromContext(ctx)
    var keyValueAttrs []attribute.KeyValue
    for k, v := range attrs {
        keyValueAttrs = append(keyValueAttrs, attribute.Any(k, v))
    }
    span.AddEvent(name, trace.WithAttributes(keyValueAttrs...))
}

// 记录错误
func RecordError(ctx context.Context, err error) {
    span := trace.SpanFromContext(ctx)
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())
}
```

### 3. 结构化日志集成

```go
package logging

import (
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

var logger *zap.Logger

func InitLogger(level string, output string) error {
    var config zap.Config
    
    if level == "debug" {
        config = zap.NewDevelopmentConfig()
    } else {
        config = zap.NewProductionConfig()
        config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
    }
    
    // 设置日志级别
    lvl, err := zapcore.ParseLevel(level)
    if err != nil {
        return err
    }
    config.Level.SetLevel(lvl)
    
    // 设置输出
    if output == "file" {
        config.OutputPaths = []string{"logs/app.log"}
        config.ErrorOutputPaths = []string{"logs/error.log"}
    }
    
    var err error
    logger, err = config.Build()
    return err
}

func GetLogger() *zap.Logger {
    return logger
}

// 带上下文的日志记录
func WithContext(ctx context.Context, fields ...zap.Field) context.Context {
    return context.WithValue(ctx, "logger", logger.With(fields...))
}

func FromContext(ctx context.Context) *zap.Logger {
    if l, ok := ctx.Value("logger").(*zap.Logger); ok {
        return l
    }
    return logger
}
```

## 预设仪表板

### 1. API服务监控仪表板

```json
{
  "dashboard": {
    "title": "API Service Dashboard",
    "panels": [
      {
        "title": "HTTP Requests Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])",
            "legendFormat": "{{method}} {{endpoint}}"
          }
        ]
      },
      {
        "title": "Request Duration",
        "type": "graph", 
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "{{method}} {{endpoint}} p95"
          }
        ]
      },
      {
        "title": "Error Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total{status=~\"5..\"}[5m])",
            "legendFormat": "5xx Errors"
          }
        ]
      }
    ]
  }
}
```

### 2. 数据库监控仪表板

```json
{
  "dashboard": {
    "title": "Database Monitoring",
    "panels": [
      {
        "title": "Active Connections",
        "type": "gauge",
        "targets": [
          {
            "expr": "database_connections"
          }
        ]
      },
      {
        "title": "Query Performance",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(db_query_duration_seconds_sum[5m]) / rate(db_query_duration_seconds_count[5m])",
            "legendFormat": "Avg Query Time"
          }
        ]
      }
    ]
  }
}
```

## 告警配置

### Prometheus告警规则

创建 `deploy/monitoring/alert.rules`：

```yaml
groups:
- name: video-server-alerts
  rules:
  # 高错误率告警
  - alert: HighErrorRate
    expr: rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m]) > 0.05
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "High error rate detected"
      description: "{{ $labels.method }} {{ $labels.endpoint }} has error rate of {{ printf \"%.2f\" $value }}%"

  # 高延迟告警
  - alert: HighLatency
    expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 1
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "High latency detected"
      description: "{{ $labels.method }} {{ $labels.endpoint }} has 95th percentile latency of {{ printf \"%.2f\" $value }}s"

  # 数据库连接过多
  - alert: TooManyDBConnections
    expr: database_connections > 50
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "Too many database connections"
      description: "Currently {{ $value }} database connections active"
```

### Alertmanager配置

```yaml
# alertmanager.yml
global:
  smtp_smarthost: 'smtp.gmail.com:587'
  smtp_from: 'alerts@yourcompany.com'
  smtp_auth_username: 'your-email@gmail.com'
  smtp_auth_password: 'your-app-password'

route:
  group_by: ['alertname']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 3h
  receiver: 'team-email'

receivers:
- name: 'team-email'
  email_configs:
  - to: 'dev-team@yourcompany.com'
    send_resolved: true
```

## 使用指南

### 1. 启动监控系统

```bash
# 启动监控组件
docker-compose -f docker-compose.monitoring.yml up -d

# 启动应用服务（包含监控集成）
docker-compose up -d api-server scheduler worker
```

### 2. 访问监控界面

- **Grafana**: http://localhost:3001 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Jaeger**: http://localhost:16686

### 3. 查看指标

```bash
# 查看原始指标
curl http://localhost:8080/metrics

# 查询Prometheus指标
curl 'http://localhost:9090/api/v1/query?query=http_requests_total'
```

### 4. 追踪请求

在应用代码中添加追踪：

```go
// 在处理请求时开始追踪
ctx, span := tracing.StartSpan(c.Request.Context(), "user_login")
defer span.End()

// 添加业务相关信息
tracing.AddSpanAttribute(ctx, "user.id", userID)
tracing.AddSpanAttribute(ctx, "login.method", "password")

// 记录关键事件
tracing.AddSpanEvent(ctx, "validation_completed")
```

## 性能基准

### 推荐资源配置

| 组件 | CPU | 内存 | 存储 |
|------|-----|------|------|
| Prometheus | 2核 | 4GB | 50GB |
| Grafana | 1核 | 2GB | 10GB |
| Jaeger | 2核 | 4GB | 20GB |
| Loki | 2核 | 4GB | 100GB |

### 数据保留策略

```yaml
# Prometheus数据保留
--storage.tsdb.retention.time=30d

# Jaeger数据保留
BADGER_EPHEMERAL=false
BADGER_DIRECTORY_VALUE=/badger/data

# Loki数据保留
chunk_store_config:
  max_look_back_period: 720h  # 30天
```

## 故障排除

### 常见问题

1. **指标未显示**
   ```bash
   # 检查应用是否暴露metrics端点
   curl http://api-server:8080/metrics
   
   # 检查Prometheus抓取配置
   docker-compose logs prometheus
   ```

2. **追踪数据缺失**
   ```bash
   # 检查Jaeger服务状态
   docker-compose logs jaeger
   
   # 验证追踪配置
   echo $JAEGER_ENDPOINT
   ```

3. **Grafana无法连接数据源**
   ```bash
   # 检查网络连接
   docker-compose exec grafana ping prometheus
   
   # 检查数据源配置
   cat deploy/monitoring/grafana-datasources.yml
   ```

## 扩展建议

1. **日志聚合**: 集成ELK Stack进行日志分析
2. **APM集成**: 考虑使用商业APM解决方案
3. **自动化运维**: 基于监控数据实现自动扩缩容
4. **安全监控**: 添加安全事件监控和告警

## 参考资源

- [Prometheus官方文档](https://prometheus.io/docs/)
- [Grafana仪表板库](https://grafana.com/grafana/dashboards/)
- [OpenTelemetry文档](https://opentelemetry.io/docs/)
- [Jaeger使用手册](https://www.jaegertracing.io/docs/)