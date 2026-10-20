package config

import "testing"

func validProductionConfig() *Config {
	return &Config{
		Server: ServerConfig{
			CookieSecure:            true,
			VideoModerationRequired: true,
			CORS:                    CORSConfig{AllowedOrigins: []string{"https://video.example.com"}},
			TrustedProxies:          []string{"172.30.0.0/24"},
		},
		Storage: StorageConfig{
			Backend:           "s3",
			S3Endpoint:        "https://media.example.com",
			S3Bucket:          "media",
			S3AccessKey:       "access",
			S3SecretKey:       "secret",
			S3BackupEndpoint:  "https://backup.example.com",
			S3BackupRegion:    "sfo3",
			S3BackupBucket:    "backup",
			S3BackupAccessKey: "backup-access",
			S3BackupSecretKey: "backup-secret",
		},
	}
}

func TestValidateProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	if err := validProductionConfig().ValidateProduction(); err != nil {
		t.Fatalf("valid production config rejected: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{"insecure cookie", func(c *Config) { c.Server.CookieSecure = false }},
		{"moderation disabled", func(c *Config) { c.Server.VideoModerationRequired = false }},
		{"wildcard cors", func(c *Config) { c.Server.CORS.AllowedOrigins = []string{"*"} }},
		{"local storage", func(c *Config) { c.Storage.Backend = "local" }},
		{"missing backup", func(c *Config) { c.Storage.S3BackupBucket = "" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := validProductionConfig()
			test.mutate(cfg)
			if err := cfg.ValidateProduction(); err == nil {
				t.Fatal("expected unsafe production config to be rejected")
			}
		})
	}
}
