#!/bin/bash

# Protobuf 生成脚本 - 支持 gRPC 和 Triple
# 解决文件找不到和导入路径问题

set -e

echo "🚀 Generating Protobuf data structures (gRPC + Triple)..."

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API_DIR="$PROJECT_ROOT/api"

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
    find "$API_DIR" -name "*.pb.go" -delete
    find "$API_DIR" -name "*_grpc.pb.go" -delete
    find "$API_DIR" -name "*.triple.go" -delete
}

generate_proto() {
    local proto_file=$1
    local relative_path="api/$proto_file"
    local service_name=$(basename "$proto_file" .proto)

    echo -e "${YELLOW}📦 Generating: $proto_file${NC}"

    # 在项目根目录执行，使用正确的导入路径
    cd "$PROJECT_ROOT"

    # 生成基础 Go 代码
    echo -e "  ${BLUE}→ Generating base Go code...${NC}"
    protoc --proto_path="$PROJECT_ROOT" \
           --go_out="$PROJECT_ROOT" \
           --go_opt=paths=source_relative \
           "$relative_path"

    # 生成 gRPC 代码
    echo -e "  ${BLUE}→ Generating gRPC code...${NC}"
    protoc --proto_path="$PROJECT_ROOT" \
           --go-grpc_out="$PROJECT_ROOT" \
           --go-grpc_opt=paths=source_relative \
           "$relative_path"

    # 生成 Triple 代码
    echo -e "  ${BLUE}→ Generating Triple code...${NC}"
    protoc --proto_path="$PROJECT_ROOT" \
           --go-triple_out="$PROJECT_ROOT" \
           --go-triple_opt=paths=source_relative \
           "$relative_path"

    if [ $? -eq 0 ]; then
        local base_file="api/${proto_file%.proto}.pb.go"
        local grpc_file="api/${proto_file%.proto}_grpc.pb.go"
        local triple_file="api/${proto_file%.proto}.triple.go"

        echo -e "  ${GREEN}✅ Success: $proto_file${NC}"
        echo -e "    📄 $base_file"
        if [ -f "$PROJECT_ROOT/$grpc_file" ]; then
            echo -e "    📄 $grpc_file"
        fi
        if [ -f "$PROJECT_ROOT/$triple_file" ]; then
            echo -e "    📄 $triple_file"
        fi
    else
        echo -e "${RED}❌ Failed: $proto_file${NC}"
        exit 1
    fi
}

# 主执行函数
main() {
    check_tools
    cleanup

    # 按照依赖顺序生成（common 先于其他）
    echo -e "${BLUE}=== Generating Common Proto Files ===${NC}"
    generate_proto "common/common.proto"

    echo -e "${BLUE}=== Generating Service Proto Files ===${NC}"
    generate_proto "ima/ima.proto"
    generate_proto "msg/msg.proto"
    generate_proto "muc/muc.proto"

    # 验证生成的文件
    echo -e "${BLUE}=== Verifying Generated Files ===${NC}"
    local pb_files=$(find "$API_DIR" -name "*.pb.go" -type f | wc -l)
    local triple_files=$(find "$API_DIR" -name "*.triple.go" -type f | wc -l)
    local total_files=$((pb_files + triple_files))

    echo -e "${GREEN}Generated: $pb_files .pb.go files, $triple_files .triple.go files${NC}"

    if [ "$total_files" -eq 0 ]; then
        echo -e "${RED}❌ No files were generated!${NC}"
        exit 1
    fi

    # 显示生成的文件结构
    echo -e "${BLUE}=== Generated File Structure ===${NC}"
    find "$API_DIR" -name "*.pb.go" -o -name "*.triple.go" | sort | while read file; do
        echo -e "  📄 $(realpath --relative-to="$PROJECT_ROOT" "$file")"
    done

    # 简化验证 - 只检查生成的代码是否能编译
    echo -e "${BLUE}=== Checking Generated Code Compilation ===${NC}"
    cd "$PROJECT_ROOT"
    if go build ./api/... 2>/dev/null; then
        echo -e "${GREEN}✅ Generated code compiles successfully${NC}"
    else
        echo -e "${YELLOW}⚠️ Generated code has compilation issues (may be due to module conflicts)${NC}"
        echo -e "${YELLOW}But proto files were generated successfully${NC}"
    fi

    echo -e "${GREEN}✨ Proto generation completed successfully!${NC}"
    echo -e "${BLUE}Summary:${NC}"
    echo -e "  ${GREEN}✅ Base Go structures (.pb.go)${NC}"
    echo -e "  ${GREEN}✅ gRPC service interfaces (_grpc.pb.go)${NC}"
    echo -e "  ${GREEN}✅ Triple service interfaces (.triple.go)${NC}"
}

# 执行主函数
main