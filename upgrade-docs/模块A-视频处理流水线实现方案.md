# 模块A：视频处理流水线 — 实现方案

> 目标：在现有视频上传流程之上，引入 Kafka 消息队列实现异步处理流水线，覆盖的技术点包括：Kafka 生产/消费、消息幂等性、分布式锁（Redis）、死信队列、视频状态机。

---

很好，但是我们还是把kafka之类依赖的外部服务部署到我的服务器上，就和mysql和redis一样，只有dev环境才用docker部署所有服务。所以：1.写一个deploy-kaf
  ka.sh的脚本，然后ssh到root@139.196.242.169这台机器上，切换到admin用户，切换到home/admin目录下，把服务部署上去，注意数据卷挂载到宿主机的~/data/ka
  fka-data下，端口用kafka常用的端口

## 一、现状 vs 目标

### 现状流程（同步）
```
POST /user/:name/videos
  → 生成 vid
  → 上传视频文件到 OSS（stream-service HTTP）
  → 上传封面到 OSS
  → 写 video_info 到 DB（status 字段不存在）
  → 返回 200，视频立即可播放
```

### 目标流程（异步流水线）
```
POST /user/:name/videos
  → 生成 vid
  → 上传视频到 OSS
  → 上传封面到 OSS
  → 写 video_info，status = PENDING
  → 发 Kafka 消息 [video.uploaded]
  → 返回 200（此时视频暂不可播放）

[异步]
transcode-service 消费 [video.uploaded]
  → 获取分布式锁（防重复处理）
  → 更新状态 PENDING → PROCESSING
  → mock 转码（sleep 3~5s）
  → 成功：发 [video.transcoded]
  → 失败：重试3次，仍失败→发 [video.transcode.dlq] + 状态变 FAILED

review-service 消费 [video.transcoded]
  → mock 审核（直接通过，或按配置 5% 拒绝率模拟测试）
  → 通过：发 [video.approved]   → status = AVAILABLE
  → 拒绝：发 [video.rejected]   → status = REJECTED

api-service 消费 [video.approved] / [video.rejected] / [video.transcode.dlq]
  → 更新 DB 最终状态
```

### 视频状态机
```
                   ┌─ transcode 失败(3次) → FAILED
PENDING → PROCESSING ┤
                   └─ transcode 成功 → [video.transcoded] ─┬─ review 通过 → AVAILABLE
                                                           └─ review 拒绝 → REJECTED
```

---

## 二、涉及的文件改动清单

### 新增文件

| 文件 | 说明 |
|------|------|
| `internal/kafka/producer.go` | 封装 Kafka 生产者（带重试） |
| `internal/kafka/consumer.go` | 封装 Kafka 消费者（手动 ACK、重试、DLQ） |
| `internal/kafka/message.go` | 消息体结构定义（VideoEvent） |
| `cmd/transcode/main.go` | transcode-service 入口 |
| `cmd/review/main.go` | review-service 入口 |
| `migrations/000004_add_video_status.up.sql` | 给 video_info 表加 status 列 |
| `migrations/000004_add_video_status.down.sql` | 回滚 |

### 修改文件

| 文件 | 改动内容 |
|------|---------|
| `api/defs/modle.go` | VideoInfo 增加 `Status` 字段 |
| `api/dbops/dbservice.go` | 新增 `AddNewVideo` 时写 PENDING 状态；新增 `UpdateVideoStatus()` |
| `api/handlers.go` | `AddNewVideo` 成功后发 Kafka 消息；`GetVideoInfo` 返回值加入 status |
| `api/handlers.go` | 新增 Kafka 消费者 goroutine（消费 approved/rejected/dlq） |
| `docker-compose.yml` | 新增 Kafka 服务（KRaft 模式，无需 Zookeeper）|
| `docker-compose.dev.yml` | 同上 |
| `go.mod` | 新增 `github.com/segmentio/kafka-go` 依赖 |

---

## 三、数据库变更

### migration 000004

```sql
-- up
ALTER TABLE video_info
  ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'pending'
  COMMENT 'pending/processing/available/rejected/failed';

CREATE INDEX idx_video_status ON video_info(status);

-- down
ALTER TABLE video_info DROP COLUMN status;
```

**状态值约定**（写成常量，不用 enum 方便扩展）：
```
pending     → 刚上传，等待处理
processing  → 转码中
available   → 审核通过，可播放
rejected    → 审核拒绝
failed      → 转码失败（重试耗尽）
```

---

## 四、Kafka Topics 设计

