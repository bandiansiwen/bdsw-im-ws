package consumer

import (
	"bdsw-im-ws/api/business_message"
	"bdsw-im-ws/api/muc"
	"log"

	"dubbo.apache.org/dubbo-go/v3/config"
)

type IMAServiceConsumer struct {
	BusinessMessageService business_message.BusinessMessageService
	MUCService             muc.MUCService
}

func NewIMAServiceConsumer() (*IMAServiceConsumer, error) {
	client := &IMAServiceConsumer{}

	// 获取 BusinessMessageService 客户端
	config.SetConsumerService(&client.BusinessMessageService)
	// 获取 MUCService 客户端
	config.SetConsumerService(&client.MUCService)

	log.Println("Dubbo clients initialized successfully")
	return client, nil
}
