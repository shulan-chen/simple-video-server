# 为什么需要Makefile？
# 1. 统一构建命令（不用记复杂的go build参数）
# 2. 支持批量操作（一次构建所有服务）
# 3. 方便CI/CD集成
# 4. 提供开发便利命令（test、fmt、lint）

# 变量定义
BINARY_DIR=bin
API_BINARY=$(BINARY_DIR)/api-service
WEB_BINARY=$(BINARY_DIR)/web-service
STREAM_BINARY=$(BINARY_DIR)/stream-service
SCHEDULER_BINARY=$(BINARY_DIR)/scheduler-service

# Go编译参数
# -ldflags "-s -w" 可以减小二进制文件大小（去除调试信息）
GO_BUILD_FLAGS=-ldflags "-s -w"

# 默认目标：构建所有服务
.PHONY: all
all: build

# 创建输出目录
$(BINARY_DIR):
	mkdir -p $(BINARY_DIR)

# 构建API服务
.PHONY: build-api
build-api: $(BINARY_DIR)
	@echo "构建API服务..."
	go build $(GO_BUILD_FLAGS) -o $(API_BINARY) ./cmd/api

# 构建Web服务
.PHONY: build-web
build-web: $(BINARY_DIR)
	@echo "构建Web服务..."
	go build $(GO_BUILD_FLAGS) -o $(WEB_BINARY) ./cmd/web

# 构建Stream服务
.PHONY: build-stream
build-stream: $(BINARY_DIR)
	@echo "构建Stream服务..."
	go build $(GO_BUILD_FLAGS) -o $(STREAM_BINARY) ./cmd/stream

# 构建Scheduler服务
.PHONY: build-scheduler
build-scheduler: $(BINARY_DIR)
	@echo "构建Scheduler服务..."
	go build $(GO_BUILD_FLAGS) -o $(SCHEDULER_BINARY) ./cmd/scheduler

# 构建所有服务
.PHONY: build
build: build-api build-web build-stream build-scheduler
	@echo "✅ 所有服务构建完成！"

# 运行API服务
.PHONY: run-api
run-api:
	@echo "启动API服务..."
	go run ./cmd/api/main.go

# 运行Web服务
.PHONY: run-web
run-web:
	@echo "启动Web服务..."
	go run ./cmd/web/main.go

# 运行Stream服务
.PHONY: run-stream
run-stream:
	@echo "启动Stream服务..."
	go run ./cmd/stream/main.go

# 运行Scheduler服务
.PHONY: run-scheduler
run-scheduler:
	@echo "启动Scheduler服务..."
	go run ./cmd/scheduler/main.go

# 后台运行所有服务（使用构建后的二进制）
.PHONY: start
start: build
	@echo "后台启动所有服务..."
	nohup $(API_BINARY) > logs/api.log 2>&1 &
	nohup $(WEB_BINARY) > logs/web.log 2>&1 &
	nohup $(STREAM_BINARY) > logs/stream.log 2>&1 &
	nohup $(SCHEDULER_BINARY) > logs/scheduler.log 2>&1 &
	@echo "✅ 所有服务已后台启动！"
	@echo "📄 查看日志："
	@echo "  - API服务: tail -f logs/api.log"
	@echo "  - Web服务: tail -f logs/web.log"
	@echo "  - Stream服务: tail -f logs/stream.log"
	@echo "  - Scheduler服务: tail -f logs/scheduler.log"

# 停止所有服务
.PHONY: stop
stop:
	@echo "停止所有服务..."
	-pkill -f api-service
	-pkill -f web-service
	-pkill -f stream-service
	-pkill -f scheduler-service
	@echo "✅ 所有服务已停止！"

# 重启所有服务
.PHONY: restart
restart: stop start

# 清理编译产物
.PHONY: clean
clean:
	@echo "清理编译产物..."
	rm -rf $(BINARY_DIR)
	rm -f logs/*.log
	@echo "✅ 清理完成！"

# 运行测试
.PHONY: test
test:
	@echo "运行测试..."
	go test -v ./...

# 代码格式化
.PHONY: fmt
fmt:
	@echo "格式化代码..."
	go fmt ./...

# 代码检查
.PHONY: vet
vet:
	@echo "代码静态检查..."
	go vet ./...

# 依赖管理
.PHONY: tidy
tidy:
	@echo "整理依赖..."
	go mod tidy

# 健康检查
.PHONY: health
health:
	@echo "检查所有服务健康状态..."
	@echo "API服务:" && curl -s http://localhost:8000/health/ready | jq . || echo "❌ API服务不可用"
	@echo "Web服务:" && curl -s http://localhost:8080/health/ready | jq . || echo "❌ Web服务不可用"
	@echo "Stream服务:" && curl -s http://localhost:9090/health/ready | jq . || echo "❌ Stream服务不可用"
	@echo "Scheduler服务:" && curl -s http://localhost:8001/health/ready | jq . || echo "❌ Scheduler服务不可用"

# 显示帮助信息
.PHONY: help
help:
	@echo "可用命令："
	@echo "  make build          - 构建所有服务"
	@echo "  make build-api      - 仅构建API服务"
	@echo "  make build-web      - 仅构建Web服务"
	@echo "  make build-stream   - 仅构建Stream服务"
	@echo "  make build-scheduler- 仅构建Scheduler服务"
	@echo "  make run-api        - 运行API服务（前台）"
	@echo "  make run-web        - 运行Web服务（前台）"
	@echo "  make run-stream     - 运行Stream服务（前台）"
	@echo "  make run-scheduler  - 运行Scheduler服务（前台）"
	@echo "  make start          - 后台启动所有服务"
	@echo "  make stop           - 停止所有服务"
	@echo "  make restart        - 重启所有服务"
	@echo "  make test           - 运行测试"
	@echo "  make fmt            - 格式化代码"
	@echo "  make vet            - 静态代码检查"
	@echo "  make tidy           - 整理依赖"
	@echo "  make health         - 检查服务健康状态"
	@echo "  make clean          - 清理编译产物"
