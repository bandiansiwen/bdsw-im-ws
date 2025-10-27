package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	Port      string `mapstructure:"PORT"`
	Env       string `mapstructure:"ENV"`
	JWTSecret string `mapstructure:"JWT_SECRET"`

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
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("ENV", "development")
	viper.SetDefault("JWT_SECRET", "your-secret-key")

	// WebSocket 默认配置
	viper.SetDefault("WS_READ_BUFFER_SIZE", 1024)
	viper.SetDefault("WS_WRITE_BUFFER_SIZE", 1024)
	viper.SetDefault("WS_ENABLE_COMPRESSION", true)
	viper.SetDefault("WS_HANDSHAKE_TIMEOUT", 10)
	viper.SetDefault("WS_MAX_MESSAGE_SIZE", 512)
	viper.SetDefault("WS_PING_INTERVAL", 30)
	viper.SetDefault("WS_PONG_WAIT", 60)
	viper.SetDefault("WS_WRITE_WAIT", 10)

	viper.AutomaticEnv()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("Unable to decode config into struct: %v", err)
	}

	return &config
}
