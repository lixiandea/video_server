package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/lixiandea/video_server/pkg/logging"
	"go.uber.org/zap"
)

var Client *redis.Client

// Config Redis 配置
type Config struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// InitRedis 初始化 Redis 客户端
func InitRedis(cfg *Config) error {
	Client = redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := Client.Ping(ctx).Result()
	if err != nil {
		logging.GetLogger().Error("Failed to connect to Redis", zap.Error(err))
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logging.GetLogger().Info("Redis initialized successfully",
		zap.String("addr", cfg.Addr),
		zap.Int("pool_size", cfg.PoolSize))

	return nil
}

// GetClient 返回 Redis 客户端实例
func GetClient() *redis.Client {
	if Client == nil {
		logging.GetLogger().Warn("Redis client not initialized")
		return nil
	}
	return Client
}

// CacheManager 缓存管理器
type CacheManager struct {
	client *redis.Client
	logger *zap.Logger
}

// NewCacheManager 创建缓存管理器
func NewCacheManager(client *redis.Client) *CacheManager {
	return &CacheManager{
		client: client,
		logger: logging.GetLogger(),
	}
}

// Get 从缓存获取数据
func (cm *CacheManager) Get(ctx context.Context, key string, dest interface{}) error {
	if cm.client == nil {
		return fmt.Errorf("redis client not available")
	}

	data, err := cm.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil // 缓存未命中
		}
		cm.logger.Warn("Failed to get from cache",
			zap.String("key", key),
			zap.Error(err))
		return err
	}

	if err := json.Unmarshal(data, dest); err != nil {
		cm.logger.Warn("Failed to unmarshal cache data",
			zap.String("key", key),
			zap.Error(err))
		return err
	}

	return nil
}

// Set 设置缓存数据
func (cm *CacheManager) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if cm.client == nil {
		return fmt.Errorf("redis client not available")
	}

	data, err := json.Marshal(value)
	if err != nil {
		cm.logger.Warn("Failed to marshal value",
			zap.String("key", key),
			zap.Error(err))
		return err
	}

	if err := cm.client.Set(ctx, key, data, expiration).Err(); err != nil {
		cm.logger.Warn("Failed to set cache",
			zap.String("key", key),
			zap.Error(err))
		return err
	}

	return nil
}

// Delete 删除缓存
func (cm *CacheManager) Delete(ctx context.Context, keys ...string) error {
	if cm.client == nil {
		return fmt.Errorf("redis client not available")
	}

	return cm.client.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
func (cm *CacheManager) Exists(ctx context.Context, key string) (bool, error) {
	if cm.client == nil {
		return false, fmt.Errorf("redis client not available")
	}

	result, err := cm.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}

	return result > 0, nil
}

// GetOrSet 获取或设置缓存（带回调）
func (cm *CacheManager) GetOrSet(ctx context.Context, key string, dest interface{}, expiration time.Duration, fetchFunc func() (interface{}, error)) error {
	// 尝试从缓存获取
	if err := cm.Get(ctx, key, dest); err == nil && dest != nil {
		cm.logger.Debug("Cache hit", zap.String("key", key))
		return nil
	}

	// 缓存未命中，执行获取函数
	data, err := fetchFunc()
	if err != nil {
		return err
	}

	// 设置缓存
	if err := cm.Set(ctx, key, data, expiration); err != nil {
		cm.logger.Warn("Failed to set cache after fetch",
			zap.String("key", key),
			zap.Error(err))
	}

	// 将数据复制到 dest
	if dest != nil {
		jsonData, _ := json.Marshal(data)
		json.Unmarshal(jsonData, dest)
	}

	return nil
}

// Close 关闭 Redis 连接
func Close() error {
	if Client != nil {
		return Client.Close()
	}
	return nil
}
