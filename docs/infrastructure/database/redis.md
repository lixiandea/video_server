# Redis 部署和配置指南

## 概述

Redis是视频服务项目的缓存和会话存储系统，主要用于：
- 用户会话管理
- 热点数据缓存
- 限流计数器
- 分布式锁

## 部署方式

### 1. Docker 部署 (推荐)

#### 使用 Docker Compose

```yaml
# docker-compose.yml
version: '3.8'
services:
  redis:
    image: redis:alpine
    container_name: video_server_redis
    restart: always
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
      - ./docker/redis/redis.conf:/usr/local/etc/redis/redis.conf
    command: redis-server /usr/local/etc/redis/redis.conf
    networks:
      - video_network

volumes:
  redis_data:

networks:
  video_network:
    driver: bridge
```

#### Redis 配置文件

创建 `docker/redis/redis.conf`：

```conf
# 基本配置
bind 0.0.0.0
port 6379
timeout 0
tcp-keepalive 300

# 持久化配置
save 900 1
save 300 10
save 60 10000
dbfilename dump.rdb
dir /data

# 内存管理
maxmemory 256mb
maxmemory-policy allkeys-lru

# 安全配置
requirepass your_redis_password
appendonly yes
appendfilename "appendonly.aof"
appendfsync everysec

# 日志配置
loglevel notice
logfile ""

# 网络配置
tcp-backlog 511
```

#### 环境变量配置

```bash
# .env
REDIS_PASSWORD=your_redis_password
REDIS_MAXMEMORY=256mb
```

#### 启动服务

```bash
# 启动Redis服务
docker-compose up -d redis

# 查看服务状态
docker-compose ps redis

# 查看日志
docker-compose logs -f redis
```

### 2. 独立安装

#### macOS

```bash
# 使用Homebrew安装
brew install redis

# 启动Redis服务
brew services start redis

# 连接测试
redis-cli ping
```

#### Ubuntu/Debian

```bash
# 更新包列表
sudo apt update

# 安装Redis
sudo apt install redis-server

# 启动服务
sudo systemctl start redis-server
sudo systemctl enable redis-server

# 配置Redis
sudo nano /etc/redis/redis.conf
```

#### Windows

