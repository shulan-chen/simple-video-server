# CLAUDE.md

此文件为 Claude Code (claude.ai/code) 在此代码仓库中工作时提供指导。

## 项目概述

这是一个使用 Go 构建的视频流服务器，采用**真正的微服务架构**，包含四个独立部署的服务。应用使用 Gin 框架、GORM 进行数据库操作、JWT 进行身份验证、Redis 进行会话缓存，以及阿里云 OSS 进行视频存储。

## 架构（微服务版本）

系统由四个**独立进程**的服务组成，每个服务：
- 有独立的 `main.go` 入口（在 `cmd/` 目录下）
- 可以独立编译、部署、扩容
- 有独立的健康检查和优雅关闭
- 故障隔离（一个服务崩溃不影响其他服务）

### 服务列表

1. **API 服务** (`cmd/api/main.go`, 端口 8000)
   - 用户认证（JWT）、视频元数据、评论管理
   - 依赖：MySQL、Redis

2. **Web 服务** (`cmd/web/main.go`, 端口 8080)
   - 提供 HTML 模板的前端服务
   - 代理到 API 和 Stream 服务

3. **Stream 服务** (`cmd/stream/main.go`, 端口 9090)
   - 视频上传/下载、OSS 集成
   - 连接限流（默认10个并发）

4. **Scheduler 服务** (`cmd/scheduler/main.go`, 端口 8001)
   - 后台任务调度（延迟删除视频）
   - 提供管理接口
   - 依赖：MySQL

### 核心组件

- **api/**: 用户身份验证 (JWT)、视频/评论 CRUD 操作、会话管理
- **web/**: 前端处理器和后端服务 API 代理
- **stream/**: 视频流、OSS 客户端、连接限流器
- **scheduler/**: 异步任务执行的 Worker 模式（目前处理视频清理）
- **config/**: 基于 Viper 的配置管理，从 `config/config.json` 读取
- **api/dbops/**: GORM 数据库连接和操作
- **api/session/**: 混合会话管理（sync.Map + Redis 实现分布式一致性）

## 构建和运行

### 方式1：使用 Makefile（推荐）

```bash
# 构建所有服务
make build

# 构建单个服务
make build-api
make build-web
make build-stream
make build-scheduler

# 运行单个服务（前台，用于调试）
make run-api
make run-web
make run-stream
make run-scheduler

# 后台启动所有服务
make start

# 停止所有服务
make stop

# 检查服务健康状态
make health
```

### 方式2：使用 Docker Compose（模拟生产环境）

```bash
# 一键启动所有服务（包括 MySQL 和 Redis）
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f api-service

# 停止服务
docker-compose down

# 扩容某个服务
docker-compose up -d --scale stream-service=3
```

### 方式3：手动启动（了解底层原理）

```bash
# 构建
go build -o bin/api-service ./cmd/api/main.go

# 运行
./bin/api-service
```

## 测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./api/dbops
go test ./scheduler

# 运行并显示详细输出
go test -v ./...
```

## 配置

应用需要 `config/config.json` 文件，结构如下：

```json
{
    "lb_addr": ":8080",
    "oss_addr": "oss-cn-shanghai.aliyuncs.com",
    "oss_region": "cn-shanghai",
    "oss_key": "YOUR_OSS_KEY",
    "oss_secret": "YOUR_OSS_SECRET",
    "oss_bucket": "bucket-name",
    "db_addr": "host:3306",
    "db_user": "username",
    "db_pwd": "password",
    "db_name": "video_server",
    "video_delete_delay_time": 300,
    "redis_addr": "host:6379",
    "redis_pwd": "",
    "redis_db": 0
}
```

**注意**: `config/config.json` 已被 gitignore。配置可以通过环境变量覆盖（Viper 的 AutomaticEnv 功能）。

## 数据库表结构

应用使用 MySQL 和 GORM。主要数据表：

- **users**: 用户账号（id, name, password, isVaild, created_at）
- **video_info**: 视频元数据（id, vid, author_id, name, create_time, click_count）
- **comments**: 视频评论（id, comment_id, video_id, author_id, content, create_time）
- **video_delete_record**: 待删除的视频（id, vid）- 由 scheduler 处理
- **sessions**: 会话存储（id, session_id, user_id, username, ttl）

## 认证流程

1. **注册/登录**: 用户凭据 → 生成 JWT token → Token 作为 session 存储在 Redis + sync.Map 中
2. **验证**: 中间件检查 `X-Session-Id` 请求头 → 解析 JWT → 向请求头添加 `X-User-Name` 和 `X-User-Id`
3. **会话存储**: 混合方式，使用本地 sync.Map（快速）+ Redis（分布式一致性）
4. **豁免**: `/user` POST（注册）和 `/user/:username` POST（登录）绕过认证

## 视频工作流

1. **上传**: POST 到 `/videos/upload/:vid-id` (stream 服务) → 上传到 OSS
2. **元数据**: POST 到 `/user/:user_name/videos` (api 服务) → 在数据库中存储视频信息
3. **流式传输**: GET `/videos/:vid-id` (stream 服务) → 生成预签名 OSS URL（1小时有效期）
4. **删除**: DELETE `/user/:user_name/videos/:vid` → 添加到 `video_delete_record` → Scheduler 延迟后从 OSS 删除

## 端口映射

- **8000**: API 服务（用户/视频/评论接口）
- **8080**: Web 服务（HTML 模板，代理到 API/Stream）
- **9090**: Stream 服务（视频上传/下载，带限流）

## 开发注意事项

### 微服务相关
- **每个服务独立部署**：修改一个服务不影响其他服务
- **健康检查**：每个服务提供 `/health/live`、`/health/ready`、`/health/startup` 三个端点
- **优雅关闭**：收到 SIGTERM/SIGINT 后，会等待现有请求处理完（默认30秒超时）
- **配置管理**：使用 `internal/config` 统一管理，支持环境变量覆盖
- **服务通信**：当前是直接HTTP调用，后续可改为 gRPC 或消息队列

### 代码规范
- 所有服务使用 Gin 框架的 `gin.Default()` 中间件（日志 + 恢复）
- 日志使用 zap logger（通过 `utils.InitLogging()` 初始化）
- Stream 服务实现连接限流（默认 10 个并发连接）
- 通过 `utils.NewUUID()` 生成会话 ID 和视频 ID 的 UUID
- Scheduler 使用 worker 模式，具有可配置的 ticker 间隔
- 视频删除有意延迟（默认 300 秒），以便恢复

### Docker 相关
- 使用多阶段构建，镜像大小从 800MB 优化到 20MB
- 容器内使用非 root 用户运行（安全最佳实践）
- 健康检查集成到 Dockerfile 和 docker-compose.yml
- 网络隔离：所有服务在同一个 bridge 网络中

### 故障排查
- 查看服务日志：`tail -f logs/<service>.log` 或 `docker-compose logs -f <service>`
- 检查健康状态：`make health` 或 `curl http://localhost:<port>/health/ready`
- 进入容器调试：`docker exec -it video-server-<service> sh`
