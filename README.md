# bdsw-im-ms项目介绍

## WebSocket连接服务的唯一职责
- 连接管理 - 建立、维持、断开WebSocket连接
- 心跳处理 - 维持连接活跃（ping/pong）
- 基础认证 - 连接级别的token认证

所有IM相关的业务逻辑都应该在业务服务器中处理。

## 技术栈选择
```
语言: Go 1.25
协议: WebSocket + Dubbo3 Triple
序列化: Protobuf
服务发现: Nacos
缓存: Redis
```
选择理由：
- Go 语言：高并发、轻量级、编译部署简单，适合网关层的高并发连接管理
- WebSocket：双向实时通信，适合IM场景的即时消息推送
- Dubbo3 Triple：基于 HTTP/2 的 RPC 协议，与 gRPC 兼容，支持流式通信
- Protobuf：高性能二进制序列化，跨语言支持完善
- Nacos：阿里开源的服务发现组件，与 Dubbo 生态集成良好

## 项目目录如下
```
项目结构如下：

bdsw-im-ws/
├── main.go // 程序入口
├── config/ // 配置文件目录
│ ├── server.yml // Dubbo服务端配置
│ ├── client.yml // Dubbo客户端配置
│ ├── redis.yml // Redis配置
│ ├── websocket.yml // WebSocket配置
│ └── app.yml // 应用配置
├── api/
│ ├── im_gateway.proto // Protobuf定义
│ ├── im_gateway.pb.go // 由protoc生成的Go代码
│ └── im_gateway_triple.pb.go // 由protoc-gen-go-triple生成的Go代码
├── internal/
│ ├── service/ // 业务逻辑层
│ │ ├── gateway_service.go // 网关主服务
│ │ ├── connection_manager.go // 连接管理器
│ │ ├── token_cache.go // Token缓存
│ │ └── websocket_server.go // WebSocket服务器
│ ├── dubbo/ // Dubbo相关
│ │ ├── client.go // Dubbo客户端
│ │ ├── service_provider.go // Dubbo服务提供者
│ │ └── nacos_discovery.go // Nacos服务发现（暂未实现，可能包含在配置中）
│ ├── redis/ // Redis客户端
│ │ └── client.go
│ └── config/ // 配置加载
│ └── config.go
└── pkg/
└── utils/ // 工具包
├── id_generator.go // ID生成器
└── validator.go // 验证器（暂未实现）
```

### 1. 主程序 (main.go)
**作用**
- 应用程序入口点
- 初始化所有组件和依赖
- 启动服务和优雅关闭

**方法作用**
- main(): 程序入口，初始化配置、Redis、Dubbo客户端，启动网关服务
- waitForShutdown(): 监听系统信号，实现优雅关闭

### 2. API 定义 (api/im_gateway.proto)
**作用**
- 定义所有 Dubbo 服务的 Protobuf 接口
- 生成 Go 和 Java 互通的序列化代码

**消息类型作用**
- TokenValidateRequest/Response: Token 验证请求和响应
- KickUserRequest/Response: 踢用户下线请求和响应
- ClientMessage: 客户端上行消息格式
- ServerMessage: 服务端下行消息格式
- PushRequest/Response: 消息推送请求和响应
- ConnectionStatus: 用户连接状态信息
- Heartbeat: 服务心跳信息

**服务接口作用**
- IMAGatewayService: IMA 网关提供的服务（供业务服务调用）
- BusinessMessageService: 业务消息服务（IMA 调用处理业务）
- MUCService: 认证服务（IMA 调用进行 Token 验证）
- ServerPushService: 服务端推送服务（流式通信）

### 3. 网关主服务 (internal/service/gateway_service.go)
**作用**
- IMA 服务的核心业务逻辑
- 管理 WebSocket 连接和消息路由
- 协调 Dubbo 服务调用

**方法作用**

**生命周期管理**
- NewGatewayService(): 创建网关服务实例
- Start(): 启动所有服务组件
- Shutdown(): 优雅关闭，清理资源

**连接管理**
- handleWebSocketConnection(): 处理新 WebSocket 连接，包含 Token 验证
- validateToken(): 调用 MUC 服务验证用户 Token
- shouldKickExistingConnection(): 检查是否需要踢掉现有连接（多端登录策略）
- kickExistingConnection(): 踢用户下线
- sendAuthResult(): 发送认证结果给客户端

**消息处理**
- handleClientMessage(): 处理客户端上行消息，转发到业务服务
- sendServerMessageToUser(): 向指定用户发送服务端消息
- broadcastMessage(): 广播消息给所有在线用户
- sendErrorToUser(): 发送错误消息给用户

