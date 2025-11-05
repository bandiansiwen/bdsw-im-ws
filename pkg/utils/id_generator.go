package utils

import (
	"fmt"
	"os"
	"time"
)

func GenerateInstanceID() string {
	hostname, _ := os.Hostname()
	return fmt.Sprintf("ima-gateway-%s-%d", hostname, time.Now().Unix())
}

func GenerateMessageID() string {
	return fmt.Sprintf("msg-%d", time.Now().UnixNano())
}
