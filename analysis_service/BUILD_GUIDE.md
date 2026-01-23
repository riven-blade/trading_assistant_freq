# 多平台镜像构建指南

## 问题说明

在 Mac M1/M2 (ARM64) 上构建的镜像无法在 AMD64 服务器上运行，会出现平台不匹配错误：
```
The requested image's platform (linux/arm64) does not match the detected host platform (linux/amd64)
```

## 解决方案：构建多平台镜像

使用 Docker Buildx 构建同时支持 AMD64 和 ARM64 的镜像。

## 首次使用（一次性设置）

### 1. 设置 buildx builder

```bash
cd analysis_service

# 方法1：使用 Makefile
make setup

# 方法2：手动设置
docker buildx create --name multiplatform --use
docker buildx inspect --bootstrap
```

## 构建和推送

### 方法1：使用 Makefile（推荐）

```bash
cd analysis_service

# 构建并推送多平台镜像
make push

# 或者分步执行
make build  # 构建
make push   # 推送
```

### 方法2：使用脚本

```bash
cd analysis_service

# 构建并推送 latest
./build.sh

# 构建并推送指定版本
./build.sh 0.1.0
```

### 方法3：手动命令

```bash
cd analysis_service

# 确保使用 buildx builder
docker buildx use multiplatform

# 构建并推送多平台镜像
docker buildx build \
    --platform linux/amd64,linux/arm64 \
    --tag ddhdocker/trading-analysis:latest \
    --push \
    .
```

## 验证镜像

### 查看镜像支持的平台

```bash
docker buildx imagetools inspect ddhdocker/trading-analysis:latest
```

输出应该包含：
```
MediaType: application/vnd.docker.distribution.manifest.list.v2+json
Digest:    sha256:...

Manifests:
  Name:      linux/amd64
  MediaType: application/vnd.docker.distribution.manifest.v2+json
  Platform:  linux/amd64

  Name:      linux/arm64
  MediaType: application/vnd.docker.distribution.manifest.v2+json
  Platform:  linux/arm64
```

## 部署使用

在任何平台（AMD64 或 ARM64）上都可以直接使用：

```bash
# 拉取镜像（自动选择匹配的平台）
docker pull ddhdocker/trading-analysis:latest

# 运行容器
docker run -d -p 8000:8000 ddhdocker/trading-analysis:latest

# 或使用 docker-compose
cd deploy
docker-compose up -d analysis-service
```

## 常见问题

### Q: buildx 命令不存在？
A: 更新 Docker Desktop 到最新版本，buildx 已内置。

### Q: 构建很慢？
A: 多平台构建需要模拟不同架构，第一次会慢一些。后续构建会使用缓存。

### Q: 推送失败？
A: 确保已登录 Docker Hub：
```bash
docker login
```

### Q: 如何只构建 AMD64？
A: 修改 `--platform` 参数：
```bash
docker buildx build \
    --platform linux/amd64 \
    --tag ddhdocker/trading-analysis:latest \
    --push \
    .
```

## 性能说明

- **构建时间**：多平台构建比单平台慢 1.5-2 倍
- **镜像大小**：多平台镜像本身不会更大，Docker 会自动选择匹配的平台
- **运行性能**：运行时性能完全一致，没有任何损失

## 推荐工作流

1. **开发阶段**：本地构建单平台镜像测试
   ```bash
   docker build -t trading-analysis:dev .
   ```

2. **发布阶段**：构建多平台镜像推送
   ```bash
   make push
   ```

3. **生产部署**：直接拉取使用
   ```bash
   docker-compose up -d analysis-service
   ```
