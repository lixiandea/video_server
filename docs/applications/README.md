# 应用服务部署文档

本目录包含视频服务项目各应用组件的部署和配置文档。

## 目录结构

```
applications/
├── api-server/        # API网关服务
│   ├── deployment.md # 部署指南
│   ├── configuration.md # 配置说明
│   └── scaling.md    # 扩缩容指南
├── scheduler/         # 任务调度服务
│   ├── deployment.md # 部署指南
│   └── jobs.md       # 任务配置
├── worker/            # 后台工作服务
│   ├── deployment.md # 部署指南
│   └── processing.md # 处理逻辑
├── frontend/          # 前端服务
│   ├── deployment.md # 部署指南
│   └── nginx.md      # Nginx配置
└── README.md         # 本文件
```

## 应用服务概览

### 1. API Server (api-server)
- **端口**: 8080
- **职责**: 处理所有HTTP请求，提供RESTful API
- **技术栈**: Go + Gin框架
- **依赖**: MySQL, Redis

### 2. Scheduler (scheduler)
- **端口**: 8089
- **职责**: 执行定时任务和后台作业
- **技术栈**: Go
- **依赖**: MySQL

### 3. Worker (worker)
- **端口**: 无(后台服务)
- **职责**: 处理异步任务和队列作业
- **技术栈**: Go
- **依赖**: MySQL, Redis

### 4. Frontend (frontend)
- **端口**: 3000
- **职责**: 提供Web界面和静态资源
- **技术栈**: HTML/CSS/JavaScript 或 Vue.js
- **依赖**: API Server

## 部署架构

```
外部请求 → Load Balancer → API Server (多实例)
                    ↓
              Scheduler (单实例)
                    ↓
              Worker (多实例)
                    ↓
              Frontend (多实例)
```

## 部署方式对比

| 部署方式 | 适用场景 | 优点 | 缺点 |
|---------|---------|------|------|
| Docker Compose | 开发/测试环境 | 简单快捷 | 不适合生产 |
| Kubernetes | 生产环境 | 高可用、自动扩缩容 | 复杂度高 |
| 传统部署 | 特定需求 | 灵活性高 | 运维成本高 |

## 快速开始

### Docker Compose 部署 (推荐用于开发)

```bash
# 进入项目根目录
cd /path/to/video_server

# 构建所有服务镜像
docker-compose build

# 启动所有应用服务
docker-compose up -d api-server scheduler worker frontend

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f api-server
```

### 独立部署各服务

#### API Server

```bash
# 构建
cd cmd/api-server
go build -o ../../bin/api-server .

# 运行
cd ../..
./bin/api-server
```

#### Scheduler

```bash
# 构建
cd cmd/scheduler
go build -o ../../bin/scheduler .

# 运行
cd ../..
./bin/scheduler
```

#### Worker

```bash
# 构建
cd cmd/worker
go build -o ../../bin/worker .

# 运行
cd ../..
./bin/worker
```

## 环境配置

### 配置文件结构

```yaml
# config.yaml - 主配置文件
server:
  port: "8080"
  mode: "release"  # debug/release/test
  read_timeout: 30
  write_timeout: 30

database:
  host: "mysql"
  port: 3306
  user: "video_user"
  password: "video_password"
  name: "video_server"
  charset: "utf8mb4"

redis:
  addr: "redis:6379"
  password: ""
  db: 0

storage:
  video_dir: "./storage/videos/"
  temp_dir: "./storage/temp/"

jwt:
  secret: "your-jwt-secret-key"
  expire_hours: 24

logging:
  level: "info"  # debug/info/warn/error
  format: "json" # json/console
```

### 环境变量覆盖

```bash
# 服务器配置
export SERVER_PORT=8080
export SERVER_MODE=release

# 数据库配置
export DB_HOST=mysql
export DB_PORT=3306
export DB_USER=video_user
export DB_PASSWORD=video_password
export DB_NAME=video_server

# Redis配置
export REDIS_ADDR=redis:6379
export REDIS_PASSWORD=

# 存储配置
export VIDEO_STORAGE_PATH=./storage/videos/
export TEMP_STORAGE_PATH=./storage/temp/

# JWT配置
export JWT_SECRET=your-secret-key
export JWT_EXPIRE_HOURS=24
```

## 服务发现与负载均衡

### Docker网络服务发现

```yaml
# docker-compose.yml中的网络配置
services:
  api-server-1:
    networks:
      - app_network
      
  api-server-2:
    networks:
      - app_network

networks:
  app_network:
    driver: bridge
```

### Nginx反向代理配置

```nginx
# nginx.conf
upstream api_backend {
    server api-server-1:8080;
    server api-server-2:8080;
    server api-server-3:8080;
}

server {
    listen 80;
    server_name api.videoservice.com;
    
    location / {
        proxy_pass http://api_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

## 健康检查

### HTTP健康检查端点

```go
// API Server健康检查
r.GET("/health", func(c *gin.Context) {
    c.JSON(200, gin.H{
        "status": "healthy",
        "service": "api-server",
        "timestamp": time.Now().Unix(),
    })
})

