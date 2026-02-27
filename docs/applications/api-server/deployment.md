# API Server 部署指南

## 概述

API Server是视频服务的核心组件，负责处理所有HTTP请求，提供RESTful API接口。

## 架构设计

### 服务层次结构

```
HTTP请求 → Router → Middleware → Handler → Service → Repository → Database
                ↘ Authentication → Redis
```

### 核心组件

1. **Router**: 路由分发和路径匹配
2. **Middleware**: 认证、日志、限流等横切关注点
3. **Handler**: HTTP请求处理器
4. **Service**: 业务逻辑层
5. **Repository**: 数据访问层

## 部署配置

### Docker部署

#### Dockerfile

```dockerfile
# Dockerfile.apiserver
FROM golang:1.21-alpine AS builder

# 安装依赖
RUN apk add --no-cache git ca-certificates tzdata

# 设置工作目录
WORKDIR /app

# 复制go mod文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api-server ./cmd/api-server

# 最终镜像
FROM alpine:latest

# 安装ca证书
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# 从builder阶段复制二进制文件
COPY --from=builder /app/api-server .

# 复制配置文件
COPY config.yaml .
COPY .env .

# 创建必要的目录
RUN mkdir -p storage/videos storage/temp logs

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1

# 启动命令
CMD ["./api-server"]
```

#### 构建和运行

```bash
# 构建镜像
docker build -t video-server-api:latest -f Dockerfile.apiserver .

# 运行容器
docker run -d \
  --name api-server \
  -p 8080:8080 \
  -v $(pwd)/storage:/app/storage \
  -v $(pwd)/logs:/app/logs \
  --env-file .env \
  video-server-api:latest
```

### Docker Compose配置

```yaml
# docker-compose.yml
version: '3.8'

services:
  api-server:
    build:
      context: .
      dockerfile: Dockerfile.apiserver
    container_name: video_server_api
    restart: always
    ports:
      - "8080:8080"
    environment:
      - SERVER_PORT=8080
      - SERVER_MODE=release
      - DB_HOST=mysql
      - DB_PORT=3306
      - DB_USER=video_user
      - DB_PASSWORD=video_password
      - DB_NAME=video_server
      - REDIS_ADDR=redis:6379
      - JWT_SECRET=your-jwt-secret
    depends_on:
      mysql:
        condition: service_healthy
      redis:
        condition: service_started
    volumes:
      - ./storage:/app/storage
      - ./logs:/app/logs
    networks:
      - video_network
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 60s

networks:
  video_network:
    driver: bridge
```

## 配置管理

### 配置文件

```yaml
# config.yaml
server:
  port: "8080"
  mode: "release"  # debug/release/test
  read_timeout: 30
  write_timeout: 30
  idle_timeout: 60
  max_header_bytes: 1048576  # 1MB

database:
  host: "localhost"
  port: 3306
  user: "video_user"
  password: "video_password"
  name: "video_server"
  charset: "utf8mb4"
  parse_time: true
  loc: "Asia/Shanghai"
  max_idle_conns: 10
  max_open_conns: 100
  conn_max_lifetime: 3600  # 1小时

redis:
  addr: "localhost:6379"
  password: ""
  db: 0
  pool_size: 20
  min_idle_conns: 5
  dial_timeout: 5
  read_timeout: 3
  write_timeout: 3

storage:
  video_dir: "./storage/videos/"
  temp_dir: "./storage/temp/"
  max_file_size: 104857600  # 100MB
  allowed_extensions: [".mp4", ".avi", ".mov"]

jwt:
  secret: "your-jwt-secret-key-here"
  expire_hours: 24
  issuer: "video-server"
  audience: "video-users"

logging:
  level: "info"      # debug/info/warn/error
  format: "json"     # json/console
  output: "file"     # stdout/file/both
  file_path: "./logs/api-server.log"
  max_size: 100      # MB
  max_age: 30        # days
  max_backups: 10

rate_limit:
  enabled: true
  requests_per_second: 100
  burst: 200

cors:
  enabled: true
  allow_origins: ["*"]
  allow_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
  allow_headers: ["Content-Type", "Authorization", "X-Requested-With"]
```

### 环境变量优先级

