package consumer

import (
	"log"

	"github.com/bdsw/bdsw-im-ws/api/msg"
	"github.com/bdsw/bdsw-im-ws/api/muc"

	"dubbo.apache.org/dubbo-go/v3/config"
)

type IMAServiceConsumer struct {
	MsgService msg.MsgServiceClient
	MUCService muc.MUCServiceClient
}

func NewIMAServiceConsumer() (*IMAServiceConsumer, error) {
	client := &IMAServiceConsumer{}

	// 获取 BusinessMessageService 客户端
	config.SetConsumerService(&client.MsgService)
	// 获取 MUCService 客户端
	config.SetConsumerService(&client.MUCService)

	log.Println("Dubbo clients initialized successfully")
	return client, nil
}
