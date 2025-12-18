package config

import (
	"fmt"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/spf13/viper"
)

// Config holds all configuration for a service
type Config struct {
	Service   ServiceConfig   `mapstructure:"service"`
	HTTP      HTTPConfig      `mapstructure:"http"`
	GRPC      GRPCConfig      `mapstructure:"grpc"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Kafka     KafkaConfig     `mapstructure:"kafka"`
	Auth      AuthConfig      `mapstructure:"auth"`
	Logger    LoggerConfig    `mapstructure:"logger"`
	Telemetry TelemetryConfig `mapstructure:"telemetry"`
	Version   string          `mapstructure:"version"`
}

// ServiceConfig holds service-specific configuration
type ServiceConfig struct {
	Name        string `mapstructure:"name" envconfig:"SERVICE_NAME"`
	Environment string `mapstructure:"environment" envconfig:"ENVIRONMENT" default:"development"`
}

// HTTPConfig holds HTTP server configuration
type HTTPConfig struct {
	Port         int           `mapstructure:"port" envconfig:"HTTP_PORT" default:"8080"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout" envconfig:"HTTP_READ_TIMEOUT" default:"10s"`
	WriteTimeout time.Duration `mapstructure:"write_timeout" envconfig:"HTTP_WRITE_TIMEOUT" default:"10s"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout" envconfig:"HTTP_IDLE_TIMEOUT" default:"120s"`
}

// GRPCConfig holds gRPC server configuration
type GRPCConfig struct {
	Port int `mapstructure:"port" envconfig:"GRPC_PORT" default:"9090"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string        `mapstructure:"host" envconfig:"DB_HOST" default:"localhost"`
	Port            int           `mapstructure:"port" envconfig:"DB_PORT" default:"5432"`
	User            string        `mapstructure:"user" envconfig:"DB_USER" default:"postgres"`
	Password        string        `mapstructure:"password" envconfig:"DB_PASSWORD" default:"postgres"`
	Database        string        `mapstructure:"database" envconfig:"DB_NAME" default:"linkflow"`
	Schema          string        `mapstructure:"schema" envconfig:"DB_SCHEMA"`
	SSLMode         string        `mapstructure:"ssl_mode" envconfig:"DB_SSL_MODE" default:"disable"`
	MaxOpenConns    int           `mapstructure:"max_open_conns" envconfig:"DB_MAX_OPEN_CONNS" default:"25"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns" envconfig:"DB_MAX_IDLE_CONNS" default:"5"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime" envconfig:"DB_CONN_MAX_LIFETIME" default:"5m"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time" envconfig:"DB_CONN_MAX_IDLE_TIME" default:"10m"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host         string        `mapstructure:"host" envconfig:"REDIS_HOST" default:"localhost"`
	Port         int           `mapstructure:"port" envconfig:"REDIS_PORT" default:"6379"`
	Password     string        `mapstructure:"password" envconfig:"REDIS_PASSWORD"`
	DB           int           `mapstructure:"db" envconfig:"REDIS_DB" default:"0"`
	PoolSize     int           `mapstructure:"pool_size" envconfig:"REDIS_POOL_SIZE" default:"10"`
	MinIdleConns int           `mapstructure:"min_idle_conns" envconfig:"REDIS_MIN_IDLE_CONNS" default:"5"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout" envconfig:"REDIS_DIAL_TIMEOUT" default:"5s"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout" envconfig:"REDIS_READ_TIMEOUT" default:"3s"`
	WriteTimeout time.Duration `mapstructure:"write_timeout" envconfig:"REDIS_WRITE_TIMEOUT" default:"3s"`
}

// KafkaConfig holds Kafka configuration
type KafkaConfig struct {
	Brokers       []string `mapstructure:"brokers" envconfig:"KAFKA_BROKERS" default:"localhost:9092"`
	ConsumerGroup string   `mapstructure:"consumer_group" envconfig:"KAFKA_CONSUMER_GROUP"`
	Topics        []string `mapstructure:"topics" envconfig:"KAFKA_TOPICS"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret           string        `mapstructure:"jwt_secret" envconfig:"JWT_SECRET" default:"super-secret-key"`
	JWTExpiry           time.Duration `mapstructure:"jwt_expiry" envconfig:"JWT_EXPIRY" default:"1h"`
	RefreshTokenExpiry  time.Duration `mapstructure:"refresh_token_expiry" envconfig:"REFRESH_TOKEN_EXPIRY" default:"168h"`
	PasswordMinLength   int           `mapstructure:"password_min_length" envconfig:"PASSWORD_MIN_LENGTH" default:"8"`
	MaxLoginAttempts    int           `mapstructure:"max_login_attempts" envconfig:"MAX_LOGIN_ATTEMPTS" default:"5"`
	LockoutDuration     time.Duration `mapstructure:"lockout_duration" envconfig:"LOCKOUT_DURATION" default:"15m"`
}

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level      string `mapstructure:"level" envconfig:"LOG_LEVEL" default:"info"`
	Format     string `mapstructure:"format" envconfig:"LOG_FORMAT" default:"json"`
	OutputPath string `mapstructure:"output_path" envconfig:"LOG_OUTPUT_PATH" default:"stdout"`
}

// TelemetryConfig holds telemetry configuration
type TelemetryConfig struct {
	MetricsEnabled bool   `mapstructure:"metrics_enabled" envconfig:"METRICS_ENABLED" default:"true"`
	TracingEnabled bool   `mapstructure:"tracing_enabled" envconfig:"TRACING_ENABLED" default:"true"`
	JaegerEndpoint string `mapstructure:"jaeger_endpoint" envconfig:"JAEGER_ENDPOINT" default:"http://localhost:14268/api/traces"`
	ServiceName    string `mapstructure:"service_name" envconfig:"TELEMETRY_SERVICE_NAME"`
}

// Load loads configuration from files and environment
func Load(serviceName string) (*Config, error) {
	var cfg Config

	// Set default service name
	cfg.Service.Name = serviceName
	cfg.Telemetry.ServiceName = serviceName

	// Set config file paths
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("./configs/services/" + serviceName)
	viper.AddConfigPath(".")

	// Read config file if exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found; ignore error and continue with env vars
	}

	// Unmarshal config file
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Override with environment variables
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to process env vars: %w", err)
	}

	// Service-specific environment variables
	envPrefix := fmt.Sprintf("%s_", toEnvPrefix(serviceName))
	if err := envconfig.Process(envPrefix, &cfg); err != nil {
		return nil, fmt.Errorf("failed to process service env vars: %w", err)
	}

	// Set schema based on service name if not provided
	if cfg.Database.Schema == "" {
		cfg.Database.Schema = serviceName + "_service"
	}

	// Set Kafka consumer group if not provided
	if cfg.Kafka.ConsumerGroup == "" {
		cfg.Kafka.ConsumerGroup = serviceName + "-consumer"
	}

	// Set version
	if version := os.Getenv("VERSION"); version != "" {
		cfg.Version = version
	} else {
		cfg.Version = "dev"
	}

	return &cfg, nil
}

// DSN returns the database connection string
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode)
}

// RedisAddr returns the Redis address
func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// toEnvPrefix converts service name to environment variable prefix
func toEnvPrefix(name string) string {
	result := ""
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result += "_"
		}
		if r >= 'a' && r <= 'z' {
			result += string(r - 32) // Convert to uppercase
		} else {
			result += string(r)
		}
	}
	return result
}