// 详细健康检查
r.GET("/health/detail", func(c *gin.Context) {
    health := map[string]interface{}{
        "database": checkDBConnection(),
        "redis": checkRedisConnection(),
        "disk_space": checkDiskSpace(),
        "memory": checkMemoryUsage(),
    }
    c.JSON(200, health)
})
```

### Docker健康检查配置

```yaml
services:
  api-server:
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 60s
```

## 监控集成

### 应用指标暴露

```go
// 注册Prometheus指标
func registerMetrics() {
    prometheus.MustRegister(httpRequestsTotal)
    prometheus.MustRegister(httpRequestDuration)
    prometheus.MustRegister(activeUsers)
    prometheus.MustRegister(videoUploadsTotal)
}

// 指标中间件
func metricsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()
        
        // 记录指标
        httpRequestsTotal.WithLabelValues(
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
```

### 日志配置

```go
// 结构化日志配置
func setupLogging(config LogConfig) {
    logger, _ := zap.NewProduction()
    defer logger.Sync()
    
    // 添加常用字段
    logger = logger.With(
        zap.String("service", "api-server"),
        zap.String("version", "1.0.0"),
    )
}
```

## 安全配置

### TLS/SSL配置

```go
// HTTPS服务器配置
func startHTTPSServer() {
    cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
    if err != nil {
        log.Fatal(err)
    }
    
    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
        MinVersion:   tls.VersionTLS12,
    }
    
    server := &http.Server{
        Addr:      ":8443",
        TLSConfig: tlsConfig,
        Handler:   router,
    }
    
    log.Fatal(server.ListenAndServeTLS("", ""))
}
```

### CORS配置

```go
// CORS中间件
func corsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Access-Control-Allow-Origin", "*")
        c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
        
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }
        
        c.Next()
    }
}
```

## 备份与恢复

### 应用数据备份

```bash
#!/bin/bash
# backup-app.sh

BACKUP_DIR="/backups/app"
DATE=$(date +%Y%m%d_%H%M%S)

# 创建备份目录
mkdir -p $BACKUP_DIR/$DATE

# 备份配置文件
cp config.yaml $BACKUP_DIR/$DATE/
cp .env $BACKUP_DIR/$DATE/

# 备份存储数据
tar -czf $BACKUP_DIR/$DATE/storage.tar.gz ./storage/

# 备份日志
tar -czf $BACKUP_DIR/$DATE/logs.tar.gz ./logs/

# 清理旧备份
find $BACKUP_DIR -type d -mtime +7 -exec rm -rf {} \;

echo "Backup completed: $BACKUP_DIR/$DATE"
```

### 灾难恢复

```bash
#!/bin/bash
# restore-app.sh

BACKUP_PATH=$1

if [ -z "$BACKUP_PATH" ]; then
    echo "Usage: $0 <backup_path>"
    exit 1
fi

# 停止服务
docker-compose down

# 恢复配置
cp $BACKUP_PATH/config.yaml ./
cp $BACKUP_PATH/.env ./

# 恢复数据
tar -xzf $BACKUP_PATH/storage.tar.gz
tar -xzf $BACKUP_PATH/logs.tar.gz

# 启动服务
docker-compose up -d

echo "Restore completed from $BACKUP_PATH"
```

## 性能调优

### 连接池优化

```go
// 数据库连接池配置
func setupDatabase() *gorm.DB {
    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
    
    sqlDB, _ := db.DB()
    sqlDB.SetMaxIdleConns(10)        // 最大空闲连接
    sqlDB.SetMaxOpenConns(100)       // 最大打开连接
    sqlDB.SetConnMaxLifetime(time.Hour) // 连接最大生存时间
    
    return db
}

// Redis连接池配置
func setupRedis() *redis.Client {
    client := redis.NewClient(&redis.Options{
        Addr:         "localhost:6379",
        PoolSize:     20,              // 连接池大小
        MinIdleConns: 5,               // 最小空闲连接
        PoolTimeout:  30 * time.Second, // 池获取超时
    })
    return client
}
```

### 缓存策略

```go
// 多级缓存实现
type CacheManager struct {
    redis *redis.Client
    local *bigcache.BigCache
}

func (cm *CacheManager) Get(key string) ([]byte, error) {
    // 首先检查本地缓存
    if data, err := cm.local.Get(key); err == nil {
        return data, nil
    }
    
    // 然后检查Redis
    data, err := cm.redis.Get(context.Background(), key).Bytes()
    if err == nil {
        // 异步更新本地缓存
        go cm.local.Set(key, data)
    }
    
    return data, err
}
```

## 故障排除

### 常见问题诊断

```bash
# 检查服务状态
docker-compose ps

# 查看详细日志
docker-compose logs api-server

# 检查资源使用
docker stats

# 网络连接检查
docker-compose exec api-server netstat -tlnp

# 进入容器调试
docker-compose exec api-server sh
```

### 性能问题排查

```bash
# CPU和内存分析
go tool pprof http://localhost:8080/debug/pprof/profile

# 内存分配分析
go tool pprof http://localhost:8080/debug/pprof/heap

# 协程分析
curl http://localhost:8080/debug/pprof/goroutine?debug=2
```

## 下一步

- [API Server详细部署](./api-server/deployment.md)
- [Scheduler任务配置](./scheduler/jobs.md)
- [Worker处理逻辑](./worker/processing.md)
- [前端部署指南](./frontend/deployment.md)
- [基础设施部署](../infrastructure/README.md)