```go
// 配置加载顺序：环境变量 > 配置文件 > 默认值
func LoadConfig() *Config {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath(".")
    
    // 读取配置文件
    if err := viper.ReadInConfig(); err != nil {
        log.Fatalf("Error reading config file: %v", err)
    }
    
    // 环境变量绑定
    viper.BindEnv("server.port", "SERVER_PORT")
    viper.BindEnv("database.host", "DB_HOST")
    viper.BindEnv("database.password", "DB_PASSWORD")
    viper.BindEnv("redis.addr", "REDIS_ADDR")
    viper.BindEnv("jwt.secret", "JWT_SECRET")
    
    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        log.Fatalf("Unable to decode into struct: %v", err)
    }
    
    return &config
}
```

## 启动脚本

### 生产环境启动脚本

```bash
#!/bin/bash
# start-api-server.sh

set -e

# 配置变量
APP_NAME="api-server"
APP_PORT=${SERVER_PORT:-8080}
LOG_FILE="./logs/${APP_NAME}.log"
PID_FILE="./${APP_NAME}.pid"

# 创建必要目录
mkdir -p ./logs ./storage/videos ./storage/temp

# 检查是否已在运行
if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    if ps -p "$PID" > /dev/null; then
        echo "$APP_NAME is already running (PID: $PID)"
        exit 1
    else
        rm -f "$PID_FILE"
    fi
fi

# 启动应用
echo "Starting $APP_NAME..."

# 设置环境变量
export GIN_MODE=${SERVER_MODE:-release}

# 后台启动并记录PID
nohup ./$APP_NAME > "$LOG_FILE" 2>&1 &
echo $! > "$PID_FILE"

# 等待服务启动
sleep 5

# 检查服务状态
if curl -f http://localhost:$APP_PORT/health > /dev/null 2>&1; then
    echo "$APP_NAME started successfully (PID: $(cat $PID_FILE))"
else
    echo "Failed to start $APP_NAME"
    rm -f "$PID_FILE"
    exit 1
fi
```

### 停止脚本

```bash
#!/bin/bash
# stop-api-server.sh

APP_NAME="api-server"
PID_FILE="./${APP_NAME}.pid"

if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    if ps -p "$PID" > /dev/null; then
        echo "Stopping $APP_NAME (PID: $PID)..."
        kill "$PID"
        
        # 等待优雅关闭
        TIMEOUT=30
        while [ $TIMEOUT -gt 0 ] && ps -p "$PID" > /dev/null; do
            sleep 1
            TIMEOUT=$((TIMEOUT - 1))
        done
        
        if ps -p "$PID" > /dev/null; then
            echo "Force killing $APP_NAME..."
            kill -9 "$PID"
        fi
        
        rm -f "$PID_FILE"
        echo "$APP_NAME stopped"
    else
        echo "$APP_NAME is not running"
        rm -f "$PID_FILE"
    fi
else
    echo "$APP_NAME is not running"
fi
```

## 监控集成

### 健康检查端点

```go
// 健康检查路由
func setupHealthRoutes(r *gin.Engine) {
    // 基础健康检查
    r.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "status":  "healthy",
            "service": "api-server",
            "version": "1.0.0",
            "uptime":  time.Since(startTime).String(),
        })
    })
    
    // 详细健康检查
    r.GET("/health/detail", func(c *gin.Context) {
        health := map[string]interface{}{
            "status":    "healthy",
            "database":  checkDatabaseHealth(),
            "redis":     checkRedisHealth(),
            "storage":   checkStorageHealth(),
            "system":    checkSystemHealth(),
            "timestamp": time.Now().Unix(),
        }
        
        statusCode := http.StatusOK
        for _, check := range health {
            if checkStatus, ok := check.(map[string]interface{}); ok {
                if status, exists := checkStatus["status"]; exists && status != "healthy" {
                    statusCode = http.StatusServiceUnavailable
                    health["status"] = "unhealthy"
                    break
                }
            }
        }
        
        c.JSON(statusCode, health)
    })
    
    // 就绪检查
    r.GET("/ready", func(c *gin.Context) {
        if isReady.Load() {
            c.Status(http.StatusOK)
        } else {
            c.Status(http.StatusServiceUnavailable)
        }
    })
}
```

### Prometheus指标

