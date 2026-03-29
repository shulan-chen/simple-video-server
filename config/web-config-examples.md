# Web 网关配置示例

## 开发环境配置

```json
{
  "api_addr": ":8000",
  "web_addr": ":8080",
  "stream_addr": ":9090",
  "db_addr": "localhost:3306",
  "db_user": "root",
  "db_pwd": "password",
  "db_name": "video_server",
  "redis_addr": "localhost:6379",
  "redis_pwd": "",
  "redis_db": 0,
  "oss_addr": "oss-cn-shanghai.aliyuncs.com",
  "oss_region": "cn-shanghai",
  "oss_key": "YOUR_OSS_KEY",
  "oss_secret": "YOUR_OSS_SECRET",
  "oss_bucket": "video-bucket",
  "video_delete_delay_time": 300
}
```

## 生产环境配置（Docker Compose）

```json
{
  "api_addr": "http://api-service:8000",
  "web_addr": ":8080",
  "stream_addr": "http://stream-service:9090",
  "db_addr": "mysql:3306",
  "db_user": "prod_user",
  "db_pwd": "${DB_PASSWORD}",
  "db_name": "video_server_prod",
  "redis_addr": "redis:6379",
  "redis_pwd": "${REDIS_PASSWORD}",
  "redis_db": 0
}
```

## 生产环境配置（Kubernetes）

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: web-config
data:
  API_ADDR: "http://api-service.default.svc.cluster.local:8000"
  WEB_ADDR: ":8080"
  STREAM_ADDR: "http://stream-service.default.svc.cluster.local:9090"
  DB_ADDR: "mysql.database.svc.cluster.local:3306"
  REDIS_ADDR: "redis.cache.svc.cluster.local:6379"
---
apiVersion: v1
kind: Secret
metadata:
  name: web-secrets
type: Opaque
stringData:
  DB_PASSWORD: "your-db-password"
  REDIS_PASSWORD: "your-redis-password"
  OSS_KEY: "your-oss-key"
  OSS_SECRET: "your-oss-secret"
```

## 配置说明

### 服务地址配置

| 配置项 | 格式 | 示例 | 说明 |
|--------|------|------|------|
| `api_addr` | `:port` 或 `host:port` 或 `http://url` | `:8000` | API 服务地址 |
| `web_addr` | `:port` | `:8080` | Web 服务监听地址 |
| `stream_addr` | `:port` 或 `host:port` 或 `http://url` | `:9090` | Stream 服务地址 |

**地址解析规则**：
- `:8000` → `http://localhost:8000`（开发环境）
- `api-service` → `http://api-service`（Docker Compose / K8s）
- `http://api.example.com` → `http://api.example.com`（完整URL）

### 限流配置（当前为代码配置，可扩展为配置文件）

```json
{
  "rate_limits": {
    "global": {
      "rate": "100/s",
      "burst": 200
    },
    "api_proxy": {
      "rate": "10/s",
      "burst": 20
    },
    "video_proxy": {
      "rate": "5/s",
      "burst": 10
    }
  }
}
```

### 熔断器配置（当前为代码配置，可扩展为配置文件）

```json
{
  "circuit_breakers": {
    "api-service": {
      "max_failures": 5,
      "timeout": "30s",
      "half_open_success": 3
    },
    "stream-service": {
      "max_failures": 5,
      "timeout": "30s",
      "half_open_success": 3
    }
  }
}
```

## 环境变量覆盖

所有配置项都可以通过环境变量覆盖（优先级高于配置文件）：

```bash
# 开发环境
export API_ADDR=":8000"
export WEB_ADDR=":8080"
export STREAM_ADDR=":9090"

# 测试环境
export API_ADDR="http://api-test.internal"
export STREAM_ADDR="http://stream-test.internal"

# 生产环境
export API_ADDR="http://api-service.prod.svc.cluster.local"
export STREAM_ADDR="http://stream-service.prod.svc.cluster.local"
export DB_PASSWORD="$(cat /run/secrets/db_password)"
export REDIS_PASSWORD="$(cat /run/secrets/redis_password)"
```

## Viper 配置加载顺序

Viper 按以下顺序读取配置（后者覆盖前者）：

1. 默认值（代码中的 default）
2. 配置文件（config/config.json）
3. 环境变量（最高优先级）

**示例**：

```go
// config.json
{
  "api_addr": ":8000"
}

// 环境变量
export API_ADDR="http://api-prod.com"

// 最终结果
config.AppConfig.APIAddr = "http://api-prod.com"  // 环境变量优先
```
