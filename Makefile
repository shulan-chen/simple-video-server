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

# 数据库Migration配置
DB_USER ?= yanghao
DB_PWD ?= 123456
DB_HOST ?= 139.196.242.169
DB_PORT ?= 3306
DB_NAME ?= video_server
DB_URL = mysql://$(DB_USER):$(DB_PWD)@tcp($(DB_HOST):$(DB_PORT))/$(DB_NAME)?charset=utf8mb4&parseTime=True
MIGRATION_PATH = ./migrations

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

#不编译后台启动所以服务
.PHONY: strat-without-compile
start-without-compile:
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
	@echo "API服务:" && \
		if curl -s -f http://localhost:8000/health/ready > /dev/null 2>&1; then \
			echo "✅ API服务正常"; \
		else \
			echo "❌ API服务不可用"; \
		fi
	@echo "Web服务:" && \
		if curl -s -f http://localhost:8080/health/ready > /dev/null 2>&1; then \
			echo "✅ Web服务正常"; \
		else \
			echo "❌ Web服务不可用"; \
		fi
	@echo "Stream服务:" && \
		if curl -s -f http://localhost:9090/health/ready > /dev/null 2>&1; then \
			echo "✅ Stream服务正常"; \
		else \
			echo "❌ Stream服务不可用"; \
		fi
	@echo "Scheduler服务:" && \
		if curl -s -f http://localhost:8001/health/ready > /dev/null 2>&1; then \
			echo "✅ Scheduler服务正常"; \
		else \
			echo "❌ Scheduler服务不可用"; \
		fi

# ========== Swagger文档命令 ==========

# 生成Swagger API文档
.PHONY: swagger
swagger:
	@echo "📖 生成Swagger文档..."
	@if command -v swag &> /dev/null; then \
		swag init --generalInfo api.go --dir ./api --output ./api/docs --parseDependency --parseInternal; \
	elif [ -f ~/Desktop/GO_projects/bin/swag ]; then \
		~/Desktop/GO_projects/bin/swag init --generalInfo api.go --dir ./api --output ./api/docs --parseDependency --parseInternal; \
	elif [ -f $(go env GOPATH)/bin/swag ]; then \
		$(go env GOPATH)/bin/swag init --generalInfo api.go --dir ./api --output ./api/docs --parseDependency --parseInternal; \
	else \
		echo "❌ swag工具未安装，请先安装:"; \
		echo "   go install github.com/swaggo/swag/cmd/swag@v1.8.12"; \
		exit 1; \
	fi
	@echo "✅ Swagger文档生成完成！"
	@echo "🌐 启动服务后访问: http://localhost:8000/swagger/index.html"

# 检查swag工具是否已安装
.PHONY: swagger-check
swagger-check:
	@if command -v swag &> /dev/null; then \
		echo "✅ swag已安装，版本: $$(swag --version)"; \
	else \
		echo "❌ swag未安装"; \
		echo "安装命令: go install github.com/swaggo/swag/cmd/swag@v1.8.12"; \
	fi

# ========== 一次性迁移工具 ==========

# 将数据库中的预签名封面URL迁移为永久公开URL（只需执行一次）
.PHONY: fix-thumbnails
fix-thumbnails:
	@echo "🔧 迁移封面URL（预签名 → 永久公开）..."
	go run ./cmd/tools/fix_thumbnails/main.go
	@echo "✅ 迁移完成！"

# ========== 监控命令 ==========

# 启动监控组件（Prometheus + Grafana）
.PHONY: monitor-start
monitor-start:
	@echo "启动监控组件..."
	docker-compose up -d prometheus grafana
	@echo "✅ 监控组件已启动！"
	@echo "  - Prometheus: http://localhost:9091"
	@echo "  - Grafana:    http://localhost:3000  (admin / admin123)"

# 停止监控组件
.PHONY: monitor-stop
monitor-stop:
	docker-compose stop prometheus grafana

