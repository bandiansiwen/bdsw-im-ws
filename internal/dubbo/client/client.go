package client

import (
	"bdsw-im-ws/api/common"
	"bdsw-im-ws/api/muc"
	"bdsw-im-ws/api/server_push"
	"context"
	"log"
	"time"

	"bdsw-im-ws/api/business_message"

	"dubbo.apache.org/dubbo-go/v3/config"
)

type Client struct {
	BusinessMessageService business_message.BusinessMessageService
	MUCService             muc.MUCService
	ServerPushService      server_push.ServerPushService
}

func NewClient() (*Client, error) {
	client := &Client{}

	// 获取 BusinessMessageService 客户端
	config.SetConsumerService(&client.BusinessMessageService)
	// 获取 MUCService 客户端
	config.SetConsumerService(&client.MUCService)
	// 获取 ServerPushService 客户端
	config.SetConsumerService(&client.ServerPushService)

	log.Println("Dubbo clients initialized successfully")
	return client, nil
}

func (c *Client) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 检查 MUC 服务是否可用
	_, err := c.MUCService.ValidateToken(ctx, &muc.TokenValidateRequest{
		UserId: "healthcheck",
		Token:  "healthcheck",
	})

	if err != nil {
		log.Printf("MUC service health check failed: %v", err)
		return err
	}

	// 检查 BusinessMessageService 是否可用
	_, err = c.BusinessMessageService.HandleClientMessage(ctx, &common.ClientMessage{
		UserId: "healthcheck",
		Type:   "health",
		Action: "check",
	})

	if err != nil {
		log.Printf("BusinessMessageService health check failed: %v", err)
	}

	return nil
}
