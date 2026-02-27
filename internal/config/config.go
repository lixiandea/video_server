package config

import (
    "github.com/spf13/viper"
    "log"
    "strings"
)

type Config struct {
    Server   ServerConfig   `mapstructure:"server"`
    Database DatabaseConfig `mapstructure:"database"`
    Redis    RedisConfig    `mapstructure:"redis"`
    Storage  StorageConfig  `mapstructure:"storage"`
}

type ServerConfig struct {
    Port         string `mapstructure:"port"`
    Mode         string `mapstructure:"mode"`
    ReadTimeout  int    `mapstructure:"read_timeout"`  // seconds
    WriteTimeout int    `mapstructure:"write_timeout"` // seconds
    MaxFileSize  int64  `mapstructure:"max_file_size"` // bytes
    RateLimit    RateLimitConfig `mapstructure:"rate_limit"`
}

type RateLimitConfig struct {
    Enabled bool    `mapstructure:"enabled"`
    Rate    float64 `mapstructure:"rate"`  // requests per second
    Burst   int     `mapstructure:"burst"` // max burst size
}

type DatabaseConfig struct {
    Host     string `mapstructure:"host"`
    Port     int    `mapstructure:"port"`
    User     string `mapstructure:"user"`
    Password string `mapstructure:"password"`
    Name     string `mapstructure:"name"`
    Charset  string `mapstructure:"charset"`
}

type RedisConfig struct {
    Addr         string `mapstructure:"addr"`
    Password     string `mapstructure:"password"`
    DB           int    `mapstructure:"db"`
    PoolSize     int    `mapstructure:"pool_size"`
    MinIdleConns int    `mapstructure:"min_idle_conns"`
    DialTimeout  int    `mapstructure:"dial_timeout"`  // seconds
    ReadTimeout  int    `mapstructure:"read_timeout"`  // seconds
    WriteTimeout int    `mapstructure:"write_timeout"` // seconds
}

type StorageConfig struct {
    VideoDir    string `mapstructure:"video_dir"`
    TemplateDir string `mapstructure:"template_dir"`
    TempDir     string `mapstructure:"temp_dir"`
}

var AppConfig *Config

func LoadConfig() *Config {
    viper.SetDefault("server.port", "8080")
    viper.SetDefault("server.mode", "debug")
    viper.SetDefault("server.read_timeout", 30)
    viper.SetDefault("server.write_timeout", 30)
    viper.SetDefault("server.max_file_size", int64(50*1024*1024)) // 50MB
    viper.SetDefault("server.rate_limit.enabled", true)
    viper.SetDefault("server.rate_limit.rate", 10.0)
    viper.SetDefault("server.rate_limit.burst", 20)

    viper.SetDefault("database.host", "localhost")
    viper.SetDefault("database.port", 3306)
    viper.SetDefault("database.user", "root")
    viper.SetDefault("database.password", "password")
    viper.SetDefault("database.name", "video_server")
    viper.SetDefault("database.charset", "utf8mb4")

    viper.SetDefault("redis.addr", "localhost:6379")
    viper.SetDefault("redis.password", "")
    viper.SetDefault("redis.db", 0)
    viper.SetDefault("redis.pool_size", 20)
    viper.SetDefault("redis.min_idle_conns", 5)
    viper.SetDefault("redis.dial_timeout", 5)
    viper.SetDefault("redis.read_timeout", 3)
    viper.SetDefault("redis.write_timeout", 3)

    viper.SetDefault("storage.video_dir", "./storage/videos/")
    viper.SetDefault("storage.template_dir", "./templates/")
    viper.SetDefault("storage.temp_dir", "./storage/temp/")

    // Enable environment variable binding
    viper.AutomaticEnv()
    viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

    // Bind environment variables for database
    viper.BindEnv("database.host", "DB_HOST")
    viper.BindEnv("database.port", "DB_PORT")
    viper.BindEnv("database.user", "DB_USER")
    viper.BindEnv("database.password", "DB_PASSWORD")
    viper.BindEnv("database.name", "DB_NAME")

    // Bind environment variables for redis
    viper.BindEnv("redis.addr", "REDIS_ADDR")
    viper.BindEnv("redis.password", "REDIS_PASSWORD")

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

    AppConfig = &config
    return &config
}