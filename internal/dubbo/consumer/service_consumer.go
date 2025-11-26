package consumer

import (
	"log"

	"dubbo.apache.org/dubbo-go/v3/config"
	"github.com/bdsw/bdsw-im-ws/api/msg"
	"github.com/bdsw/bdsw-im-ws/api/muc"
)

type IMAServiceConsumer struct {
	MsgService msg.MsgServiceClientImpl
	MUCService muc.MUCServiceClientImpl
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
