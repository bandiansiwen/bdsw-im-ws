package config

import (
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Port string `mapstructure:"PORT"`
	Env  string `mapstructure:"ENV"`

	// Nacos配置
	Nacos struct {
		ServerAddr string `mapstructure:"NACOS_SERVER_ADDR"`
		Namespace  string `mapstructure:"NACOS_NAMESPACE"`
		Group      string `mapstructure:"NACOS_GROUP"`
		DataID     string `mapstructure:"NACOS_DATA_ID"`
	} `mapstructure:"NACOS"`

	// 服务名称（从Nacos发现）
	ServiceNames struct {
		UserService     string `mapstructure:"USER_SERVICE_NAME"`
		BusinessService string `mapstructure:"BUSINESS_SERVICE_NAME"`
	} `mapstructure:"SERVICE_NAMES"`

	WebSocket struct {
		ReadBufferSize    int   `mapstructure:"WS_READ_BUFFER_SIZE"`
		WriteBufferSize   int   `mapstructure:"WS_WRITE_BUFFER_SIZE"`
		EnableCompression bool  `mapstructure:"WS_ENABLE_COMPRESSION"`
		HandshakeTimeout  int   `mapstructure:"WS_HANDSHAKE_TIMEOUT"`
		MaxMessageSize    int64 `mapstructure:"WS_MAX_MESSAGE_SIZE"`
		PingInterval      int   `mapstructure:"WS_PING_INTERVAL"`
		PongWait          int   `mapstructure:"WS_PONG_WAIT"`
		WriteWait         int   `mapstructure:"WS_WRITE_WAIT"`
	} `mapstructure:"WEBSOCKET"`
}

func Load() *Config {
	// 设置默认值
	setDefaults()

	// 根据环境加载对应的.env文件
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	// 加载.env文件
	viper.SetConfigName(".env." + env)
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/websocket-service/")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: No config file found, using environment variables and defaults: %v", err)
	}

	// 加载通用的.env文件（如果存在）
	viper.SetConfigName(".env.development")
	if err := viper.MergeInConfig(); err == nil {
		log.Printf("Loaded common .env.development file")
	}

	// 自动读取环境变量（优先级最高）
	viper.AutomaticEnv()
	viper.SetEnvPrefix("WS")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("Unable to decode config into struct: %v", err)
	}

	logConfig(&config)
	return &config
}

func setDefaults() {
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("ENV", "development")

	// Nacos默认配置
	viper.SetDefault("NACOS_SERVER_ADDR", "localhost:8848")
	viper.SetDefault("NACOS_NAMESPACE", "public")
	viper.SetDefault("NACOS_GROUP", "DEFAULT_GROUP")
	viper.SetDefault("NACOS_DATA_ID", "websocket-service")

	// 服务名称默认值
	viper.SetDefault("USER_SERVICE_NAME", "user-service")
	viper.SetDefault("BUSINESS_SERVICE_NAME", "im-business-service")

	// WebSocket默认配置
	viper.SetDefault("WS_READ_BUFFER_SIZE", 1024)
	viper.SetDefault("WS_WRITE_BUFFER_SIZE", 1024)
	viper.SetDefault("WS_ENABLE_COMPRESSION", true)
	viper.SetDefault("WS_HANDSHAKE_TIMEOUT", 10)
	viper.SetDefault("WS_MAX_MESSAGE_SIZE", 512)
	viper.SetDefault("WS_PING_INTERVAL", 30)
	viper.SetDefault("WS_PONG_WAIT", 60)
	viper.SetDefault("WS_WRITE_WAIT", 10)
}

func logConfig(cfg *Config) {
	log.Printf("=== Configuration ===")
	log.Printf("Environment: %s", cfg.Env)
	log.Printf("Port: %s", cfg.Port)
	log.Printf("Nacos Server: %s", cfg.Nacos.ServerAddr)
	log.Printf("Nacos Namespace: %s", cfg.Nacos.Namespace)
	log.Printf("User Service: %s", cfg.ServiceNames.UserService)
	log.Printf("Business Service: %s", cfg.ServiceNames.BusinessService)
	log.Printf("WebSocket - ReadBuffer: %d, WriteBuffer: %d",
		cfg.WebSocket.ReadBufferSize, cfg.WebSocket.WriteBufferSize)
	log.Printf("=====================")
}
