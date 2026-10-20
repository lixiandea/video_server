package config

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
	"os"
	"strings"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Storage  StorageConfig  `mapstructure:"storage"`
}

type ServerConfig struct {
	Port                    string          `mapstructure:"port"`
	Mode                    string          `mapstructure:"mode"`
	ReadTimeout             int             `mapstructure:"read_timeout"`  // seconds
	WriteTimeout            int             `mapstructure:"write_timeout"` // seconds
	MaxFileSize             int64           `mapstructure:"max_file_size"` // bytes
	JWTSecret               string          `mapstructure:"jwt_secret"`    // from JWT_SECRET env
	AccessTokenTTL          int             `mapstructure:"access_token_ttl"`
	RefreshTokenTTL         int             `mapstructure:"refresh_token_ttl"`
	CookieSecure            bool            `mapstructure:"cookie_secure"`
	CookieDomain            string          `mapstructure:"cookie_domain"`
	StreamTokenTTL          int             `mapstructure:"stream_token_ttl"` // seconds, playback token lifetime
	MetricsToken            string          `mapstructure:"metrics_token"`    // 可选，/metrics 的 Bearer 令牌
	VideoModerationRequired bool            `mapstructure:"video_moderation_required"`
	UserVideoQuota          int64           `mapstructure:"user_video_quota"`
	UserStorageQuotaBytes   int64           `mapstructure:"user_storage_quota_bytes"`
	UserDailyUploadLimit    int             `mapstructure:"user_daily_upload_limit"`
	TrustedProxies          []string        `mapstructure:"trusted_proxies"`
	RateLimit               RateLimitConfig `mapstructure:"rate_limit"`
	CORS                    CORSConfig      `mapstructure:"cors"`
}

type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

type RateLimitConfig struct {
	Enabled bool    `mapstructure:"enabled"`
	Rate    float64 `mapstructure:"rate"`  // requests per second
	Burst   int     `mapstructure:"burst"` // max burst size
}

type DatabaseConfig struct {
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	User      string `mapstructure:"user"`
	Password  string `mapstructure:"password"`
	Name      string `mapstructure:"name"`
	Charset   string `mapstructure:"charset"`
	TLSCAPath string `mapstructure:"tls_ca_path"`
	LogLevel  string `mapstructure:"log_level"`
	MaxOpenConns int `mapstructure:"max_open_conns"`
	MaxIdleConns int `mapstructure:"max_idle_conns"`
	// AutoMigrate 控制服务启动时是否执行 GORM AutoMigrate。
	// 默认关闭：结构变更统一走 `migrate up`（见 docs/database-migration-strategy.md）；
	// 仅本地/CI 追求「一条命令起库」时才开启，且 release 模式会强制忽略。
	AutoMigrate bool `mapstructure:"auto_migrate"`
}

type RedisConfig struct {
	Addr         string `mapstructure:"addr"`
	Password     string `mapstructure:"password"`
	TLSEnabled   bool   `mapstructure:"tls_enabled"`
	TLSCAPath    string `mapstructure:"tls_ca_path"`
	DB           int    `mapstructure:"db"`
	PoolSize     int    `mapstructure:"pool_size"`
	MinIdleConns int    `mapstructure:"min_idle_conns"`
	DialTimeout  int    `mapstructure:"dial_timeout"`  // seconds
	ReadTimeout  int    `mapstructure:"read_timeout"`  // seconds
	WriteTimeout int    `mapstructure:"write_timeout"` // seconds
}

type StorageConfig struct {
	Backend           string `mapstructure:"backend"` // local or s3
	VideoDir          string `mapstructure:"video_dir"`
	TemplateDir       string `mapstructure:"template_dir"`
	TempDir           string `mapstructure:"temp_dir"`
	S3Endpoint        string `mapstructure:"s3_endpoint"`
	S3Region          string `mapstructure:"s3_region"`
	S3Bucket          string `mapstructure:"s3_bucket"`
	S3AccessKey       string `mapstructure:"s3_access_key"`
	S3SecretKey       string `mapstructure:"s3_secret_key"`
	S3UseSSL          bool   `mapstructure:"s3_use_ssl"`
	S3BackupEndpoint  string `mapstructure:"s3_backup_endpoint"`
	S3BackupRegion    string `mapstructure:"s3_backup_region"`
	S3BackupBucket    string `mapstructure:"s3_backup_bucket"`
	S3BackupAccessKey string `mapstructure:"s3_backup_access_key"`
	S3BackupSecretKey string `mapstructure:"s3_backup_secret_key"`
}

var AppConfig *Config