从 [Redis官网](https://redis.io/download) 下载Windows版本或使用WSL安装。

## 配置详解

### 1. 内存配置

```conf
# 最大内存限制
maxmemory 256mb

# 内存淘汰策略
maxmemory-policy allkeys-lru
# 可选策略：
# - allkeys-lru: 对所有key使用LRU算法
# - volatile-lru: 对设置了过期时间的key使用LRU
# - allkeys-random: 随机删除key
# - volatile-random: 随机删除设置了过期时间的key
# - volatile-ttl: 删除最近要过期的key
```

### 2. 持久化配置

```conf
# RDB快照
save 900 1      # 900秒内至少1个key改变则保存
save 300 10     # 300秒内至少10个key改变则保存
save 60 10000   # 60秒内至少10000个key改变则保存

# AOF持久化
appendonly yes
appendfilename "appendonly.aof"
appendfsync everysec  # 每秒同步一次
```

### 3. 安全配置

```conf
# 设置密码
requirepass your_strong_password

# 重命名危险命令
rename-command FLUSHDB ""
rename-command FLUSHALL ""
rename-command KEYS ""
rename-command CONFIG ""
```

## 应用集成

### Go应用配置

```go
package main

import (
    "context"
    "github.com/go-redis/redis/v8"
    "time"
)

type RedisClient struct {
    client *redis.Client
}

func NewRedisClient(addr, password string, db int) *RedisClient {
    client := redis.NewClient(&redis.Options{
        Addr:     addr,
        Password: password,
        DB:       db,
        PoolSize: 20,
        MinIdleConns: 5,
        DialTimeout:  5 * time.Second,
        ReadTimeout:  3 * time.Second,
        WriteTimeout: 3 * time.Second,
    })

    return &RedisClient{client: client}
}

// Session管理示例
func (r *RedisClient) SetSession(sessionID string, userID int, expire time.Duration) error {
    ctx := context.Background()
    return r.client.Set(ctx, sessionID, userID, expire).Err()
}

func (r *RedisClient) GetSession(sessionID string) (int, error) {
    ctx := context.Background()
    val, err := r.client.Get(ctx, sessionID).Result()
    if err != nil {
        return 0, err
    }
    
    var userID int
    _, err = fmt.Sscanf(val, "%d", &userID)
    return userID, err
}

// 缓存操作示例
func (r *RedisClient) CacheVideoInfo(videoID string, data []byte, expire time.Duration) error {
    ctx := context.Background()
    return r.client.Set(ctx, "video:"+videoID, data, expire).Err()
}

func (r *RedisClient) GetCachedVideoInfo(videoID string) ([]byte, error) {
    ctx := context.Background()
    return r.client.Get(ctx, "video:"+videoID).Bytes()
}

// 限流实现
func (r *RedisClient) RateLimit(key string, limit int64, window time.Duration) (bool, error) {
    ctx := context.Background()
    
    // 使用滑动窗口算法
    now := time.Now().Unix()
    pipeline := r.client.TxPipeline()
    
    // 移除过期的记录
    pipeline.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(now-int64(window.Seconds()), 10))
    
    // 添加当前请求
    pipeline.ZAdd(ctx, key, &redis.Z{
        Score:  float64(now),
        Member: now,
    })
    
    // 设置过期时间
    pipeline.Expire(ctx, key, window)
    
    // 获取当前窗口内的请求数
    pipeline.ZCard(ctx, key)
    
    cmds, err := pipeline.Exec(ctx)
    if err != nil {
        return false, err
    }
    
    count := cmds[len(cmds)-1].(*redis.IntCmd).Val()
    return count <= limit, nil
}
```

### 配置文件集成

```yaml
# config.yaml
redis:
  addr: "localhost:6379"  # Docker环境下使用 "redis:6379"
  password: "your_redis_password"
  db: 0
  pool_size: 20
  min_idle_conns: 5
```

## 数据结构使用场景

### 1. String - 基础键值对

```go
// 存储简单的键值对
client.Set(ctx, "user:1:name", "张三", 0)
name, _ := client.Get(ctx, "user:1:name").Result()
```

### 2. Hash - 对象存储

```go
// 存储用户信息
client.HSet(ctx, "user:1", map[string]interface{}{
    "name": "张三",
    "email": "zhangsan@example.com",
    "avatar": "avatar.jpg",
})

// 获取用户信息
userInfo, _ := client.HGetAll(ctx, "user:1").Result()
```

### 3. List - 消息队列

```go
// 生产消息
client.LPush(ctx, "video_queue", videoData)

// 消费消息
message, _ := client.BRPop(ctx, 30*time.Second, "video_queue").Result()
```

### 4. Set - 标签系统

```go
// 添加视频标签
client.SAdd(ctx, "video:1:tags", "科技", "教育", "编程")

// 获取视频标签
tags, _ := client.SMembers(ctx, "video:1:tags").Result()

// 查找同时包含多个标签的视频
client.SInter(ctx, "tag:科技:videos", "tag:教育:videos")
```

### 5. Sorted Set - 排行榜

```go
// 添加视频观看次数
client.ZAdd(ctx, "video_views", &redis.Z{
    Score:  1000,
    Member: "video_1",
})

// 获取热门视频排行
topVideos, _ := client.ZRevRangeWithScores(ctx, "video_views", 0, 9).Result()
```

## 性能优化

### 1. 连接池配置

```go
redisOptions := &redis.Options{
    Addr:         "localhost:6379",
    Password:     "password",
    DB:           0,
    PoolSize:     100,        // 连接池大小
    MinIdleConns: 10,         // 最小空闲连接
    MaxConnAge:   time.Hour,  // 连接最大存活时间
    PoolTimeout:  30 * time.Second, // 获取连接超时时间
    IdleTimeout:  10 * time.Minute, // 空闲连接超时时间
}
```

### 2. Pipeline批处理

```go
// 批量操作提高性能
pipe := client.Pipeline()
pipe.Set(ctx, "key1", "value1", 0)
pipe.Set(ctx, "key2", "value2", 0)
pipe.Set(ctx, "key3", "value3", 0)
_, err := pipe.Exec(ctx)
```

### 3. Lua脚本原子操作

```go
// 原子性的计数器增加
script := `
local current = redis.call('GET', KEYS[1])
if current == false then
    current = 0
end
current = tonumber(current) + tonumber(ARGV[1])
redis.call('SET', KEYS[1], current)
return current
`

result, err := client.Eval(ctx, script, []string{"counter"}, 1).Result()
```

## 监控与维护

### 1. 健康检查

```bash
# 基本连通性检查
redis-cli ping

# 详细信息检查
redis-cli info

# 检查特定部分信息
redis-cli info memory
redis-cli info cpu
redis-cli info clients
```

### 2. 性能监控

```bash
# 实时监控
redis-cli --stat

# 监控慢查询
redis-cli slowlog get 10

# 查看客户端连接
redis-cli client list
```

### 3. 内存使用监控

```go
// Go应用中监控Redis内存使用
func (r *RedisClient) GetMemoryInfo() (map[string]string, error) {
    ctx := context.Background()
    info, err := r.client.Info(ctx, "memory").Result()
    if err != nil {
        return nil, err
    }
    
    // 解析info字符串
    result := make(map[string]string)
    lines := strings.Split(info, "\n")
    for _, line := range lines {
        if strings.Contains(line, ":") {
            parts := strings.SplitN(line, ":", 2)
            if len(parts) == 2 {
                key := strings.TrimSpace(parts[0])
                value := strings.TrimSpace(parts[1])
                result[key] = value
            }
        }
    }
    
    return result, nil
}
```

### 4. 数据备份

```bash
# RDB备份
redis-cli BGSAVE

# AOF重写
redis-cli BGREWRITEAOF

# 手动复制数据文件
cp /var/lib/redis/dump.rdb /backup/location/
```

## 安全加固

### 1. 网络安全

```conf
# 只绑定内网IP
bind 127.0.0.1 192.168.1.100

# 启用保护模式
protected-mode yes
```

### 2. 访问控制

```conf
# 设置强密码
requirepass VeryStrongPassword123!

# 限制客户端连接数
maxclients 10000
```

### 3. 命令重命名

```conf
# 禁用危险命令
rename-command FLUSHDB ""
rename-command FLUSHALL ""
rename-command KEYS ""
rename-command CONFIG ""
rename-command SHUTDOWN ""
rename-command DEBUG ""
```

## 故障排除

### 常见问题及解决方案

1. **连接超时**
   ```bash
   # 检查Redis服务状态
   docker-compose ps redis
   
   # 检查端口监听
   netstat -tlnp | grep 6379
   
   # 检查防火墙
   iptables -L
   ```

2. **内存不足**
   ```bash
   # 查看内存使用情况
   redis-cli info memory
   
   # 调整内存策略
   redis-cli CONFIG SET maxmemory-policy allkeys-lru
   
   # 清理过期key
   redis-cli EVAL "return redis.call('EXPIRE', KEYS[1], ARGV[1])" 1 somekey 3600
   ```

3. **性能问题**
   ```bash
   # 查看慢查询
   redis-cli SLOWLOG GET 10
   
   # 分析key分布
   redis-cli --bigkeys
   
   # 查看热点key
   redis-cli --hotkeys
   ```

4. **数据一致性**
   ```bash
   # 检查AOF状态
   redis-cli info persistence
   
   # 强制同步到磁盘
   redis-cli BGSAVE
   
   # 检查主从同步状态
   redis-cli info replication
   ```

## 高可用部署

### 1. 主从复制

```yaml
# master节点
redis-master:
  image: redis:alpine
  command: redis-server --appendonly yes

# slave节点
redis-slave:
  image: redis:alpine
  command: redis-server --slaveof redis-master 6379 --appendonly yes
  depends_on:
    - redis-master
```

### 2. Sentinel哨兵模式

```yaml
# sentinel配置
redis-sentinel:
  image: redis:alpine
  command: >
    bash -c "
      echo 'port 26379' > /etc/redis/sentinel.conf &&
      echo 'sentinel monitor mymaster redis-master 6379 2' >> /etc/redis/sentinel.conf &&
      echo 'sentinel down-after-milliseconds mymaster 5000' >> /etc/redis/sentinel.conf &&
      redis-sentinel /etc/redis/sentinel.conf
    "
```

### 3. Cluster集群模式

```bash
# 创建Redis集群
redis-cli --cluster create \
  127.0.0.1:7000 127.0.0.1:7001 127.0.0.1:7002 \
  127.0.0.1:7003 127.0.0.1:7004 127.0.0.1:7005 \
  --cluster-replicas 1
```

## 参考资源

- [Redis官方文档](https://redis.io/documentation)
- [Redis命令参考](https://redis.io/commands)
- [Go Redis客户端文档](https://github.com/go-redis/redis)
- [Redis性能调优指南](https://redis.io/topics/latency)