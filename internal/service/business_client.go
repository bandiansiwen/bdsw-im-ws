package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	_ "log"
	"net/http"
	"time"

	"bdsw-im-ws/pkg/nacos"
)

type BusinessServiceClient struct {
	discovery   *nacos.ServiceDiscovery
	serviceName string
	httpClient  *http.Client
}

func NewBusinessServiceClient(discovery *nacos.ServiceDiscovery, serviceName string) *BusinessServiceClient {
	return &BusinessServiceClient{
		discovery:   discovery,
		serviceName: serviceName,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type MessageRequest struct {
	ClientID  string      `json:"client_id"`
	UserID    string      `json:"user_id"`
	Message   interface{} `json:"message"`
	Timestamp time.Time   `json:"timestamp"`
	IP        string      `json:"ip,omitempty"`
}

type MessageResponse struct {
	Success        bool            `json:"success"`
	UserID         string          `json:"user_id,omitempty"`
	SendToClient   bool            `json:"send_to_client"`
	Data           interface{}     `json:"data,omitempty"`
	TargetMessages []TargetMessage `json:"target_messages,omitempty"`
	Error          string          `json:"error,omitempty"`
}

type TargetMessage struct {
	ClientID string      `json:"client_id,omitempty"`
	UserID   string      `json:"user_id,omitempty"`
	Data     interface{} `json:"data"`
}

func (c *BusinessServiceClient) ForwardMessage(clientID, userID string, message interface{}, ip string) (*MessageResponse, error) {
	// 从Nacos获取服务地址
	serviceURL, err := c.discovery.GetServiceURL(c.serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get business service URL: %v", err)
	}

	request := MessageRequest{
		ClientID:  clientID,
		UserID:    userID,
		Message:   message,
		Timestamp: time.Now(),
		IP:        ip,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	resp, err := c.httpClient.Post(serviceURL+"/api/messages/forward", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to send request to business service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("business service returned status: %d", resp.StatusCode)
	}

	var response MessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &response, nil
}

func (c *BusinessServiceClient) NotifyConnectionEstablished(clientID, ip, platform string) error {
	serviceURL, err := c.discovery.GetServiceURL(c.serviceName)
	if err != nil {
		return fmt.Errorf("failed to get business service URL: %v", err)
	}

	event := map[string]interface{}{
		"type":      "connection_established",
		"client_id": clientID,
		"ip":        ip,
		"platform":  platform,
		"timestamp": time.Now(),
	}

	jsonData, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = c.httpClient.Post(serviceURL+"/api/events/connection/established", "application/json", bytes.NewBuffer(jsonData))
	return err
}

func (c *BusinessServiceClient) NotifyConnectionClosed(clientID, userID string) error {
	serviceURL, err := c.discovery.GetServiceURL(c.serviceName)
	if err != nil {
		return fmt.Errorf("failed to get business service URL: %v", err)
	}

	event := map[string]interface{}{
		"type":      "connection_closed",
		"client_id": clientID,
		"user_id":   userID,
		"timestamp": time.Now(),
	}

	jsonData, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = c.httpClient.Post(serviceURL+"/api/events/connection/closed", "application/json", bytes.NewBuffer(jsonData))
	return err
}
