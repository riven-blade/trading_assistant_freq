# Trading Assistant 一键打包 Makefile

.PHONY: all clean build-frontend build-backend build-backend-linux package dev install-deps docker-build docker-buildx docker-run docker-stop docker-logs docker-shell docker-clean docker-deploy help

# Docker 相关变量
IMAGE_NAME := ddhdocker/trading-assistant-freq
IMAGE_TAG := v0.0.98
CONTAINER_NAME := trading-assistant-freq

# 默认目标
all: package

# 安装依赖
install-deps:
	@echo "🔧 安装后端依赖..."
	go mod download
	@echo "🔧 安装前端依赖..."
	cd web && npm install

# 清理构建文件
clean:
	@echo "🧹 清理构建文件..."
	rm -rf web/build/
	rm -rf dist/
	go clean

# 构建前端
build-frontend:
	@echo "🏗️  构建前端项目..."
	cd web && npm run build
	@echo "✅ 前端构建完成"

build-backend: build-frontend
	@echo "🏗️  构建 Linux AMD64 后端项目..."
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/trading_assistant .
	@echo "✅ Linux AMD64 后端构建完成"

# 一键打包
package: build-backend
	@echo "📦 打包完成！"
	@echo "👉 执行文件位于: bin/trading_assistant"
	@echo "👉 启动服务: ./bin/trading_assistant"

# 开发模式启动
dev:
	@echo "🚀 启动开发模式..."
	@echo "启动前端开发服务器..."
	cd web && npm start &
	@echo "等待2秒后启动后端..."
	sleep 2
	@echo "启动后端服务..."
	go run .

# 生产模式启动
start: package
	@echo "🚀 启动生产环境..."
	./dist/trading_assistant

# Docker 构建镜像
docker-build: build-backend-linux
	@echo "🐳 构建 Docker 镜像..."
	docker build --platform linux/amd64 -t $(IMAGE_NAME):$(IMAGE_TAG) .
	@echo "✅ Docker 镜像构建完成: $(IMAGE_NAME):$(IMAGE_TAG)"

# 帮助信息
help:
	@echo "Trading Assistant Makefile 使用说明:"
	@echo ""
	@echo "📦 基础构建命令:"
	@echo "  make install-deps   - 安装前后端依赖"
	@echo "  make clean          - 清理构建文件"
	@echo "  make build-frontend - 仅构建前端"
	@echo "  make build-backend  - 构建后端（包含前端）"
	@echo "  make package        - 一键打包（默认）"
	@echo ""
	@echo "🚀 运行命令:"
	@echo "  make dev            - 开发模式启动"
	@echo "  make start          - 生产模式启动"
	@echo ""
	@echo "🐳 Docker 命令:"
	@echo "  make docker-build      - 构建 Docker 镜像 (linux/amd64)"
	@echo "  make docker-buildx     - 构建多架构镜像并推送"
	@echo "  make docker-run        - 运行 Docker 容器"
	@echo "  make docker-stop       - 停止 Docker 容器"
	@echo "  make docker-logs       - 查看容器日志"
	@echo "  make docker-shell      - 进入容器 shell"
	@echo "  make docker-clean      - 清理 Docker 镜像"
	@echo "  make docker-deploy     - 完整 Docker 部署"
	@echo "  make build-backend-linux - 构建 Linux AMD64 后端"
	@echo ""
	@echo "  make help           - 显示此帮助信息"
	@echo ""
	@echo "📋 使用示例:"
	@echo "  本地开发: make install-deps && make dev"
	@echo "  本地构建: make package && make start"
	@echo "  Docker部署: make docker-deploy"


#scp ./x6.py root@168.93.214.185:/root/service/data/strategies