func LoadConfig() *Config {
	LoadSecretFiles()

	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("server.read_timeout", 30)
	viper.SetDefault("server.write_timeout", 30)
	viper.SetDefault("server.max_file_size", int64(50*1024*1024)) // 50MB
	viper.SetDefault("server.rate_limit.enabled", true)
	viper.SetDefault("server.rate_limit.rate", 10.0)
	viper.SetDefault("server.rate_limit.burst", 20)
	viper.SetDefault("server.jwt_secret", "")
	viper.SetDefault("server.access_token_ttl", 900)
	viper.SetDefault("server.refresh_token_ttl", 30*24*60*60)
	viper.SetDefault("server.cookie_secure", false)
	viper.SetDefault("server.cookie_domain", "")
	viper.SetDefault("server.metrics_token", "")
	viper.SetDefault("server.stream_token_ttl", 7200) // 2h
	viper.SetDefault("server.video_moderation_required", false)
	viper.SetDefault("server.user_video_quota", int64(100))
	viper.SetDefault("server.user_storage_quota_bytes", int64(10*1024*1024*1024))
	viper.SetDefault("server.user_daily_upload_limit", 20)
	viper.SetDefault("server.trusted_proxies", []string{"127.0.0.1", "::1"})
	viper.SetDefault("server.cors.allowed_origins", []string{"http://localhost:3000", "http://localhost:8000"})

	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 3306)
	viper.SetDefault("database.user", "root")
	viper.SetDefault("database.password", "")
	viper.SetDefault("database.name", "video_server")
	viper.SetDefault("database.charset", "utf8mb4")
	viper.SetDefault("database.tls_ca_path", "")
	viper.SetDefault("database.log_level", "warn")
	viper.SetDefault("database.max_open_conns", 100)
	viper.SetDefault("database.max_idle_conns", 10)
	viper.SetDefault("database.auto_migrate", false)

	viper.SetDefault("redis.addr", "localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.tls_enabled", false)
	viper.SetDefault("redis.tls_ca_path", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.pool_size", 20)
	viper.SetDefault("redis.min_idle_conns", 5)
	viper.SetDefault("redis.dial_timeout", 5)
	viper.SetDefault("redis.read_timeout", 3)
	viper.SetDefault("redis.write_timeout", 3)

	viper.SetDefault("storage.backend", "local")
	viper.SetDefault("storage.video_dir", "./storage/videos/")
	viper.SetDefault("storage.template_dir", "./templates/")
	viper.SetDefault("storage.temp_dir", "./storage/temp/")
	viper.SetDefault("storage.s3_endpoint", "")
	viper.SetDefault("storage.s3_region", "")
	viper.SetDefault("storage.s3_bucket", "")
	viper.SetDefault("storage.s3_access_key", "")
	viper.SetDefault("storage.s3_secret_key", "")
	viper.SetDefault("storage.s3_use_ssl", true)
	viper.SetDefault("storage.s3_backup_endpoint", "")
	viper.SetDefault("storage.s3_backup_region", "")
	viper.SetDefault("storage.s3_backup_bucket", "")
	viper.SetDefault("storage.s3_backup_access_key", "")
	viper.SetDefault("storage.s3_backup_secret_key", "")

	// Enable environment variable binding
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Bind environment variables for database
	viper.BindEnv("database.host", "DB_HOST")
	viper.BindEnv("database.port", "DB_PORT")
	viper.BindEnv("database.user", "DB_USER")
	viper.BindEnv("database.password", "DB_PASSWORD")
	viper.BindEnv("database.name", "DB_NAME")
	viper.BindEnv("database.tls_ca_path", "DB_TLS_CA_PATH")
	viper.BindEnv("database.log_level", "DB_LOG_LEVEL")
	viper.BindEnv("database.max_open_conns", "DB_MAX_OPEN_CONNS")
	viper.BindEnv("database.max_idle_conns", "DB_MAX_IDLE_CONNS")
	viper.BindEnv("database.auto_migrate", "DB_AUTO_MIGRATE")

	// Bind environment variables for redis
	viper.BindEnv("redis.addr", "REDIS_ADDR")
	viper.BindEnv("redis.password", "REDIS_PASSWORD")
	viper.BindEnv("redis.tls_enabled", "REDIS_TLS_ENABLED")
	viper.BindEnv("redis.tls_ca_path", "REDIS_TLS_CA_PATH")

	// Bind JWT secret: no default, so a missing value fails fast at startup
	viper.BindEnv("server.mode", "SERVER_MODE", "GIN_MODE")
	viper.BindEnv("server.jwt_secret", "JWT_SECRET")
	viper.BindEnv("server.access_token_ttl", "ACCESS_TOKEN_TTL")
	viper.BindEnv("server.refresh_token_ttl", "REFRESH_TOKEN_TTL")
	viper.BindEnv("server.cookie_secure", "AUTH_COOKIE_SECURE")
	viper.BindEnv("server.cookie_domain", "AUTH_COOKIE_DOMAIN")
	viper.BindEnv("server.metrics_token", "METRICS_TOKEN")
	viper.BindEnv("server.stream_token_ttl", "STREAM_TOKEN_TTL")
	viper.BindEnv("server.video_moderation_required", "VIDEO_MODERATION_REQUIRED")
	viper.BindEnv("server.user_video_quota", "USER_VIDEO_QUOTA")
	viper.BindEnv("server.user_storage_quota_bytes", "USER_STORAGE_QUOTA_BYTES")
	viper.BindEnv("server.user_daily_upload_limit", "USER_DAILY_UPLOAD_LIMIT")

	// Bind storage directories so containers can mount them explicitly
	viper.BindEnv("storage.backend", "STORAGE_BACKEND")
	viper.BindEnv("storage.video_dir", "STORAGE_VIDEO_DIR")
	viper.BindEnv("storage.temp_dir", "STORAGE_TEMP_DIR")
	viper.BindEnv("storage.s3_endpoint", "S3_ENDPOINT")
	viper.BindEnv("storage.s3_region", "S3_REGION")
	viper.BindEnv("storage.s3_bucket", "S3_BUCKET")
	viper.BindEnv("storage.s3_access_key", "S3_ACCESS_KEY")
	viper.BindEnv("storage.s3_secret_key", "S3_SECRET_KEY")
	viper.BindEnv("storage.s3_use_ssl", "S3_USE_SSL")
	viper.BindEnv("storage.s3_backup_endpoint", "S3_BACKUP_ENDPOINT")
	viper.BindEnv("storage.s3_backup_region", "S3_BACKUP_REGION")
	viper.BindEnv("storage.s3_backup_bucket", "S3_BACKUP_BUCKET")
	viper.BindEnv("storage.s3_backup_access_key", "S3_BACKUP_ACCESS_KEY")
	viper.BindEnv("storage.s3_backup_secret_key", "S3_BACKUP_SECRET_KEY")

	// Allow containers to override the CORS allowlist with a comma-separated list
	if origins := os.Getenv("CORS_ALLOWED_ORIGINS"); origins != "" {
		viper.Set("server.cors.allowed_origins", strings.Split(origins, ","))
	}
	if proxies := os.Getenv("TRUSTED_PROXIES"); proxies != "" {
		viper.Set("server.trusted_proxies", strings.Split(proxies, ","))
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Config file not found, using defaults: %v", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("Unable to decode config into struct: %v", err)
	}

	// release 模式一律不允许启动时改表：结构变更必须走显式的 migrate up
	if config.Server.Mode == "release" && config.Database.AutoMigrate {
		log.Println("Warning: DB_AUTO_MIGRATE is ignored in release mode; run `migrate up` instead")
		config.Database.AutoMigrate = false
	}

	AppConfig = &config
	return &config
}

