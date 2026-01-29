package config

import (
    "github.com/spf13/viper"
    "log"
)

type Config struct {
    Server   ServerConfig   `mapstructure:"server"`
    Database DatabaseConfig `mapstructure:"database"`
    Storage  StorageConfig  `mapstructure:"storage"`
}

type ServerConfig struct {
    Port         string `mapstructure:"port"`
    Mode         string `mapstructure:"mode"`
    ReadTimeout  int    `mapstructure:"read_timeout"`  // seconds
    WriteTimeout int    `mapstructure:"write_timeout"` // seconds
    MaxFileSize  int64  `mapstructure:"max_file_size"` // bytes
}

type DatabaseConfig struct {
    Host     string `mapstructure:"host"`
    Port     int    `mapstructure:"port"`
    User     string `mapstructure:"user"`
    Password string `mapstructure:"password"`
    Name     string `mapstructure:"name"`
    Charset  string `mapstructure:"charset"`
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
    
    viper.SetDefault("database.host", "localhost")
    viper.SetDefault("database.port", 3306)
    viper.SetDefault("database.user", "root")
    viper.SetDefault("database.password", "password")
    viper.SetDefault("database.name", "video_server")
    viper.SetDefault("database.charset", "utf8mb4")
    
    viper.SetDefault("storage.video_dir", "./storage/videos/")
    viper.SetDefault("storage.template_dir", "./templates/")
    viper.SetDefault("storage.temp_dir", "./storage/temp/")

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