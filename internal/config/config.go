package config

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config 全局配置
type Config struct {
	App       AppConfig       `yaml:"app"`
	Redis     RedisConfig     `yaml:"redis"`
	WebSocket WebSocketConfig `yaml:"websocket"`
}

// AppConfig 应用配置
type AppConfig struct {
	Name        string            `yaml:"name"`
	Version     string            `yaml:"version"`
	Environment string            `yaml:"environment"`
	Discovery   DiscoveryConfig   `yaml:"discovery"`
	Service     ServiceConfig     `yaml:"service"`
	Security    SecurityConfig    `yaml:"security"`
	Performance PerformanceConfig `yaml:"performance"`
	Monitor     MonitorConfig     `yaml:"monitor"`
	Log         LogConfig         `yaml:"log"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Standalone RedisStandaloneConfig `yaml:"standalone"`
	KeyPrefix  string                `yaml:"key_prefix"`
	TTL        RedisTTLConfig        `yaml:"ttl"`
}

type RedisStandaloneConfig struct {
	Addr               string `yaml:"addr"`
	Password           string `yaml:"password"`
	DB                 int    `yaml:"db"`
	PoolSize           int    `yaml:"pool_size"`
	MinIdleConns       int    `yaml:"min_idle_conns"`
	MaxRetries         int    `yaml:"max_retries"`
	DialTimeout        string `yaml:"dial_timeout"`
	ReadTimeout        string `yaml:"read_timeout"`
	WriteTimeout       string `yaml:"write_timeout"`
	PoolTimeout        string `yaml:"pool_timeout"`
	IdleTimeout        string `yaml:"idle_timeout"`
	IdleCheckFrequency string `yaml:"idle_check_frequency"`
}

type RedisTTLConfig struct {
	UserConnection int `yaml:"user_connection"`
	InstanceStatus int `yaml:"instance_status"`
	TokenCache     int `yaml:"token_cache"`
	OnlineUsers    int `yaml:"online_users"`
}

// WebSocketConfig WebSocket配置
type WebSocketConfig struct {
	Server    WebSocketServerConfig `yaml:"server"`
	CORS      CORSConfig            `yaml:"cors"`
	RateLimit RateLimitConfig       `yaml:"rate_limit"`
}

type WebSocketServerConfig struct {
	Port              string `yaml:"port"`
	ReadBufferSize    int    `yaml:"read_buffer_size"`
	WriteBufferSize   int    `yaml:"write_buffer_size"`
	MaxMessageSize    int    `yaml:"max_message_size"`
	HandshakeTimeout  string `yaml:"handshake_timeout"`
	WriteTimeout      string `yaml:"write_timeout"`
	PongWait          string `yaml:"pong_wait"`
	PingPeriod        string `yaml:"ping_period"`
	MaxConnections    int    `yaml:"max_connections"`
	EnableCompression bool   `yaml:"enable_compression"`
}

type CORSConfig struct {
	Enabled      bool     `yaml:"enabled"`
	AllowOrigins []string `yaml:"allow_origins"`
	AllowMethods []string `yaml:"allow_methods"`
	AllowHeaders []string `yaml:"allow_headers"`
}

type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled"`
	RequestsPerSecond int  `yaml:"requests_per_second"`
	Burst             int  `yaml:"burst"`
}

// 其他配置结构...
type DiscoveryConfig struct {
	Nacos NacosConfig `yaml:"nacos"`
}

type NacosConfig struct {
	ServerAddr  string `yaml:"server_addr"`
	Namespace   string `yaml:"namespace"`
	Group       string `yaml:"group"`
	ClusterName string `yaml:"cluster_name"`
}

type ServiceConfig struct {
	IMA          ServiceInstanceConfig `yaml:"ima"`
	Dependencies DependenciesConfig    `yaml:"dependencies"`
}

type ServiceInstanceConfig struct {
	InstanceID string `yaml:"instance_id"`
	Weight     int    `yaml:"weight"`
	Healthy    bool   `yaml:"healthy"`
	Enabled    bool   `yaml:"enabled"`
}

type DependenciesConfig struct {
	MUCService  ServiceReferenceConfig `yaml:"muc_service"`
	MSGService  ServiceReferenceConfig `yaml:"msg_service"`
	PushService ServiceReferenceConfig `yaml:"push_service"`
}

type ServiceReferenceConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Group   string `yaml:"group"`
}

type SecurityConfig struct {
	TokenValidation TokenValidationConfig `yaml:"token_validation"`
	MultiLogin      MultiLoginConfig      `yaml:"multi_login"`
}

type TokenValidationConfig struct {
	Enabled      bool   `yaml:"enabled"`
	CacheEnabled bool   `yaml:"cache_enabled"`
	CacheTTL     string `yaml:"cache_ttl"`
	Timeout      string `yaml:"timeout"`
}

type MultiLoginConfig struct {
	Strategy           string   `yaml:"strategy"`
	MaxDevicesPerUser  int      `yaml:"max_devices_per_user"`
	AllowedDeviceTypes []string `yaml:"allowed_device_types"`
}

type PerformanceConfig struct {
	MaxGoroutines        int    `yaml:"max_goroutines"`
	ConnectionPoolSize   int    `yaml:"connection_pool_size"`
	MessageQueueSize     int    `yaml:"message_queue_size"`
	BatchProcessSize     int    `yaml:"batch_process_size"`
	BatchProcessInterval string `yaml:"batch_process_interval"`
}

type MonitorConfig struct {
	Enabled         bool   `yaml:"enabled"`
	PrometheusPort  string `yaml:"prometheus_port"`
	MetricsPath     string `yaml:"metrics_path"`
	HealthCheckPath string `yaml:"health_check_path"`
	StatsInterval   string `yaml:"stats_interval"`
}

type LogConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	Output     string `yaml:"output"`
	FilePath   string `yaml:"file_path"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
	Compress   bool   `yaml:"compress"`
}

var (
	globalConfig *Config
	configOnce   sync.Once
)

// LoadConfig 加载配置
func LoadConfig(configPaths ...string) (*Config, error) {
	var err error
	configOnce.Do(func() {
		globalConfig, err = loadConfigFiles(configPaths...)
	})
	return globalConfig, err
}

// GetConfig 获取全局配置
func GetConfig() *Config {
	return globalConfig
}

func loadConfigFiles(configPaths ...string) (*Config, error) {
	config := &Config{}

	// 默认配置文件路径
	if len(configPaths) == 0 {
		configPaths = []string{
			"config/app_dev.yml",
			"config/redis/redis.yml",
			"config/websocket/websocket.yml",
			"config/dubbo/consumer.yml",
			"config/dubbo/dubbo.yml",
		}
	}

	for _, path := range configPaths {
		if err := loadYAMLFile(path, config); err != nil {
			return nil, err
		}
	}

	return config, nil
}

func loadYAMLFile(filePath string, config interface{}) error {
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("Config file %s does not exist, skipping", filePath)
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, config)
}

// InitConfig 初始化配置
func InitConfig() error {
	// 确定配置文件路径
	configDir := "config"
	if envConfigDir := os.Getenv("CONFIG_DIR"); envConfigDir != "" {
		configDir = envConfigDir
	}

	configPaths := []string{
		filepath.Join(configDir, "app_dev.yml"),
		filepath.Join(configDir, "redis/redis.yml"),
		filepath.Join(configDir, "websocket/websocket.yml"),
		filepath.Join(configDir, "dubbo/consumer.yml"),
		filepath.Join(configDir, "dubbo/dubbo.yml"),
	}

	_, err := LoadConfig(configPaths...)
	return err
}
