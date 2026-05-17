# 为什么使用多阶段构建？
# 1. 减小镜像大小（只包含运行时需要的文件）
# 2. 不包含构建工具（Go编译器等）
# 3. 提高安全性（攻击面更小）
# 4. 加快部署速度（镜像小，传输快）

# ========== 构建阶段 ==========
FROM golang:1.24-alpine AS builder

# 安装必要的工具
RUN apk add --no-cache git make

# 设置工作目录
WORKDIR /build

# 复制依赖文件（利用Docker缓存）
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建参数：指定要构建的服务
ARG SERVICE_NAME
ENV SERVICE_NAME=${SERVICE_NAME}

# 构建二进制文件
# CGO_ENABLED=0: 禁用CGO，生成静态链接的二进制（可以在alpine中运行）
# -ldflags "-s -w": 去除调试信息，减小二进制大小
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "-s -w" \
    -o /build/service \
    ./cmd/${SERVICE_NAME}/main.go

# ========== 运行阶段 ==========
FROM alpine:latest

# 安装运行时依赖
# ca-certificates: HTTPS请求需要
# tzdata: 时区数据
# curl: 健康检查需要
RUN apk add --no-cache ca-certificates tzdata curl

# 设置时区
ENV TZ=Asia/Shanghai

# 创建非root用户（安全最佳实践）
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/service /app/service

# 复制配置文件
COPY --chown=appuser:appuser config /app/config

# 创建日志目录
RUN mkdir -p /app/logs && chown -R appuser:appuser /app

# 切换到非root用户
USER appuser

# 暴露端口（根据不同服务会有不同）
# 注意：这只是文档作用，实际端口由docker-compose指定
EXPOSE 8000 8080 9090 8001

# 健康检查（默认配置，可被docker-compose覆盖）
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8000/health/ready || exit 1

# 启动服务
CMD ["/app/service"]

# 镜像大小对比：
# 不使用多阶段构建：~800MB（包含Go编译器）
# 使用多阶段构建：~20MB（只包含运行时）