**流式通信**
- connectToServerPushStream(): 连接到业务服务的推送流
- handleServerPush(): 处理服务端推送的消息
- sendHeartbeat(): 发送心跳到推送流

**状态管理**
- registerUserConnection(): 注册用户连接到 Redis
- unregisterUserConnection(): 从 Redis 注销用户连接
- notifyUserStatusChange(): 通知用户状态变化
- updateInstanceStatus(): 更新服务实例状态

**健康监控**
- startHealthCheck(): 启动健康检查
- startStatsReporter(): 启动统计报告

### 4. 连接管理器 (internal/service/connection_manager.go)
**作用**
- 管理所有 WebSocket 连接
- 提供线程安全的连接操作
- 支持按用户和设备管理连接

**方法作用**

**连接管理**
- Register(): 注册用户连接（用户ID + 设备ID）
- Unregister(): 注销用户连接
- Get(): 获取指定用户的连接
- GetConnection(): 获取完整的连接信息

**查询统计**
- GetUserDevices(): 获取用户的所有设备ID
- GetAllUserDevices(): 获取所有用户的设备信息
- GetOnlineCount(): 获取在线用户总数

**消息广播**
- BroadcastToUser(): 向用户的所有设备广播消息
- Broadcast(): 向所有在线用户广播消息
- CloseAll(): 关闭所有连接（服务关闭时使用）

### 5. Token 缓存 (internal/service/token_cache.go)
**作用**
- 缓存 Token 验证结果，减少重复验证开销
- 自动清理过期缓存

**方法作用**
- NewTokenCache(): 创建 Token 缓存实例
- Get(): 从缓存获取 Token 验证结果
- Set(): 设置 Token 验证结果到缓存
- Invalidate(): 使某个用户的 Token 缓存失效
- startCleanup(): 启动定期清理任务
- cleanup(): 清理过期缓存

### 6. WebSocket 服务器 (internal/service/websocket_server.go)
**作用**
- 管理 WebSocket 服务器
- 处理连接升级和路由

**方法作用**
- NewWebSocketServer(): 创建 WebSocket 服务器
- Start(): 启动 WebSocket 服务器
- handleWebSocket(): 处理 WebSocket 连接请求，升级 HTTP 连接
- Shutdown(): 关闭 WebSocket 服务器

### 7. Dubbo 客户端 (internal/dubbo/client.go)
**作用**
- 封装所有 Dubbo 服务客户端
- 提供服务发现和负载均衡

**方法作用**
- NewClient(): 创建 Dubbo 客户端，初始化所有服务引用
- HealthCheck(): 检查 Dubbo 服务健康状态

### 8. Dubbo 服务提供者 (internal/dubbo/service_provider.go)
**作用**
- 实现 IMA 网关的 Dubbo 服务接口
- 供其他服务调用 IMA 网关功能

**方法作用**
- NewIMAGatewayServiceProvider(): 创建服务提供者实例
- PushToUser(): 处理单用户推送请求
- PushToUsers(): 处理批量推送请求
- Broadcast(): 处理广播消息请求
- KickUser(): 处理踢用户下线请求

### 9. Redis 客户端 (internal/redis/client.go)
**作用**
- 封装 Redis 操作
- 管理用户连接状态和服务实例状态

方法作用
- NewClient(): 创建 Redis 客户端
- Ping(): 检查 Redis 连接
- SetUserConnection(): 设置用户连接信息
- RemoveUserConnection(): 移除用户连接信息
- SetInstanceStatus(): 设置服务实例状态
- RemoveInstanceStatus(): 移除服务实例状态
- GetUserImaInstance(): 获取用户所在的 IMA 实例
- Close(): 关闭 Redis 连接

### 10. 配置管理 (internal/config/config.go)
**作用**
- 加载和管理所有配置文件
- 提供类型安全的配置访问

**方法作用**
- LoadConfig(): 加载配置文件（单例模式）
- GetConfig(): 获取全局配置实例
- loadConfigFiles(): 加载多个配置文件
- loadYAMLFile(): 加载单个 YAML 文件
- InitConfig(): 初始化配置系统

### 11. 工具函数 (pkg/utils/id_generator.go)
**作用**
- 提供通用的工具函数

**方法作用**
- GenerateInstanceID(): 生成服务实例ID
- GenerateMessageID(): 生成消息ID

### 配置文件作用

- server.yml: Dubbo 服务端配置，定义 IMA 服务提供的 Dubbo 接口
- client.yml: Dubbo 客户端配置，定义如何调用 MUC、MSG 等服务
- redis.yml: Redis 连接和缓存配置
- websocket.yml: WebSocket 服务器配置
- app.yml: 应用通用配置
- dev/app.yml: 开发环境特定配置
- prod/app.yml: 生产环境特定配置







