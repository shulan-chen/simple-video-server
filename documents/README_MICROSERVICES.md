# 微服务架构使用指南

## 🎯 架构变更说明

### 改造前（单体应用）
```
video-server/
  └── main.go  # 一个进程启动4个服务
```

### 改造后（真正的微服务）
```
video-server/
  └── cmd/
      ├── api/       # API服务独立进程
      ├── web/       # Web服务独立进程
      ├── stream/    # Stream服务独立进程
      └── scheduler/ # Scheduler服务独立进程
```

## 🚀 快速开始

### 方式1：本地开发（推荐）

#### 1. 安装依赖
```bash
go mod tidy
```

#### 2. 启动单个服务（用于开发调试）
```bash
# 启动API服务
make run-api

# 启动Web服务（新终端）
make run-web

# 启动Stream服务（新终端）
make run-stream

# 启动Scheduler服务（新终端）
make run-scheduler
```

#### 3. 后台启动所有服务
```bash
# 构建并后台启动
make start

# 查看日志
tail -f logs/api.log
tail -f logs/web.log
tail -f logs/stream.log
tail -f logs/scheduler.log

# 停止所有服务
make stop
```

#### 4. 健康检查
```bash
# 检查所有服务状态
make health

# 或手动检查
curl http://localhost:8000/health/ready  # API
curl http://localhost:8080/health/ready  # Web
curl http://localhost:9090/health/ready  # Stream
curl http://localhost:8001/health/ready  # Scheduler
```

---

### 方式2：Docker Compose（模拟生产环境）

#### 1. 启动所有服务（包括MySQL和Redis）
```bash
docker-compose up -d
```

#### 2. 查看服务状态
```bash
docker-compose ps
```

#### 3. 查看日志
```bash
# 查看所有服务日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f api-service
```

#### 4. 停止服务
```bash
docker-compose down
```

---

## 🔍 为什么要这样改？

### 1. **独立部署**
```
旧方式：改一行代码 → 全部服务重启
新方式：改API代码 → 只重启API服务，其他服务不受影响
```

### 2. **独立扩容**
```
旧方式：Stream服务压力大 → 只能整个应用复制，浪费资源
新方式：Stream服务压力大 → 只扩Stream服务
```

**示例：**
```bash
# Docker Compose扩容Stream服务到3个实例
docker-compose up -d --scale stream-service=3
```

### 3. **故障隔离**
```
旧方式：API服务panic → 所有服务崩溃（包括正在上传的视频）
新方式：API服务panic → 只影响API，Stream服务正常运行
```

### 4. **健康检查**
```
旧方式：服务挂了不知道
新方式：
- Liveness: 进程是否存活（K8s用于重启决策）
- Readiness: 服务是否就绪（负载均衡用于流量转发）
- Startup: 慢启动检测
```

### 5. **优雅关闭**
```
旧方式：kill -9 → 连接断开、数据丢失
新方式：
1. 收到SIGTERM信号
2. 标记为不可用（停止接收新请求）
3. 处理完现有请求（最多等30秒）
4. 关闭数据库连接
5. 退出进程
```

---

## 📊 服务端口映射

| 服务 | 端口 | 用途 |
|------|------|------|
| API | 8000 | 用户认证、视频元数据、评论 |
| Web | 8080 | 前端页面、API代理 |
| Stream | 9090 | 视频上传/下载、OSS |
| Scheduler | 8001 | 后台任务、管理接口 |

---

## 🛠️ 开发工作流

### 场景1：修改API服务
```bash
# 1. 修改代码
vim api/handlers.go

# 2. 运行测试
go test ./api/...

# 3. 本地测试
make run-api

# 4. 构建
make build-api

# 5. 启动
./bin/api-service
```

### 场景2：调试服务间通信
```bash
# 启动docker-compose（包含所有依赖）
docker-compose up -d mysql redis

# 本地运行服务（方便调试）
make run-api      # 终端1
make run-web      # 终端2
make run-stream   # 终端3
```

### 场景3：压测单个服务
```bash
# 只启动Stream服务
make run-stream

# 压测
ab -n 1000 -c 10 http://localhost:9090/videos/test-vid
```

---

## 🎓 下一步优化建议

### 1. **服务注册与发现**
```
当前：硬编码服务地址
优化：使用Consul/Etcd，服务自动注册和发现
好处：动态扩缩容、自动负载均衡
```

### 2. **API网关**
```
当前：前端直接调用各个服务
优化：所有请求经过API网关（Kong/APISIX）
好处：统一认证、限流、日志、监控
```

### 3. **链路追踪**
```
当前：无法追踪一次请求的完整链路
优化：集成OpenTelemetry/Jaeger
好处：性能瓶颈分析、故障定位
```

### 4. **配置中心**
```
当前：配置文件
优化：Consul/Nacos配置中心
好处：动态配置、不重启更新配置
```

### 5. **消息队列**
```
当前：Scheduler用定时任务
优化：使用Kafka/RabbitMQ
好处：解耦、削峰、异步
```

---

## 🐛 常见问题

### Q1: 本地启动失败，提示端口被占用
```bash
# 查看端口占用
lsof -i :8000
lsof -i :8080

# 杀掉占用进程
kill -9 <PID>

# 或者修改配置文件中的端口
vim config/config.json
```

### Q2: Docker构建失败
```bash
# 清理旧镜像
docker system prune -a

# 重新构建
docker-compose build --no-cache
```

### Q3: 服务健康检查失败
```bash
# 查看服务日志
docker-compose logs api-service

# 进入容器调试
docker exec -it video-server-api sh
curl http://localhost:8000/health/ready
```

### Q4: 数据库连接失败
```bash
# 检查MySQL是否启动
docker-compose ps mysql

# 检查配置
cat config/config.json | grep db_addr

# Docker环境中应该使用容器名
"db_addr": "mysql:3306"  # ✅ 正确
"db_addr": "localhost:3306"  # ❌ 错误
```

---
  📊 改造前 vs 改造后

  改造前（单体应用）:
  ┌─────────────────────────────────┐
  │         video-server            │
  │  ┌─────┐ ┌──────┐ ┌────────┐  │
  │  │ API │ │ Web  │ │ Stream │  │
  │  └─────┘ └──────┘ └────────┘  │
  │         └─ Scheduler ─┘         │
  └─────────────────────────────────┘
      一个panic全部崩溃 ❌

  改造后（真微服务）:
  ┌──────────┐  ┌──────────┐  ┌───────────┐  ┌──────────────┐
  │   API    │  │   Web    │  │  Stream   │  │  Scheduler   │
  │ :8000    │  │  :8080   │  │  :9090    │  │   :8001      │
  │ 独立进程 │  │ 独立进程 │  │  独立进程 │  │   独立进程   │
  └──────────┘  └──────────┘  └───────────┘  └──────────────┘
      故障隔离 ✅    独立扩容 ✅    独立部署 ✅


## 📚 更多资源

- [CLAUDE.md](./CLAUDE.md) - 项目架构文档
- [Makefile](./Makefile) - 构建命令参考
- [docker-compose.yml](./docker-compose.yml) - 容器编排配置