# 查看监控状态
.PHONY: monitor-health
monitor-health:
	@echo "Prometheus:" && \
		if curl -s -f http://localhost:9091/-/ready > /dev/null 2>&1; then \
			echo "✅ Prometheus 正常"; \
		else \
			echo "❌ Prometheus 不可用"; \
		fi
	@echo "Grafana:" && \
		if curl -s -f http://localhost:3000/api/health > /dev/null 2>&1; then \
			echo "✅ Grafana 正常"; \
		else \
			echo "❌ Grafana 不可用"; \
		fi

# ========== 数据库Migration命令 ==========

# 升级数据库到最新版本
.PHONY: migrate-up
migrate-up:
	@echo "📤 升级数据库..."
	@migrate -path $(MIGRATION_PATH) -database "$(DB_URL)" up
	@echo "✅ 数据库升级完成！"

# 回滚一个版本
.PHONY: migrate-down
migrate-down:
	@echo "📥 回滚数据库..."
	@migrate -path $(MIGRATION_PATH) -database "$(DB_URL)" down 1
	@echo "✅ 数据库回滚完成！"

# 创建新的migration文件
.PHONY: migrate-create
migrate-create:
	@read -p "Migration名称（例如：add_email_to_users）: " name; \
	migrate create -ext sql -dir $(MIGRATION_PATH) -seq $$name
	@echo "✅ Migration文件已创建！"

# 查看当前数据库版本
.PHONY: migrate-version
migrate-version:
	@echo "当前数据库版本："
	@migrate -path $(MIGRATION_PATH) -database "$(DB_URL)" version

# 强制设置数据库版本（慎用！）
.PHONY: migrate-force
migrate-force:
	@read -p "⚠️  强制设置版本号: " version; \
	migrate -path $(MIGRATION_PATH) -database "$(DB_URL)" force $$version
	@echo "✅ 版本已强制设置！"

# 显示帮助信息
.PHONY: help
help:
	@echo "=== 构建命令 ==="
	@echo "  make build          - 构建所有服务"
	@echo "  make build-api      - 仅构建API服务"
	@echo "  make build-web      - 仅构建Web服务"
	@echo "  make build-stream   - 仅构建Stream服务"
	@echo "  make build-scheduler- 仅构建Scheduler服务"
	@echo ""
	@echo "=== 运行命令 ==="
	@echo "  make run-api        - 运行API服务（前台）"
	@echo "  make run-web        - 运行Web服务（前台）"
	@echo "  make run-stream     - 运行Stream服务（前台）"
	@echo "  make run-scheduler  - 运行Scheduler服务（前台）"
	@echo "  make start          - 后台启动所有服务"
	@echo "  make stop           - 停止所有服务"
	@echo "  make restart        - 重启所有服务"
	@echo ""
	@echo "=== 监控 ==="
	@echo "  make monitor-start  - 启动 Prometheus + Grafana"
	@echo "  make monitor-stop   - 停止监控组件"
	@echo "  make monitor-health - 检查监控组件健康状态"
	@echo ""
	@echo "=== 数据库Migration ==="
	@echo "  make migrate-up     - 升级数据库到最新版本"
	@echo "  make migrate-down   - 回滚一个版本"
	@echo "  make migrate-create - 创建新的migration文件"
	@echo "  make migrate-version- 查看当前数据库版本"
	@echo "  make migrate-force  - 强制设置版本（慎用）"
	@echo ""
	@echo "=== Swagger文档 ==="
	@echo "  make swagger        - 生成API文档"
	@echo "  make swagger-check  - 检查swag工具是否已安装"
	@echo ""
	@echo "=== 开发工具 ==="
	@echo "  make test           - 运行测试"
	@echo "  make fmt            - 格式化代码"
	@echo "  make vet            - 静态代码检查"
	@echo "  make tidy           - 整理依赖"
	@echo "  make health         - 检查服务健康状态"
	@echo "  make clean          - 清理编译产物"
