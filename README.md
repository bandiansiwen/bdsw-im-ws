# WebSocket连接服务的唯一职责
- 连接管理 - 建立、维持、断开WebSocket连接
- 心跳处理 - 维持连接活跃（ping/pong）
- 基础认证 - 连接级别的token认证

所有IM相关的业务逻辑都应该在业务服务器中处理。

## 技术栈选择
WebSocket库: gorilla/websocket (最成熟稳定)

HTTP框架: Gin (高性能，中间件生态好) 或 标准库

配置管理: Viper

监控: Prometheus + Grafana

压测: github.com/gorilla/websocket 自带的压测工具