package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// 保存原始环境变量
	originalEnv := os.Getenv("ENV")
	defer os.Setenv("ENV", originalEnv)

	// 测试默认配置
	os.Setenv("ENV", "test")
	cfg := Load()

	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "test", cfg.Env)
	assert.Equal(t, "localhost:8848", cfg.Nacos.ServerAddr)
	assert.Equal(t, "muc-service", cfg.ServiceNames.MucService)
	assert.Equal(t, 1024, cfg.WebSocket.ReadBufferSize)
}

func TestLoadConfigWithEnvVars(t *testing.T) {
	// 设置环境变量
	os.Setenv("PORT", "9090")
	os.Setenv("NACOS_SERVER_ADDR", "nacos-test:8848")
	os.Setenv("MUC_SERVICE_NAME", "test-muc-service")
	os.Setenv("WS_READ_BUFFER_SIZE", "2048")

	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("NACOS_SERVER_ADDR")
		os.Unsetenv("MUC_SERVICE_NAME")
		os.Unsetenv("WS_READ_BUFFER_SIZE")
	}()

	cfg := Load()

	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "nacos-test:8848", cfg.Nacos.ServerAddr)
	assert.Equal(t, "test-muc-service", cfg.ServiceNames.MucService)
	assert.Equal(t, 2048, cfg.WebSocket.ReadBufferSize)
}

func TestConfigDefaults(t *testing.T) {
	cfg := Load()

	// 测试默认值
	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "development", cfg.Env)
	assert.Equal(t, "localhost:8848", cfg.Nacos.ServerAddr)
	assert.Equal(t, "public", cfg.Nacos.Namespace)
	assert.Equal(t, "muc-service", cfg.ServiceNames.MucService)
	assert.Equal(t, "im-business-service", cfg.ServiceNames.BusinessService)
	assert.Equal(t, 1024, cfg.WebSocket.ReadBufferSize)
	assert.Equal(t, 1024, cfg.WebSocket.WriteBufferSize)
	assert.True(t, cfg.WebSocket.EnableCompression)
	assert.Equal(t, 10, cfg.WebSocket.HandshakeTimeout)
}