```go
// 自定义指标定义
var (
    httpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "api_http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "endpoint", "status"},
    )
    
    httpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "api_http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "endpoint"},
    )
    
    activeConnections = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "api_active_connections",
            Help: "Number of active HTTP connections",
        },
    )
    
    businessMetrics = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "api_business_events_total",
            Help: "Business event counters",
        },
        []string{"event_type"},
    )
)

func init() {
    // 注册指标
    prometheus.MustRegister(httpRequestsTotal)
    prometheus.MustRegister(httpRequestDuration)
    prometheus.MustRegister(activeConnections)
    prometheus.MustRegister(businessMetrics)
}

// 指标中间件
func metricsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        activeConnections.Inc()
        
        c.Next()
        
        activeConnections.Dec()
        duration := time.Since(start).Seconds()
        
        // 记录请求指标
        httpRequestsTotal.WithLabelValues(
            c.Request.Method,
            c.FullPath(),
            strconv.Itoa(c.Writer.Status()),
        ).Inc()
        
        httpRequestDuration.WithLabelValues(
            c.Request.Method,
            c.FullPath(),
        ).Observe(duration)
        
        // 记录业务指标
        if c.FullPath() == "/api/v1/videos/upload" && c.Writer.Status() == 200 {
            businessMetrics.WithLabelValues("video_upload").Inc()
        }
    }
}
```

## 安全配置

### JWT认证

```go
// JWT中间件
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        tokenString := c.GetHeader("Authorization")
        if tokenString == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
            c.Abort()
            return
        }
        
        // 验证token
        claims, err := auth.ValidateToken(tokenString)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }
        
        // 将用户信息存储到上下文
        c.Set("user_id", claims.UserID)
        c.Set("username", claims.Username)
        c.Next()
    }
}

// 权限检查中间件
func PermissionMiddleware(requiredPermission string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := c.MustGet("user_id").(uint)
        
        // 检查用户权限
        hasPermission, err := userService.HasPermission(userID, requiredPermission)
        if err != nil || !hasPermission {
            c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
            c.Abort()
            return
        }
        
        c.Next()
    }
}
```

### 请求限流

```go
// 令牌桶限流器
type RateLimiter struct {
    limiter *rate.Limiter
    keyFunc func(*gin.Context) string
}

func NewRateLimiter(rps rate.Limit, burst int, keyFunc func(*gin.Context) string) *RateLimiter {
    return &RateLimiter{
        limiter: rate.NewLimiter(rps, burst),
        keyFunc: keyFunc,
    }
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        key := rl.keyFunc(c)
        limiter := getLimiterForKey(key) // 每个用户单独的限流器
        
        if !limiter.Allow() {
            c.JSON(http.StatusTooManyRequests, gin.H{
                "error": "Rate limit exceeded",
                "retry_after": "1s",
            })
            c.Abort()
            return
        }
        
        c.Next()
    }
}

// IP地址限流
ipLimiter := NewRateLimiter(rate.Every(time.Second), 100, func(c *gin.Context) string {
    return c.ClientIP()
})

// 用户ID限流
userLimiter := NewRateLimiter(rate.Every(time.Second), 10, func(c *gin.Context) string {
    userID, exists := c.Get("user_id")
    if !exists {
        return c.ClientIP()
    }
    return fmt.Sprintf("user:%v", userID)
})
```

## 性能优化

### 连接池配置

```go
// 优化的数据库连接池
func setupOptimizedDatabase(config *DatabaseConfig) (*gorm.DB, error) {
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
        config.User, config.Password, config.Host, config.Port,
        config.Name, config.Charset, config.ParseTime, config.Loc)
    
    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
        PrepareStmt:            true,  // 预编译语句
        SkipDefaultTransaction: true,  // 跳过默认事务
    })
    
    if err != nil {
        return nil, err
    }
    
    sqlDB, err := db.DB()
    if err != nil {
        return nil, err
    }
    
    // 连接池优化配置
    sqlDB.SetMaxIdleConns(config.MaxIdleConns)
    sqlDB.SetMaxOpenConns(config.MaxOpenConns)
    sqlDB.SetConnMaxLifetime(time.Duration(config.ConnMaxLifetime) * time.Second)
    sqlDB.SetConnMaxIdleTime(10 * time.Minute)  // 最大空闲时间
    
    return db, nil
}

// Redis连接池优化
func setupOptimizedRedis(config *RedisConfig) *redis.Client {
    return redis.NewClient(&redis.Options{
        Addr:         config.Addr,
        Password:     config.Password,
        DB:           config.DB,
        PoolSize:     config.PoolSize,
        MinIdleConns: config.MinIdleConns,
        DialTimeout:  time.Duration(config.DialTimeout) * time.Second,
        ReadTimeout:  time.Duration(config.ReadTimeout) * time.Second,
        WriteTimeout: time.Duration(config.WriteTimeout) * time.Second,
        PoolTimeout:  30 * time.Second,
        IdleTimeout:  10 * time.Minute,
    })
}
```

