#!/bin/bash

set -e

echo "Running WebSocket service tests..."

# 运行nacos单元测试
go test -v ./pkg/nacos/... -cover

# 运行服务工厂测试
go test -v ./pkg/service_factory/... -cover

# 运行单元测试
echo "=== Running Unit Tests ==="
go test -v ./internal/config/... -cover
go test -v ./internal/service/... -cover
go test -v ./pkg/wsmanager/... -cover

# 运行Handler测试
echo "=== Running Handler Tests ==="
go test -v ./internal/handler/... -cover

# 运行基准测试
echo "=== Running Benchmark Tests ==="
go test -bench=. -benchmem ./pkg/wsmanager/
go test -bench=. -benchmem ./internal/handler/

# 运行集成测试（需要设置环境）
if [ "$1" = "all" ]; then
    echo "=== Running Integration Tests ==="
    go test -v ./integration/... -tags=integration
fi

# 运行所有测试
go test -v ./... -cover -short

# 生成测试覆盖率报告
echo "=== Generating Coverage Report ==="
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

echo "All tests completed!"