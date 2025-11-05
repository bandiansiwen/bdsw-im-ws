package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	Addr     string
	Password string
	DB       int
}

type Client struct {
	client *redis.Client
}

func NewClient(config *Config) *Client {
	return &Client{
		client: redis.NewClient(&redis.Options{
			Addr:     config.Addr,
			Password: config.Password,
			DB:       config.DB,
		}),
	}
}

func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *Client) SetUserConnection(userID, deviceID string, connInfo map[string]interface{}) error {
	ctx := context.Background()

	data, err := json.Marshal(connInfo)
	if err != nil {
		return err
	}

	// 设置用户设备连接信息
	deviceKey := fmt.Sprintf("user_conn:%s:%s", userID, deviceID)
	err = c.client.Set(ctx, deviceKey, data, 24*time.Hour).Err()
	if err != nil {
		return err
	}

	// 设置用户到实例的映射
	userKey := fmt.Sprintf("user_ima:%s", userID)

	// 先获取现有数据
	existingData, err := c.client.Get(ctx, userKey).Result()
	var userData map[string]interface{}
	if err == nil {
		json.Unmarshal([]byte(existingData), &userData)
	} else {
		userData = make(map[string]interface{})
	}

	// 更新数据
	userData["ima_instance"] = connInfo["ima_instance"]
	userData["last_update"] = time.Now().Unix()

	// 更新设备列表
	if devices, ok := userData["devices"].([]string); ok {
		// 检查设备是否已存在
		found := false
		for _, d := range devices {
			if d == deviceID {
				found = true
				break
			}
		}
		if !found {
			devices = append(devices, deviceID)
			userData["devices"] = devices
		}
	} else {
		userData["devices"] = []string{deviceID}
	}

	userDataBytes, _ := json.Marshal(userData)
	return c.client.Set(ctx, userKey, userDataBytes, 24*time.Hour).Err()
}

func (c *Client) RemoveUserConnection(userID, deviceID string) error {
	ctx := context.Background()

	// 删除用户设备连接信息
	deviceKey := fmt.Sprintf("user_conn:%s:%s", userID, deviceID)
	return c.client.Del(ctx, deviceKey).Err()
}

func (c *Client) SetInstanceStatus(instanceID string, status map[string]interface{}) error {
	ctx := context.Background()

	data, err := json.Marshal(status)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, fmt.Sprintf("ima_instance:%s", instanceID), data, 2*time.Minute).Err()
}

func (c *Client) RemoveInstanceStatus(instanceID string) error {
	ctx := context.Background()
	return c.client.Del(ctx, fmt.Sprintf("ima_instance:%s", instanceID)).Err()
}

func (c *Client) GetUserImaInstance(userID string) (string, error) {
	ctx := context.Background()

	data, err := c.client.Get(ctx, fmt.Sprintf("user_ima:%s", userID)).Result()
	if err != nil {
		return "", err
	}

	var userInfo map[string]interface{}
	if err := json.Unmarshal([]byte(data), &userInfo); err != nil {
		return "", err
	}

	if instance, ok := userInfo["ima_instance"].(string); ok {
		return instance, nil
	}

	return "", fmt.Errorf("ima_instance not found")
}

func (c *Client) Close() error {
	return c.client.Close()
}