### 缓存策略

```go
// 多级缓存实现
type MultiLevelCache struct {
    local  *bigcache.BigCache    // 本地缓存
    redis  *redis.Client         // Redis缓存
    logger *zap.Logger
}

func NewMultiLevelCache(redisClient *redis.Client, logger *zap.Logger) *MultiLevelCache {
    config := bigcache.DefaultConfig(10 * time.Minute)
    config.Shards = 1024
    config.MaxEntriesInWindow = 1000 * 10 * 60
    config.MaxEntrySize = 500
    config.Verbose = false
    
    localCache, _ := bigcache.NewBigCache(config)
    
    return &MultiLevelCache{
        local:  localCache,
        redis:  redisClient,
        logger: logger,
    }
}

func (mc *MultiLevelCache) Get(key string) ([]byte, error) {
    // L1: 本地缓存
    if data, err := mc.local.Get(key); err == nil {
        mc.logger.Debug("Cache hit - L1", zap.String("key", key))
        return data, nil
    }
    
    // L2: Redis缓存
    data, err := mc.redis.Get(context.Background(), key).Bytes()
    if err == nil {
        mc.logger.Debug("Cache hit - L2", zap.String("key", key))
        // 异步更新本地缓存
        go mc.local.Set(key, data)
        return data, nil
    }
    
    mc.logger.Debug("Cache miss", zap.String("key", key))
    return nil, err
}

func (mc *MultiLevelCache) Set(key string, data []byte, ttl time.Duration) error {
    // 同时写入两级缓存
    if err := mc.local.Set(key, data); err != nil {
        mc.logger.Warn("Failed to set L1 cache", zap.Error(err))
    }
    
    if err := mc.redis.Set(context.Background(), key, data, ttl).Err(); err != nil {
        mc.logger.Warn("Failed to set L2 cache", zap.Error(err))
        return err
    }
    
    return nil
}
```

## 故障排除

### 常见问题解决

```bash
# 检查服务状态
curl -v http://localhost:8080/health

# 查看应用日志
tail -f ./logs/api-server.log

# 检查系统资源
htop
df -h
free -m

# 网络连接检查
netstat -tlnp | grep 8080
ss -tlnp | grep 8080

# 数据库连接测试
mysql -h localhost -P 3306 -u video_user -p video_server -e "SELECT 1;"

# Redis连接测试
redis-cli -h localhost -p 6379 ping
```

### 性能分析

```bash
# 启用pprof
go tool pprof http://localhost:8080/debug/pprof/profile

# 内存分析
go tool pprof http://localhost:8080/debug/pprof/heap

# 协程分析
curl http://localhost:8080/debug/pprof/goroutine?debug=2

# 阻塞分析
go tool pprof http://localhost:8080/debug/pprof/block
```

## 部署检查清单

- [ ] 配置文件正确设置
- [ ] 环境变量已配置
- [ ] 数据库连接正常
- [ ] Redis服务可用
- [ ] 存储目录权限正确
- [ ] 日志目录可写
- [ ] 端口未被占用
- [ ] 健康检查端点可访问
- [ ] 监控指标正常暴露
- [ ] 安全配置已启用
- [ ] 备份策略已配置
- [ ] 告警规则已设置

## 下一步

- [Scheduler部署指南](../scheduler/deployment.md)
- [Worker处理逻辑](../worker/deployment.md)
- [基础设施监控](../../infrastructure/monitoring/README.md)