// loadSecretFiles 支持 Docker Secrets：FOO_FILE 指向文件时读取并注入 FOO。
func LoadSecretFiles() {
	for _, name := range []string{
		"DB_PASSWORD",
		"REDIS_PASSWORD",
		"JWT_SECRET",
		"METRICS_TOKEN",
		"S3_ACCESS_KEY",
		"S3_SECRET_KEY",
		"S3_BACKUP_ACCESS_KEY",
		"S3_BACKUP_SECRET_KEY",
	} {
		path := os.Getenv(name + "_FILE")
		if path == "" {
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("Unable to read %s_FILE: %v", name, err)
		}
		_ = os.Setenv(name, strings.TrimSpace(string(content)))
	}
}

// ValidateProduction 在 APP_ENV=production 时拒绝明显不安全的配置启动。
func (c *Config) ValidateProduction() error {
	if os.Getenv("APP_ENV") != "production" {
		return nil
	}
	if !c.Server.CookieSecure {
		return fmt.Errorf("AUTH_COOKIE_SECURE must be true in production")
	}
	if !c.Server.VideoModerationRequired {
		return fmt.Errorf("VIDEO_MODERATION_REQUIRED must be true in production")
	}
	for _, origin := range c.Server.CORS.AllowedOrigins {
		if strings.TrimSpace(origin) == "*" {
			return fmt.Errorf("wildcard CORS origin is not allowed in production")
		}
	}
	for _, proxy := range c.Server.TrustedProxies {
		if strings.TrimSpace(proxy) == "0.0.0.0/0" || strings.TrimSpace(proxy) == "::/0" {
			return fmt.Errorf("untrusted wildcard proxy is not allowed in production")
		}
	}
	if !strings.EqualFold(c.Storage.Backend, "s3") {
		return fmt.Errorf("production requires S3-compatible object storage")
	}
	if c.Storage.S3Endpoint == "" || c.Storage.S3Bucket == "" ||
		c.Storage.S3AccessKey == "" || c.Storage.S3SecretKey == "" {
		return fmt.Errorf("production S3 storage configuration is incomplete")
	}
	if c.Storage.S3BackupEndpoint == "" || c.Storage.S3BackupRegion == "" ||
		c.Storage.S3BackupBucket == "" || c.Storage.S3BackupAccessKey == "" || c.Storage.S3BackupSecretKey == "" {
		return fmt.Errorf("production backup object storage is required")
	}
	return nil
}
