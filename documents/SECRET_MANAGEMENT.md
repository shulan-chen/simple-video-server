# 企业级敏感配置管理方案详解

## 📋 目录

1. [问题背景](#问题背景)
2. [常见方案对比](#常见方案对比)
3. [KeyCenter/Vault 架构详解](#keycenter架构详解)
4. [安全机制原理](#安全机制原理)
5. [工作流程](#工作流程)
6. [开源方案推荐](#开源方案推荐)
7. [本项目改进建议](#本项目改进建议)

---

## 问题背景

### 当前项目存在的问题

```json
// config.json - ❌ 不安全的做法
{
    "oss_key": "LTAI5tXXXXXXXXXX",          // 明文存储
    "oss_secret": "3aBcDeFgHiJkLmNoPqRs",   // 明文存储
    "db_pwd": "123456",                      // 明文存储
    "redis_pwd": "mypassword"                // 明文存储
}
```

**风险**：
- ✅ 你已经用 `.gitignore` 屏蔽，防止提交到 GitHub（GitHub Secret Scanning 会检测）
- ❌ 但文件仍在服务器上明文存储，任何有登录权限的人都能看到
- ❌ 日志、备份、配置分发过程中可能泄露
- ❌ 无法追溯谁在什么时候访问了密钥

---

## 常见方案对比

| 方案 | 安全性 | 复杂度 | 适用场景 | 成本 |
|------|--------|--------|----------|------|
| **1. 环境变量** | ⭐⭐ | ⭐ | 个人项目、原型开发 | 免费 |
| **2. 配置中心** | ⭐⭐⭐ | ⭐⭐ | 中小团队、微服务架构 | 免费/低 |
| **3. 密钥管理系统** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | 大型企业、金融/政务 | 中/高 |

### 方案1: 环境变量（最简单）

```bash
# .env 文件（不提交到 Git）
OSS_KEY=LTAI5tXXXXXXXXXX
OSS_SECRET=3aBcDeFgHiJkLmNoPqRs
DB_PWD=123456
```

```go
// 代码中读取
ossKey := os.Getenv("OSS_KEY")
```

**优点**：
- 简单，无需额外服务
- 容器化部署友好（Docker/K8s Secrets）

**缺点**：
- 仍是明文存储（在 `.env` 文件或环境中）
- 无法集中管理、审计
- 密钥轮换困难

---

### 方案2: 配置中心（推荐中小团队）

**开源方案**：Consul、etcd、Apollo、Nacos

**架构**：

```
┌──────────────┐         ┌──────────────┐
│  配置中心     │◄────────│  管理后台     │
│  (Consul)    │         │  (加密上传)   │
└──────┬───────┘         └──────────────┘
       │
       │ HTTPS + ACL
       │
┌──────▼───────┐
│  微服务       │
│  启动时拉取   │
└──────────────┘
```

**示例（使用 Consul）**：

```bash
# 1. 加密存储到 Consul
consul kv put -base64 video-server/oss_secret "$(echo '3aBcDeFg' | base64)"

# 2. 服务启动时拉取
curl -H "X-Consul-Token: secret-token" \
     http://consul:8500/v1/kv/video-server/oss_secret
```

**优点**：
- 集中管理配置
- 支持动态更新（无需重启服务）
- 有基本的访问控制

**缺点**：
- 传输过程中仍可能被抓包（需要 TLS）
- 审计功能较弱

---

### 方案3: 密钥管理系统（企业级）

**代表产品**：
- **开源**：HashiCorp Vault、Keywhiz
- **云厂商**：AWS KMS、阿里云 KMS、腾讯云 KMS
- **企业自研**：字节 KeyCenter、美团 Shepherd

这是你观察到的公司方案，下面详细讲解。

---

## KeyCenter架构详解

### 整体架构图

```
┌─────────────────────────────────────────────────────────┐
│                    Control Plane                        │
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │  Master 1    │  │  Master 2    │  │  Master 3    │  │
│  │  (Leader)    │◄─┤  (Follower)  │◄─┤  (Follower)  │  │
│  └──────┬───────┘  └──────────────┘  └──────────────┘  │
│         │                                                │
│         │ Raft 协议（选举、复制）                         │
│         │                                                │
│  ┌──────▼────────────────────────────────────┐          │
│  │    加密存储后端 (Encrypted Storage)       │          │
│  │    - Master Key (硬件加密/云 KMS)         │          │
│  │    - Data Encryption Keys (DEK)          │          │
│  │    - 租户密钥 (Tenant Keys)               │          │
│  └───────────────────────────────────────────┘          │
└─────────────────────────────────────────────────────────┘
                           ▲
                           │ mTLS (双向认证)
                           │
┌──────────────────────────┼──────────────────────────────┐
│                    Data Plane                           │
│                                                          │
│  ┌────────────────┐     ┌────────────────┐             │
│  │  Agent 1       │     │  Agent 2       │             │
│  │  (服务器A)      │     │  (服务器B)      │             │
│  │                │     │                │             │
│  │  ┌──────────┐  │     │  ┌──────────┐  │             │
│  │  │  本地缓存  │  │     │  │  本地缓存  │  │             │
│  │  └────┬─────┘  │     │  └────┬─────┘  │             │
│  │       │        │     │       │        │             │
│  │  ┌────▼─────┐  │     │  ┌────▼─────┐  │             │
│  │  │ 微服务1   │  │     │  │ 微服务2   │  │             │
│  │  └──────────┘  │     │  └──────────┘  │             │
│  └────────────────┘     └────────────────┘             │
└─────────────────────────────────────────────────────────┘
```

### 核心组件

#### 1. Master 节点（Control Plane）

**职责**：
- 密钥生成、存储、分发
- 访问控制（ACL）
- 审计日志记录
- 密钥轮换策略

**高可用设计**：
- 多节点部署（通常 3-5 个）
- Raft 协议选举 Leader
- 数据自动复制

**示例代码（简化）**：

```go
// Master 节点核心逻辑
type MasterNode struct {
    storage      *EncryptedStorage  // 加密存储
    masterKey    []byte              // 主密钥（从硬件获取）
    auditLogger  *AuditLogger        // 审计日志
}

// 创建密钥
func (m *MasterNode) CreateSecret(ctx context.Context, req *CreateSecretRequest) error {
    // 1. 鉴权
    if !m.authorize(ctx, req.TenantID, "secret:create") {
        return ErrUnauthorized
    }

    // 2. 生成数据加密密钥（DEK）
    dek := generateRandomKey(32)

    // 3. 用 Master Key 加密 DEK
    encryptedDEK := encrypt(dek, m.masterKey)

    // 4. 用 DEK 加密实际密钥
    encryptedSecret := encrypt(req.SecretValue, dek)

    // 5. 存储
    m.storage.Save(&Secret{
        ID:              req.SecretID,
        TenantID:        req.TenantID,
        EncryptedValue:  encryptedSecret,
        EncryptedDEK:    encryptedDEK,
        CreatedAt:       time.Now(),
    })

    // 6. 记录审计日志
    m.auditLogger.Log(ctx, "secret.created", req.SecretID, req.TenantID)

    return nil
}
```

---

#### 2. Agent 节点（Data Plane）

**职责**：
- 部署在每台服务器上
- 启动时向 Master 认证并拉取密钥
- 提供本地 HTTP API（`http://127.0.0.1:8200`）
- 本地缓存（减少 Master 压力）

**示例代码**：

```go
// Agent 节点
type AgentNode struct {
    masterAddr   string
    certFile     string  // mTLS 客户端证书
    cache        *Cache  // 本地缓存
}

// 获取密钥（对外提供的 API）
func (a *AgentNode) GetSecret(secretID string) (string, error) {
    // 1. 先查本地缓存
    if cached, ok := a.cache.Get(secretID); ok {
        return cached, nil
    }

    // 2. 缓存未命中，请求 Master
    client := &http.Client{
        Transport: &http.Transport{
            TLSClientConfig: loadTLSConfig(a.certFile),
        },
    }

    resp, err := client.Get(fmt.Sprintf("%s/api/v1/secrets/%s", a.masterAddr, secretID))
    if err != nil {
        return "", err
    }

    var secret SecretResponse
    json.NewDecoder(resp.Body).Decode(&secret)

    // 3. 解密密钥（Master 返回的是已解密的）
    plaintext := secret.Value

    // 4. 写入缓存（TTL 5分钟）
    a.cache.Set(secretID, plaintext, 5*time.Minute)

    return plaintext, nil
}
```

---

#### 3. 加密存储后端

**多层加密架构**（Envelope Encryption）：

```
明文密钥 (Plaintext Secret)
    ↓ 加密
数据加密密钥 (DEK - Data Encryption Key)
    ↓ 加密
主密钥 (Master Key)
    ↓ 存储在
硬件安全模块 (HSM) / 云 KMS
```

**为什么要多层加密？**
- **性能**：DEK 用于加密大量数据（对称加密快）
- **安全**：Master Key 保护所有 DEK，只需保护一个密钥
- **轮换**：轮换 Master Key 时只需重新加密 DEK，不需要重新加密所有数据

**示例**：

```go
// 存储结构
type EncryptedSecret struct {
    ID              string    `json:"id"`
    TenantID        string    `json:"tenant_id"`
    EncryptedValue  []byte    `json:"encrypted_value"`  // 用 DEK 加密的密钥
    EncryptedDEK    []byte    `json:"encrypted_dek"`    // 用 Master Key 加密的 DEK
    Algorithm       string    `json:"algorithm"`        // AES-256-GCM
    CreatedAt       time.Time `json:"created_at"`
    RotatedAt       time.Time `json:"rotated_at"`
}
```

---

## 安全机制原理

### 1. 传输安全（mTLS 双向认证）

```
┌──────────────┐                  ┌──────────────┐
│   Agent      │                  │   Master     │
│              │──── 1. Client ───▶│              │
│  证书: A.crt │      Hello       │  证书: M.crt │
│              │◄─── 2. Server ────│              │
│              │      Hello       │              │
│              │──── 3. 验证 ──────▶│              │
│              │      M.crt       │              │
│              │◄─── 4. 验证 ───────│              │
│              │      A.crt       │              │
│              │──── 5. 加密 ──────▶│              │
│              │      通信         │              │
└──────────────┘                  └──────────────┘
```

**好处**：
- 防止中间人攻击
- 双向验证身份（Agent 和 Master 互相验证）

---

### 2. 访问控制（RBAC + ACL）

```go
// 权限模型
type Policy struct {
    TenantID   string   `json:"tenant_id"`
    ServiceID  string   `json:"service_id"`
    Secrets    []string `json:"secrets"`      // 可访问的密钥 ID
    Actions    []string `json:"actions"`      // read, write, delete
    ExpiresAt  *time.Time `json:"expires_at"` // 临时权限
}

// 鉴权逻辑
func (m *MasterNode) authorize(ctx context.Context, tenantID, action string) bool {
    // 从证书中提取 Service ID
    cert := ctx.Value("client_cert").(*x509.Certificate)
    serviceID := cert.Subject.CommonName

    // 查询权限
    policy, err := m.storage.GetPolicy(tenantID, serviceID)
    if err != nil {
        return false
    }

    // 检查权限是否过期
    if policy.ExpiresAt != nil && time.Now().After(*policy.ExpiresAt) {
        return false
    }

    // 检查是否有对应的 action 权限
    return contains(policy.Actions, action)
}
```

---

### 3. 审计日志

**记录内容**：
- 谁（ServiceID）
- 什么时候（Timestamp）
- 做了什么（Action: read/write/delete）
- 访问了什么（SecretID）
- 结果（Success/Fail）
- 来源（IP、主机名）

**示例日志**：

```json
{
  "timestamp": "2026-03-14T10:23:45Z",
  "service_id": "video-server-api",
  "tenant_id": "team-a",
  "action": "secret.read",
  "secret_id": "oss_secret",
  "result": "success",
  "source_ip": "10.0.1.5",
  "request_id": "req-abc123"
}
```

**用途**：
- 安全审计
- 异常检测（如频繁访问、非工作时间访问）
- 合规要求（SOC2、ISO 27001）

---

### 4. 密钥轮换

**自动轮换策略**：

```go
// 轮换策略
type RotationPolicy struct {
    SecretID       string        `json:"secret_id"`
    Interval       time.Duration `json:"interval"`        // 90 天
    NotifyBefore   time.Duration `json:"notify_before"`   // 提前 7 天通知
    AutoRotate     bool          `json:"auto_rotate"`     // 自动轮换
}

// 轮换流程
func (m *MasterNode) rotateSecret(secretID string) error {
    // 1. 生成新密钥
    newSecret := generateNewSecret()

    // 2. 保存新版本（版本号 +1）
    m.storage.SaveVersion(secretID, newSecret, version+1)

    // 3. 通知所有订阅者（通过 Webhook 或消息队列）
    m.notifyRotation(secretID, newSecret)

    // 4. 保留旧版本 30 天（灰度切换）
    time.AfterFunc(30*24*time.Hour, func() {
        m.storage.DeleteVersion(secretID, version)
    })

    return nil
}
```

---

### 5. 临时凭证（Dynamic Secrets）

**场景**：数据库密码、API Token

```go
// 动态生成临时数据库密码
func (m *MasterNode) GenerateTempDBCredential(serviceID string, ttl time.Duration) (*DBCredential, error) {
    // 1. 连接数据库
    db := connectToMySQL(m.rootCredential)

    // 2. 创建临时用户
    username := fmt.Sprintf("temp_%s_%d", serviceID, time.Now().Unix())
    password := generateRandomPassword()

    db.Exec(fmt.Sprintf("CREATE USER '%s'@'%%' IDENTIFIED BY '%s'", username, password))
    db.Exec(fmt.Sprintf("GRANT SELECT ON video_server.* TO '%s'@'%%'", username))

    // 3. 设置过期时间（后台任务会删除）
    m.scheduleDelete(username, ttl)

    return &DBCredential{
        Username:  username,
        Password:  password,
        ExpiresAt: time.Now().Add(ttl),
    }, nil
}
```

**好处**：
- 凭证泄露后影响范围小（自动过期）
- 每个服务实例有独立凭证（易于追溯）

---

## 工作流程

### 流程1: 配置加密上传

```bash
# 管理员操作
$ keycenter-cli create-secret \
    --name oss_secret \
    --value "3aBcDeFgHiJkLmNoPqRs" \
    --tenant video-server \
    --allowed-services api-service,stream-service

✅ Secret created: oss_secret
   ID: secret-abc123
   Encrypted and stored in Master
```

**后台发生了什么**：
1. CLI 通过 mTLS 连接 Master
2. Master 鉴权管理员身份
3. 生成 DEK，加密密钥
4. 用 Master Key 加密 DEK
5. 存储到加密数据库
6. 记录审计日志

---

### 流程2: 服务启动时获取密钥

```go
// 服务启动代码
func main() {
    // 1. 连接本地 Agent
    agentClient := keycenter.NewAgentClient("http://127.0.0.1:8200")

    // 2. 获取密钥
    ossSecret, err := agentClient.GetSecret("oss_secret")
    if err != nil {
        log.Fatalf("Failed to get secret: %v", err)
    }

    // 3. 使用明文密钥初始化 OSS 客户端
    ossClient := oss.NewClient(ossKey, ossSecret)

    // 4. 启动服务
    r := gin.Default()
    r.Run(":8000")
}
```

**Agent 后台流程**：
1. Agent 用 mTLS 证书连接 Master
2. Master 验证 Agent 身份（从证书提取 ServiceID）
3. Master 检查 ACL（`api-service` 是否有权限访问 `oss_secret`）
4. Master 解密 DEK，再解密密钥，返回明文
5. Agent 缓存密钥（TTL 5 分钟）
6. Master 记录审计日志

---

### 流程3: 密钥轮换（零停机）

```
时间线：
T0: 旧密钥 v1 在使用
T1: 管理员触发轮换，生成新密钥 v2
T2: Master 通知所有 Agent 有新版本
T3: Agent 清空缓存，下次请求获取 v2
T4: 服务重启（或监听配置变更）
T5: 旧密钥 v1 保留 30 天后删除
```

---

## 开源方案推荐

### 1. HashiCorp Vault（最流行）

**特点**：
- 功能最全（密钥管理、动态凭证、PKI、SSH）
- 社区活跃
- 支持多种存储后端（Consul、etcd、MySQL）

**快速部署**：

```bash
# 1. 启动 Vault 服务端
docker run -d --name vault \
    -p 8200:8200 \
    -e VAULT_DEV_ROOT_TOKEN_ID=myroot \
    vault:latest

# 2. 存储密钥
vault kv put secret/video-server \
    oss_secret=3aBcDeFg \
    db_pwd=123456

# 3. Go 代码读取
import (
    vault "github.com/hashicorp/vault/api"
)

func main() {
    client, _ := vault.NewClient(&vault.Config{
        Address: "http://127.0.0.1:8200",
    })
    client.SetToken("myroot")

    secret, _ := client.Logical().Read("secret/data/video-server")
    ossSecret := secret.Data["data"].(map[string]interface{})["oss_secret"].(string)

    fmt.Println("OSS Secret:", ossSecret)
}
```

---

### 2. 云厂商 KMS（最简单）

**阿里云 KMS 示例**：

```bash
# 1. 加密配置文件
aliyun kms Encrypt \
    --KeyId=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
    --Plaintext="3aBcDeFgHiJkLmNoPqRs" \
    > encrypted_secret.txt

# 2. 部署加密后的配置
echo "oss_secret_encrypted: CipherBlob..." >> config.yaml

# 3. 代码中解密
import (
    "github.com/aliyun/alibaba-cloud-sdk-go/services/kms"
)

func main() {
    client, _ := kms.NewClientWithAccessKey("cn-shanghai", accessKey, secretKey)

    request := kms.CreateDecryptRequest()
    request.CiphertextBlob = "CipherBlob..."

    response, _ := client.Decrypt(request)
    ossSecret := response.Plaintext  // 明文密钥
}
```

**优点**：
- 无需自建服务
- 与云服务深度集成
- 硬件级安全（HSM）

**缺点**：
- 供应商锁定
- 成本高（调用次数计费）

---

## 本项目改进建议

### 方案A: 最小改动（环境变量 + Docker Secrets）

```dockerfile
# docker-compose.yml
services:
  api-service:
    environment:
      - CONFIG_PATH=/app/config/config.json
    secrets:
      - oss_secret
      - db_pwd

secrets:
  oss_secret:
    file: ./secrets/oss_secret.txt  # 不提交到 Git
  db_pwd:
    file: ./secrets/db_pwd.txt
```

```go
// config/config.go
func Load() error {
    // 1. 读取非敏感配置
    viper.ReadInConfig()

    // 2. 敏感信息从 Docker Secrets 读取
    ossSecret, _ := os.ReadFile("/run/secrets/oss_secret")
    AppConfig.OssSecret = string(ossSecret)

    dbPwd, _ := os.ReadFile("/run/secrets/db_pwd")
    AppConfig.DbPwd = string(dbPwd)

    return nil
}
```

---

### 方案B: 中等改动（使用 Vault）

```bash
# 1. 部署 Vault（开发模式）
docker run -d --name vault \
    --network video-network \
    -p 8200:8200 \
    -e VAULT_DEV_ROOT_TOKEN_ID=myroot \
    vault:latest

# 2. 初始化密钥
vault kv put secret/video-server \
    oss_key=LTAI5tXXXXXXXXXX \
    oss_secret=3aBcDeFgHiJkLmNoPqRs \
    db_pwd=123456 \
    redis_pwd=mypassword
```

```go
// internal/vault/client.go
package vault

import (
    vault "github.com/hashicorp/vault/api"
)

type Client struct {
    client *vault.Client
}

func NewClient() (*Client, error) {
    config := vault.DefaultConfig()
    config.Address = os.Getenv("VAULT_ADDR") // http://vault:8200

    client, err := vault.NewClient(config)
    if err != nil {
        return nil, err
    }

    // 从环境变量读取 Token
    client.SetToken(os.Getenv("VAULT_TOKEN"))

    return &Client{client: client}, nil
}

func (c *Client) GetSecret(path string, key string) (string, error) {
    secret, err := c.client.Logical().Read(path)
    if err != nil {
        return "", err
    }

    data := secret.Data["data"].(map[string]interface{})
    return data[key].(string), nil
}
```

```go
// cmd/api/main.go 修改
func main() {
    // 1. 连接 Vault
    vaultClient, err := vault.NewClient()
    if err != nil {
        log.Fatalf("Vault 连接失败: %v", err)
    }

    // 2. 获取密钥
    config.AppConfig.OssSecret, _ = vaultClient.GetSecret("secret/data/video-server", "oss_secret")
    config.AppConfig.DbPwd, _ = vaultClient.GetSecret("secret/data/video-server", "db_pwd")

    // 3. 启动服务
    api.Start()
}
```

---

### 方案C: 生产级改动（阿里云 KMS）

```go
// internal/kms/aliyun.go
package kms

import (
    "github.com/aliyun/alibaba-cloud-sdk-go/services/kms"
)

type AliyunKMS struct {
    client *kms.Client
}

func NewAliyunKMS() (*AliyunKMS, error) {
    client, err := kms.NewClientWithAccessKey(
        "cn-shanghai",
        os.Getenv("ALIBABA_CLOUD_ACCESS_KEY"),
        os.Getenv("ALIBABA_CLOUD_SECRET_KEY"),
    )
    return &AliyunKMS{client: client}, err
}

func (k *AliyunKMS) Decrypt(ciphertext string) (string, error) {
    request := kms.CreateDecryptRequest()
    request.CiphertextBlob = ciphertext

    response, err := k.client.Decrypt(request)
    if err != nil {
        return "", err
    }

    return response.Plaintext, nil
}
```

```json
// config/config.json - 存储加密后的密钥
{
    "oss_secret_encrypted": "CipherBlobXXXXXXXX",
    "db_pwd_encrypted": "CipherBlobYYYYYYYY"
}
```

---

## 总结对比表

| 方案 | 安全性 | 实施难度 | 运维成本 | 推荐场景 |
|------|--------|----------|----------|----------|
| **环境变量** | ⭐⭐ | ⭐ | ⭐ | 个人项目、快速原型 |
| **Docker Secrets** | ⭐⭐⭐ | ⭐⭐ | ⭐ | 容器化部署 |
| **Vault** | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ | 中小团队、混合云 |
| **云 KMS** | ⭐⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐ | 生产环境、合规要求 |
| **自研 KeyCenter** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | 大型企业、定制需求 |

---

## 下一步建议

### 对于你的项目（学习项目）

我建议**方案A（环境变量 + Docker Secrets）**：
- 理解原理（密文 vs 明文）
- 快速实施（1小时内）
- 符合最佳实践（配置与代码分离）

### 如果想深入学习

我建议尝试**方案B（Vault）**：
- 搭建 Vault 开发环境（Docker 一键部署）
- 体验企业级密钥管理
- 为未来工作打基础

---

## 附录：关键术语

- **KMS**: Key Management Service，密钥管理服务
- **HSM**: Hardware Security Module，硬件安全模块
- **DEK**: Data Encryption Key，数据加密密钥
- **mTLS**: Mutual TLS，双向 TLS 认证
- **RBAC**: Role-Based Access Control，基于角色的访问控制
- **ACL**: Access Control List，访问控制列表
- **Envelope Encryption**: 信封加密，用密钥加密密钥
- **Dynamic Secrets**: 动态密钥，临时生成的凭证

---

**文档生成时间**: 2026-03-14
**适用项目**: video-server
**作者**: Claude Code (Assisted)
