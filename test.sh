# test.sh
#!/bin/bash

echo "Running WebSocket tests..."

# 运行单元测试
echo "=== Running Unit Tests ==="
go test -v ./pkg/wsmanager/... -cover
go test -v ./internal/handler/... -cover

# 运行基准测试
echo "=== Running Benchmark Tests ==="
go test -bench=. -benchmem ./pkg/wsmanager/
go test -bench=. -benchmem ./internal/handler/

# 运行集成测试（需要先启动服务）
echo "=== Running Integration Tests ==="
# 先启动服务在后台
go run cmd/server/main.go &
SERVER_PID=$!

# 等待服务启动
sleep 3

# 运行集成测试
go run integration_test/stress_test.go

# 停止服务
kill $SERVER_PID

echo "All tests completed!"