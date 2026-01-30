# Python 分析服务 Makefile

.PHONY: help build push run stop clean test

# 默认目标
.DEFAULT_GOAL := help

# 配置
IMAGE_NAME := ddhdocker/trading-analysis
VERSION ?= latest

help: ## 显示帮助信息
	@echo "Python 分析服务 - 可用命令："
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo ""

setup: ## 设置 Docker buildx（首次使用需要）
	@echo "设置 Docker buildx..."
	@if ! docker buildx inspect amd64-builder > /dev/null 2>&1; then \
		docker buildx create --name amd64-builder --use; \
		echo "✅ buildx builder 创建成功"; \
	else \
		docker buildx use amd64-builder; \
		echo "✅ buildx builder 已存在"; \
	fi
	@docker buildx inspect --bootstrap

build: ## 构建 Docker 镜像（AMD64）
	@echo "构建 AMD64 镜像: $(IMAGE_NAME):$(VERSION)"
	@if ! docker buildx inspect amd64-builder > /dev/null 2>&1; then \
		echo "创建 buildx builder..."; \
		docker buildx create --name amd64-builder --use; \
	else \
		docker buildx use amd64-builder; \
	fi
	docker buildx build \
		--platform linux/amd64 \
		--tag $(IMAGE_NAME):$(VERSION) \
		--load \
		.

build-arm64: ## 构建 Docker 镜像（ARM64/Apple Silicon）
	@echo "构建 ARM64 镜像: $(IMAGE_NAME):$(VERSION)"
	@if ! docker buildx inspect arm64-builder > /dev/null 2>&1; then \
		echo "创建 buildx builder..."; \
		docker buildx create --name arm64-builder --use; \
	else \
		docker buildx use arm64-builder; \
	fi
	docker buildx build \
		--platform linux/arm64 \
		--tag $(IMAGE_NAME):$(VERSION) \
		--load \
		.

push: ## 推送镜像到 Docker Hub（AMD64）
	@echo "推送 AMD64 镜像: $(IMAGE_NAME):$(VERSION)"
	@if ! docker buildx inspect amd64-builder > /dev/null 2>&1; then \
		echo "创建 buildx builder..."; \
		docker buildx create --name amd64-builder --use; \
	else \
		docker buildx use amd64-builder; \
	fi
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--tag $(IMAGE_NAME):$(VERSION) \
		--push \
		.
	@if [ "$(VERSION)" != "latest" ]; then \
		docker buildx build \
			--platform linux/amd64,linux/arm64 \
			--tag $(IMAGE_NAME):latest \
			--push \
			.; \
		echo "已推送: $(IMAGE_NAME):latest"; \
	fi

build-push: build push ## 构建并推送镜像

run: ## 本地运行服务
	@echo "启动服务..."
	uvicorn main:app --reload --host 0.0.0.0 --port 8000

run-docker: ## 使用 Docker 运行服务
	@echo "使用 Docker 运行服务..."
	docker run -d \
		--name trading-analysis \
		-p 8000:8000 \
		-e MYSQL_HOST=host.docker.internal \
		-e MYSQL_PORT=3306 \
		-e MYSQL_USER=trading \
		-e MYSQL_PASSWORD=your_password \
		-e MYSQL_DB=trading_analysis \
		$(IMAGE_NAME):$(VERSION)

stop: ## 停止 Docker 容器
	@echo "停止容器..."
	docker stop trading-analysis || true
	docker rm trading-analysis || true

clean: ## 清理镜像和容器
	@echo "清理..."
	docker stop trading-analysis || true
	docker rm trading-analysis || true
	docker rmi $(IMAGE_NAME):$(VERSION) || true

test: ## 运行测试
	@echo "运行测试..."
	pytest -v

install: ## 安装依赖
	@echo "安装依赖..."
	pip install -r requirements.txt

logs: ## 查看容器日志
	docker logs -f trading-analysis

shell: ## 进入容器 shell
	docker exec -it trading-analysis /bin/bash

health: ## 健康检查
	@curl -s http://localhost:8000/health | python -m json.tool

# 版本发布
release: ## 发布新版本 (使用: make release VERSION=0.1.0)
	@if [ -z "$(VERSION)" ] || [ "$(VERSION)" = "latest" ]; then \
		echo "错误: 请指定版本号，例如: make release VERSION=0.1.0"; \
		exit 1; \
	fi
	@echo "发布版本: $(VERSION)"
	@make build VERSION=$(VERSION)
	@make push VERSION=$(VERSION)
	@echo "✅ 版本 $(VERSION) 发布完成！"