| Topic | 生产者 | 消费者 | 消息内容 |
|-------|--------|--------|---------|
| `video.uploaded` | api-service | transcode-service | vid、user_id、时间戳 |
| `video.transcoded` | transcode-service | review-service | vid、user_id |
| `video.approved` | review-service | api-service | vid |
| `video.rejected` | review-service | api-service | vid、reason |
| `video.transcode.dlq` | transcode-service（重试耗尽后） | api-service | vid、错误信息 |

**消息体统一格式**（`internal/kafka/message.go`）：
```go
type VideoEvent struct {
    Vid       string    `json:"vid"`
    UserID    int       `json:"user_id"`
    EventType string    `json:"event_type"` // 与 topic 对应
    Attempt   int       `json:"attempt"`    // 重试次数，用于 DLQ 判断
    Error     string    `json:"error,omitempty"`
    Timestamp time.Time `json:"timestamp"`
}
```

---

## 五、Kafka 基础设施封装（internal/kafka）

### producer.go 核心逻辑
- 使用 `kafka-go` 的 `kafka.Writer`
- 发送失败自动重试 3 次（指数退避）
- 支持同步发送（保证消息投递）

### consumer.go 核心逻辑
- 使用 `kafka-go` 的 `kafka.Reader`，手动提交 offset（`CommitMessages`）
- **处理成功**才提交 offset（保证 at-least-once 语义）
- **消息幂等性**：消费前检查 Redis key `processed:{topic}:{vid}`，已处理则跳过并提交
- **重试逻辑**：处理失败时不提交 offset，自动重试；超过 `maxRetry`（3次）后发 DLQ 消息

```
收到消息
  │
  ├─ 检查 Redis "processed:{topic}:{vid}" → 存在？跳过（提交 offset）
  │
  ├─ 获取分布式锁 "lock:{topic}:{vid}"（transcode-service 独有）
  │
  ├─ 执行业务逻辑
  │     ├─ 成功 → 写 Redis 幂等 key（TTL 24h） → 提交 offset
  │     └─ 失败且 attempt < 3 → 不提交 offset（Kafka 自动重投）
  │                attempt >= 3 → 发 DLQ 消息 → 提交 offset（避免死循环）
  └─ 释放分布式锁
```

---

## 六、各服务改动详解

### 6.1 api-service：AddNewVideo 改动

在现有步骤4（更新封面URL）成功之后，新增步骤5：

```
步骤5：发 Kafka 消息
  → 发到 topic "video.uploaded"
  → 消息内容：{vid, user_id, timestamp}
  → 发送失败：打 error 日志，但不回滚（视频数据已完整，后续可补发）
  → 返回给前端的 VideoInfo 中 status = "pending"
```

> **为什么发 Kafka 失败不回滚**：视频和封面已完整存入 OSS + DB，数据没有损坏。Kafka 暂时不可用是基础设施问题，不应该让用户重传视频。可以有补偿任务定期扫描 status=PENDING 且创建时间 > 5 分钟的视频，重新投递消息。

### 6.2 api-service：新增 Kafka 消费者

api-service 启动时，在后台起 goroutine 消费三个 topic：
- `video.approved` → `UpdateVideoStatus(vid, "available")`
- `video.rejected` → `UpdateVideoStatus(vid, "rejected")`
- `video.transcode.dlq` → `UpdateVideoStatus(vid, "failed")`  + 打 error 日志

### 6.3 api-service：GetVideoInfo 改动

前端轮询视频状态时，直接返回 `status` 字段。播放接口增加 status 校验：

```
GET /videos/:vid/url
  → 查询 video_info.status
  → status != "available" → 返回特定错误码（如 100601: "视频处理中"）
  → status == "available" → 正常返回签名 URL
```

### 6.4 transcode-service（新服务）

```
cmd/transcode/main.go
  │
  ├─ 加载配置（复用 internal/config，新增 kafka_addr 配置项）
  ├─ 初始化日志（复用 utils.InitLogging）
  ├─ 初始化 Redis 连接（用于分布式锁）
  ├─ 初始化 Kafka 消费者（消费 video.uploaded）
  ├─ 初始化 Kafka 生产者（发 video.transcoded 或 video.transcode.dlq）
  └─ 启动消费循环 + 健康检查（复用 internal/health）

业务逻辑（handleTranscode）：
  1. 获取分布式锁 "lock:transcode:{vid}"（TTL 60s）
  2. 直接更新 DB：status = "processing"（transcode-service 直接连 DB）
  3. mock 转码：time.Sleep(3 * time.Second)
  4. 随机模拟 5% 失败率（用于测试 DLQ）
  5. 成功 → 发 video.transcoded
  6. 失败 → 不提交 offset，consumer 框架自动重试；达到 maxRetry → 发 DLQ
  7. 释放锁
```

