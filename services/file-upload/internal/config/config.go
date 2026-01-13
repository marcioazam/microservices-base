package config

import (
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the service
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Storage  StorageConfig  `mapstructure:"storage"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Upload   UploadConfig   `mapstructure:"upload"`
	Scanner  ScannerConfig  `mapstructure:"scanner"`
	Metrics  MetricsConfig  `mapstructure:"metrics"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// StorageConfig holds cloud storage configuration
type StorageConfig struct {
	Provider        string `mapstructure:"provider"` // s3 or gcs
	Bucket          string `mapstructure:"bucket"`
	Region          string `mapstructure:"region"`
	Endpoint        string `mapstructure:"endpoint"` // for localstack/minio
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	UsePathStyle    bool   `mapstructure:"use_path_style"`
	PublicAccess    bool   `mapstructure:"public_access"`
	SignedURLExpiry time.Duration `mapstructure:"signed_url_expiry"`
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWKSUrl       string        `mapstructure:"jwks_url"`
	Issuer        string        `mapstructure:"issuer"`
	Audience      string        `mapstructure:"audience"`
	CacheDuration time.Duration `mapstructure:"cache_duration"`
}

// UploadConfig holds upload-related configuration
type UploadConfig struct {
	MaxFileSize       int64         `mapstructure:"max_file_size"`        // bytes
	MaxChunkedSize    int64         `mapstructure:"max_chunked_size"`     // bytes
	ChunkSize         int64         `mapstructure:"chunk_size"`           // bytes
	ChunkThreshold    int64         `mapstructure:"chunk_threshold"`      // bytes
	SessionExpiry     time.Duration `mapstructure:"session_expiry"`
	AllowedMIMETypes  []string      `mapstructure:"allowed_mime_types"`
	RateLimitPerMin   int           `mapstructure:"rate_limit_per_min"`
	WorkerPoolSize    int           `mapstructure:"worker_pool_size"`
	RetentionDays     int           `mapstructure:"retention_days"`
}

// ScannerConfig holds malware scanner configuration
type ScannerConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Timeout  time.Duration `mapstructure:"timeout"`
}

// MetricsConfig holds observability configuration
type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
}

// Load loads configuration from file and environment
func Load(configPath string) (*Config, error) {
	v := viper.New()
	
	// Set defaults
	setDefaults(v)
	
	// Config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		v.AddConfigPath("/etc/file-upload")
	}
	
	// Environment variables
	v.SetEnvPrefix("FILE_UPLOAD")
	v.AutomaticEnv()
	
	// Read config file (optional)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}
	
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	
	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", 30*time.Second)
	v.SetDefault("server.write_timeout", 30*time.Second)
	v.SetDefault("server.shutdown_timeout", 10*time.Second)
	
	// Storage defaults
	v.SetDefault("storage.provider", "s3")
	v.SetDefault("storage.region", "us-east-1")
	v.SetDefault("storage.use_path_style", false)
	v.SetDefault("storage.public_access", false)
	v.SetDefault("storage.signed_url_expiry", 1*time.Hour)
	
	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", 5*time.Minute)
	
	// Redis defaults
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.db", 0)
	
	// Auth defaults
	v.SetDefault("auth.cache_duration", 5*time.Minute)
	
	// Upload defaults
	v.SetDefault("upload.max_file_size", 10*1024*1024)       // 10MB
	v.SetDefault("upload.max_chunked_size", 5*1024*1024*1024) // 5GB
	v.SetDefault("upload.chunk_size", 5*1024*1024)            // 5MB
	v.SetDefault("upload.chunk_threshold", 100*1024*1024)     // 100MB
	v.SetDefault("upload.session_expiry", 24*time.Hour)
	v.SetDefault("upload.allowed_mime_types", []string{
		"image/jpeg", "image/png", "image/gif",
		"application/pdf", "video/mp4", "video/quicktime",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	})
	v.SetDefault("upload.rate_limit_per_min", 100)
	v.SetDefault("upload.worker_pool_size", 10)
	v.SetDefault("upload.retention_days", 30)
	
	// Scanner defaults
	v.SetDefault("scanner.enabled", true)
	v.SetDefault("scanner.host", "localhost")
	v.SetDefault("scanner.port", 3310)
	v.SetDefault("scanner.timeout", 30*time.Second)
	
	// Metrics defaults
	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.path", "/metrics")
}
