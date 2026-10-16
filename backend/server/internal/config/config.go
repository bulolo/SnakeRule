package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Auth     AuthConfig     `mapstructure:"auth"`
	License  LicenseConfig  `mapstructure:"license"`
	Pulse    PulseConfig    `mapstructure:"pulse"`
	Log      LogConfig      `mapstructure:"log"`
	Jev      JevConfig      `mapstructure:"jev"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
	Env  string `mapstructure:"env"`
}

type DatabaseConfig struct {
	URL          string `mapstructure:"url"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

type RedisConfig struct {
	URL string `mapstructure:"url"`
}

type JWTConfig struct {
	Secret    string `mapstructure:"secret"`
	ExpiresIn string `mapstructure:"expires_in"`
}

type AuthConfig struct {
	Email      string        `mapstructure:"email"`
	Name       string        `mapstructure:"name"`
	Password   string        `mapstructure:"password"`
	SessionTTL time.Duration `mapstructure:"session_ttl"`
}

type LicenseConfig struct {
	StorePath     string `mapstructure:"store_path"`
	EncryptionKey string `mapstructure:"encryption_key"`
}

type PulseConfig struct {
	HeartbeatURL            string `mapstructure:"heartbeat_url"`
	HeartbeatIntervalSecond int    `mapstructure:"heartbeat_interval_seconds"`
	HeartbeatTimeoutSecond  int    `mapstructure:"heartbeat_timeout_seconds"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type JevConfig struct {
	APIURL    string `mapstructure:"api_url"`
	APIKey    string `mapstructure:"api_key"`
	Model     string `mapstructure:"default_model"`
	TimeoutMs int    `mapstructure:"default_timeout_ms"`
}

func Load() (*Config, error) {
	for _, path := range []string{".env", "../.env"} {
		if _, err := os.Stat(path); err == nil {
			if err := godotenv.Load(path); err != nil {
				return nil, fmt.Errorf("load %s: %w", path, err)
			}
			break
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("check %s: %w", path, err)
		}
	}
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	for _, key := range []string{
		"server.port", "server.env", "database.url", "database.max_open_conns", "database.max_idle_conns",
		"redis.url", "jwt.secret", "jwt.expires_in", "auth.email", "auth.name", "auth.password", "auth.session_ttl", "license.store_path", "license.encryption_key", "pulse.heartbeat_url", "pulse.heartbeat_interval_seconds", "pulse.heartbeat_timeout_seconds", "log.level", "log.format",
		"jev.api_url", "jev.api_key", "jev.default_model", "jev.default_timeout_ms",
	} {
		if err := viper.BindEnv(key); err != nil {
			return nil, fmt.Errorf("bind config %s: %w", key, err)
		}
	}

	// defaults
	viper.SetDefault("server.port", "8081")
	viper.SetDefault("server.env", "development")
	viper.SetDefault("auth.email", "superadmin@snakerule.cn")
	viper.SetDefault("auth.name", "SuperAdmin")
	viper.SetDefault("auth.password", "")
	viper.SetDefault("auth.session_ttl", "24h")
	viper.SetDefault("license.store_path", "./data/license.json")
	viper.SetDefault("pulse.heartbeat_url", "https://pulse.szplanner.com/api/v1/snakerule/v1/telemetry/heartbeat")
	viper.SetDefault("pulse.heartbeat_interval_seconds", 240)
	viper.SetDefault("pulse.heartbeat_timeout_seconds", 30)
	viper.SetDefault("database.max_open_conns", 20)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")
	viper.SetDefault("jev.default_timeout_ms", 200)

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}
	if cfg.Server.Port == "" {
		return nil, errors.New("SERVER_PORT must not be empty")
	}
	if cfg.Auth.SessionTTL <= 0 {
		return nil, errors.New("AUTH_SESSION_TTL must be positive")
	}
	if strings.TrimSpace(cfg.Database.URL) == "" {
		return nil, errors.New("DATABASE_URL must be configured")
	}
	if strings.TrimSpace(cfg.Auth.Email) == "" || strings.TrimSpace(cfg.Auth.Password) == "" {
		return nil, errors.New("AUTH_EMAIL and AUTH_PASSWORD must be configured")
	}
	if strings.EqualFold(cfg.Server.Env, "production") && cfg.Auth.Password == "change-this-before-run" {
		return nil, errors.New("production requires a non-default AUTH_PASSWORD")
	}
	return cfg, nil
}
