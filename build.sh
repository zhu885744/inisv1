#!/bin/bash

# 设置颜色输出
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 默认参数
GOOS=$(go env GOOS)
GOARCH=$(go env GOARCH)
OUTPUT_NAME="inis"
# 如果是macOS平台，默认不压缩，其他平台默认压缩
if [ "$GOOS" = "darwin" ]; then
    COMPRESS=false
else
    COMPRESS=true
fi
COMPRESS_LEVEL=1

# 显示帮助信息
show_help() {
    echo "INIS 构建脚本"
    echo ""
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  -o, --output NAME     指定输出文件名 (默认: inis)"
    echo "  -p, --platform OS     指定目标操作系统 (默认: 当前系统)"
    echo "  -a, --arch ARCH       指定目标架构 (默认: 当前架构)"
    echo "  -c, --compress        使用UPX压缩可执行文件 (仅Linux)"
    echo "  -l, --level NUM       UPX压缩级别 (1-9, 默认: 1)"
    echo "  -h, --help            显示帮助信息"
    echo ""
    echo "示例:"
    echo "  $0 --output myapp --platform linux --arch amd64 --compress"
    exit 0
}

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
        -o|--output)
            OUTPUT_NAME="$2"
            shift 2
            ;;
        -p|--platform)
            GOOS="$2"
            shift 2
            ;;
        -a|--arch)
            GOARCH="$2"
            shift 2
            ;;
        -c|--compress)
            COMPRESS=true
            shift
            ;;
        -l|--level)
            COMPRESS_LEVEL="$2"
            shift 2
            ;;
        -h|--help)
            show_help
            ;;
        *)
            echo -e "${RED}未知选项: $1${NC}"
            show_help
            ;;
    esac
done

echo -e "${BLUE}开始构建INIS...${NC}"

# 构建时间和版本信息
BUILD_TIME=$(date "+%Y-%m-%d %H:%M:%S")
VERSION=$(grep -o 'Version = "[^"]*"' app/facade/var.go | cut -d'"' -f2)
if [ -z "$VERSION" ]; then
    VERSION="3.1.4" # 默认版本号
fi

echo -e "${BLUE}版本: ${VERSION}${NC}"
echo -e "${BLUE}构建时间: ${BUILD_TIME}${NC}"
echo -e "${BLUE}目标平台: ${GOOS}/${GOARCH}${NC}"

# 编译
echo -e "${BLUE}编译中...${NC}"

# 使用更多优化选项来减小可执行文件大小
CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} go build \
    -ldflags "-s -w -X 'inis/app/facade.Version=${VERSION}' -X 'inis/app/facade.BuildTime=${BUILD_TIME}'" \
    -trimpath \
    -o ${OUTPUT_NAME} main.go

# 检查编译是否成功
if [ $? -ne 0 ]; then
    echo -e "${RED}编译失败!${NC}"
    exit 1
fi

# 显示编译后的文件大小
BEFORE_SIZE=$(du -h ${OUTPUT_NAME} | cut -f1)
echo -e "${GREEN}编译完成! 文件大小: ${BEFORE_SIZE}${NC}"

# 如果启用压缩且不是macOS平台，则使用UPX压缩
if [ "$COMPRESS" = true ]; then
    # 检查UPX是否安装
    if ! command -v upx &> /dev/null; then
        echo -e "${YELLOW}警告: UPX未安装，跳过压缩步骤${NC}"
    else
        echo -e "${BLUE}使用UPX压缩中...${NC}"
        
        # 备份原始文件以计算压缩比
        cp ${OUTPUT_NAME} ${OUTPUT_NAME}.bak
        
        # 根据平台选择压缩参数
        if [ "$GOOS" = "darwin" ]; then
            upx -${COMPRESS_LEVEL} --best --lzma --force-macos ${OUTPUT_NAME}
        else
            upx -${COMPRESS_LEVEL} --best --lzma ${OUTPUT_NAME}
        fi
        
        # 计算并显示压缩比例
        if [ -f "${OUTPUT_NAME}.bak" ]; then
            BEFORE_SIZE=$(stat -c %s "${OUTPUT_NAME}.bak" 2>/dev/null || stat -f %z "${OUTPUT_NAME}.bak" 2>/dev/null)
            AFTER_SIZE=$(stat -c %s "${OUTPUT_NAME}" 2>/dev/null || stat -f %z "${OUTPUT_NAME}" 2>/dev/null)
            
            if [ -n "$BEFORE_SIZE" ] && [ -n "$AFTER_SIZE" ]; then
                RATIO=$(( (BEFORE_SIZE - AFTER_SIZE) * 100 / BEFORE_SIZE ))
                echo -e "${GREEN}压缩完成! 从 $(numfmt --to=iec-i --suffix=B ${BEFORE_SIZE}) 压缩到 $(numfmt --to=iec-i --suffix=B ${AFTER_SIZE}) (减少了 ${RATIO}%)${NC}"
            else
                # 如果无法计算比例，则显示简单的大小信息
                AFTER_SIZE=$(du -h ${OUTPUT_NAME} | cut -f1)
                echo -e "${GREEN}压缩完成! 压缩后大小: ${AFTER_SIZE}${NC}"
            fi
            
            # 删除备份文件
            rm ${OUTPUT_NAME}.bak
        else
            AFTER_SIZE=$(du -h ${OUTPUT_NAME} | cut -f1)
            echo -e "${GREEN}压缩完成! 压缩后大小: ${AFTER_SIZE}${NC}"
        fi
    fi
fi

echo -e "${GREEN}构建成功! 可执行文件: ./${OUTPUT_NAME}${NC}"
echo -e "${YELLOW}提示: 如果在macOS上使用UPX压缩后无法运行，请使用未压缩版本${NC}"