### 6.5 review-service（新服务）

```
cmd/review/main.go（结构与 transcode-service 一致）

业务逻辑（handleReview）：
  1. mock 审核：默认全部通过
     可通过环境变量 REVIEW_REJECT_RATE=5 配置拒绝率（测试用）
  2. 通过 → 发 video.approved
  3. 拒绝 → 发 video.rejected（reason: "内容违规（mock）"）
  注：review-service 不更新 DB，由 api-service 消费 approved/rejected 更新
```

---

## 七、docker-compose 新增 Kafka

使用 bitnami/kafka 3.6（KRaft 模式，不需要 Zookeeper，本地部署更简单）：

```yaml
kafka:
  image: bitnami/kafka:3.6
  container_name: video-server-kafka
  ports:
    - "9092:9092"
  environment:
    - KAFKA_CFG_NODE_ID=0
    - KAFKA_CFG_PROCESS_ROLES=controller,broker
    - KAFKA_CFG_LISTENERS=PLAINTEXT://:9092,CONTROLLER://:9093,EXTERNAL://:9094
    - KAFKA_CFG_ADVERTISED_LISTENERS=PLAINTEXT://kafka:9092,EXTERNAL://localhost:9094
    - KAFKA_CFG_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT,EXTERNAL:PLAINTEXT
    - KAFKA_CFG_CONTROLLER_QUORUM_VOTERS=0@kafka:9093
    - KAFKA_CFG_CONTROLLER_LISTENER_NAMES=CONTROLLER
    - KAFKA_CFG_AUTO_CREATE_TOPICS_ENABLE=true
  volumes:
    - kafka-data:/bitnami/kafka
  networks:
    - video-network
  healthcheck:
    test: ["CMD", "kafka-topics.sh", "--bootstrap-server", "localhost:9092", "--list"]
    interval: 10s
    timeout: 5s
    retries: 5
    start_period: 30s
```

新增 transcode-service 和 review-service 容器配置（复用同一个 Dockerfile，通过 `SERVICE_NAME` 构建参数区分）。

新增配置项（`config/config.json`）：
```json
"kafka_addr": "localhost:9092"
```
容器内通过环境变量覆盖为 `KAFKA_ADDR=kafka:9092`。

---

## 八、实施步骤（建议顺序）

```
Step 1：基础设施准备
  ├─ go.mod 添加 kafka-go 依赖
  ├─ docker-compose 添加 Kafka 容器
  └─ 执行 migration 000004（加 status 列）

Step 2：内部 Kafka 封装
  ├─ internal/kafka/message.go
  ├─ internal/kafka/producer.go（先写，先验证 Kafka 连通性）
  └─ internal/kafka/consumer.go（含幂等 + 重试逻辑）

Step 3：修改 api-service
  ├─ VideoInfo 模型加 Status 字段
  ├─ dbops 加 UpdateVideoStatus()
  ├─ AddNewVideo 上传成功后发 Kafka 消息
  ├─ 启动 Kafka 消费者 goroutine（消费 approved/rejected/dlq）
  └─ 播放接口加 status 校验

Step 4：实现 transcode-service
  ├─ cmd/transcode/main.go
  └─ 本地单元测试（发一条 video.uploaded 消息，验证能消费并改状态）

Step 5：实现 review-service
  ├─ cmd/review/main.go
  └─ 端到端测试：上传视频 → 观察 status 变化链路

Step 6：分布式锁集成
  └─ transcode-service 里加 Redis 锁，测试并发场景（同一 vid 发两条消息）

Step 7：完整测试 + docker-compose 验证
  ├─ docker-compose up 全容器验证
  └─ 手动测试：上传 → status 轮询 → available 后播放
```

---

## 九、关键技术点学习价值总结

| 技术点 | 在哪里体现 | 面试价值 |
|--------|-----------|---------|
| Kafka 生产者（同步发送、重试） | api-service AddNewVideo | ⭐⭐⭐ |
| Kafka 消费者（手动 ACK） | 三个消费者 | ⭐⭐⭐ |
| 消息幂等性（Redis 去重） | consumer.go 通用逻辑 | ⭐⭐⭐⭐ |
| 分布式锁（Redis SETNX） | transcode-service | ⭐⭐⭐⭐ |
| 死信队列（DLQ） | transcode-service 重试耗尽 | ⭐⭐⭐ |
| 状态机设计 | video status 五状态流转 | ⭐⭐⭐ |
| 异步解耦（不影响上传接口响应时间） | 整体架构 | ⭐⭐⭐⭐ |
| 补偿任务（发 Kafka 失败后的兜底） | 可选 step，加分项 | ⭐⭐ |
