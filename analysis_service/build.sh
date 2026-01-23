#!/bin/bash

# Python 分析服务镜像构建和推送脚本（AMD64）

set -e

# 配置
IMAGE_NAME="ddhdocker/trading-analysis"
VERSION=${1:-"latest"}  # 默认 latest，可以通过参数指定版本

echo "=========================================="
echo "构建 Python 分析服务镜像（AMD64）"
echo "镜像名称: ${IMAGE_NAME}:${VERSION}"
echo "平台: linux/amd64"
echo "=========================================="

# 检查 buildx 是否可用
if ! docker buildx version > /dev/null 2>&1; then
    echo "错误: docker buildx 不可用"
    echo "请运行: docker buildx create --use"
    exit 1
fi

# 1. 创建并使用 buildx builder（如果不存在）
echo ""
echo "步骤 1/3: 设置 buildx builder..."
if ! docker buildx inspect amd64-builder > /dev/null 2>&1; then
    docker buildx create --name amd64-builder --use
else
    docker buildx use amd64-builder
fi

# 2. 构建并推送 AMD64 镜像
echo ""
echo "步骤 2/3: 构建并推送 AMD64 镜像..."
docker buildx build \
    --platform linux/amd64 \
    --tag ${IMAGE_NAME}:${VERSION} \
    --push \
    .

# 如果指定了版本号，同时推送 latest 标签
if [ "$VERSION" != "latest" ]; then
    echo ""
    echo "同时推送 latest 标签..."
    docker buildx build \
        --platform linux/amd64 \
        --tag ${IMAGE_NAME}:latest \
        --push \
        .
fi

# 3. 完成
echo ""
echo "=========================================="
echo "✅ AMD64 镜像构建和推送完成！"
echo "镜像: ${IMAGE_NAME}:${VERSION}"
if [ "$VERSION" != "latest" ]; then
    echo "镜像: ${IMAGE_NAME}:latest"
fi
echo "平台: linux/amd64"
echo "=========================================="
echo ""
echo "使用方法："
echo "  docker pull ${IMAGE_NAME}:${VERSION}"
echo "  docker run -d -p 8000:8000 ${IMAGE_NAME}:${VERSION}"
echo ""
