package consumer

import (
	"log"

	"dubbo.apache.org/dubbo-go/v3/config"
	"github.com/bdsw/bdsw-im-ws/proto/service"
)

type IMAServiceConsumer struct {
	GatewayService service.GatewayServiceClientImpl
}

func NewIMAServiceConsumer() (*IMAServiceConsumer, error) {
	client := &IMAServiceConsumer{}

	// 获取 GatewayService 客户端
	config.SetConsumerService(&client.GatewayService)

	log.Println("Dubbo clients initialized successfully")
	return client, nil
}
