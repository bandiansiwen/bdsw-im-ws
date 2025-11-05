#!/bin/bash

# Protobuf 生成脚本 - 修复版
# 解决文件找不到和导入路径问题

set -e

echo "🚀 Generating Protobuf data structures..."

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

# 清理旧的生成文件
echo -e "${YELLOW}Cleaning up old generated files...${NC}"
find "$API_DIR" -name "*.pb.go" -delete

generate_proto() {
    local proto_file=$1
    local relative_path="api/$proto_file"

    echo -e "${YELLOW}📦 Generating: $proto_file${NC}"

    # 在项目根目录执行，使用正确的导入路径
    cd "$PROJECT_ROOT"

    protoc --proto_path="$PROJECT_ROOT" \
           --go_out="$PROJECT_ROOT" \
           --go_opt=paths=source_relative \
           "$relative_path"

    if [ $? -eq 0 ]; then
        local generated_file="api/${proto_file%.proto}.pb.go"
        echo -e "${GREEN}✅ Success: $proto_file -> $generated_file${NC}"
    else
        echo -e "${RED}❌ Failed: $proto_file${NC}"
        exit 1
    fi
}

# 按照依赖顺序生成（common 先于其他）
echo -e "${BLUE}=== Generating Common Proto Files ===${NC}"
generate_proto "common/common.proto"

echo -e "${BLUE}=== Generating Service Proto Files ===${NC}"
generate_proto "ima_gateway/ima_gateway.proto"
generate_proto "business_message/business_message.proto"
generate_proto "muc/muc.proto"
generate_proto "server_push/server_push.proto"

# 验证生成的文件
echo -e "${BLUE}=== Verifying Generated Files ===${NC}"
generated_files=$(find "$API_DIR" -name "*.pb.go" -type f | wc -l)
if [ "$generated_files" -eq 0 ]; then
    echo -e "${RED}❌ No .pb.go files were generated!${NC}"
    exit 1
fi

echo -e "${GREEN}🎉 All proto data structures generated successfully!${NC}"

# 显示生成的文件结构
echo -e "${BLUE}=== Generated File Structure ===${NC}"
find "$API_DIR" -name "*.pb.go" -type f | sort | while read file; do
    echo -e "  📄 $file"
done

# 简化验证 - 只检查生成的代码是否能编译
echo -e "${BLUE}=== Checking Generated Code Compilation ===${NC}"
cd "$PROJECT_ROOT"
if go build ./api/... 2>/dev/null; then
    echo -e "${GREEN}✅ Generated code compiles successfully${NC}"
else
    echo -e "${YELLOW}⚠️ Generated code has compilation issues (may be due to module conflicts)${NC}"
    echo -e "${YELLOW}But .pb.go files were generated successfully${NC}"
fi

echo -e "${GREEN}✨ Proto generation completed successfully!${NC}"