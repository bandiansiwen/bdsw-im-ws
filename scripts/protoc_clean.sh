#!/bin/bash

# Protobuf 生成脚本 - 支持 gRPC 和 Triple
# 解决文件找不到和导入路径问题

set -e

echo "🚀 Generating Protobuf data structures (gRPC + Triple)..."

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROTO_DIR="$PROJECT_ROOT/proto"

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 获取 Go 模块名
GO_MODULE=$(go list -m 2>/dev/null || echo "bdsw-im-ws")
echo -e "${BLUE}Using Go module: $GO_MODULE${NC}"

# 检查必要的工具
check_tools() {
    echo -e "${BLUE}=== Checking Required Tools ===${NC}"

    if ! command -v protoc &> /dev/null; then
        echo -e "${RED}❌ protoc not found. Please install protobuf compiler.${NC}"
        exit 1
    fi
    echo -e "${GREEN}✅ protoc found${NC}"

    # 检查 Go 插件
    if ! command -v protoc-gen-go &> /dev/null; then
        echo -e "${YELLOW}⚠️ protoc-gen-go not found, installing...${NC}"
        go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    fi
    echo -e "${GREEN}✅ protoc-gen-go found${NC}"

    if ! command -v protoc-gen-go-grpc &> /dev/null; then
        echo -e "${YELLOW}⚠️ protoc-gen-go-grpc not found, installing...${NC}"
        go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
    fi
    echo -e "${GREEN}✅ protoc-gen-go-grpc found${NC}"

    # 检查 Triple 插件
    if ! command -v protoc-gen-go-triple &> /dev/null; then
        echo -e "${YELLOW}⚠️ protoc-gen-go-triple not found, installing...${NC}"
        go install github.com/dubbogo/tools/cmd/protoc-gen-go-triple@latest
    fi
    echo -e "${GREEN}✅ protoc-gen-go-triple found${NC}"
}

# 清理旧的生成文件
cleanup() {
    echo -e "${YELLOW}Cleaning up old generated files...${NC}"
    find "$PROTO_DIR" -name "*.pb.go" -delete
    find "$PROTO_DIR" -name "*_grpc.pb.go" -delete
    find "$PROTO_DIR" -name "*.triple.go" -delete
}

# 主执行函数
main() {
    check_tools
    cleanup
}

# 执行主函数
main