package service

import (
	"bdsw-im-ws/internal/model"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"bdsw-im-ws/pkg/nacos"
)

type AuthClient struct {
	discovery   *nacos.ServiceDiscovery
	serviceName string
	httpClient  *http.Client
}

func NewAuthClient(discovery *nacos.ServiceDiscovery, serviceName string) *AuthClient {
	return &AuthClient{
		discovery:   discovery,
		serviceName: serviceName,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (a *AuthClient) VerifyToken(token string) (*model.UserInfo, error) {
	if token == "" {
		return nil, fmt.Errorf("token is empty")
	}

	// 从Nacos获取服务地址
	serviceURL, err := a.discovery.GetServiceURL(a.serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get user service URL: %v", err)
	}

	// 调用用户服务的token验证接口
	req, err := http.NewRequest("GET", serviceURL+"/VerifyToken", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token verification failed with status: %d", resp.StatusCode)
	}

	var result model.VerifyTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if result.Code > 0 {
		return nil, fmt.Errorf("invalid token: %s", result.Message)
	}

	log.Printf("Token verified successfully for user: %s", result.User.Username)

	// 转换为wsmanager.UserInfo
	return &result.User, nil
}
