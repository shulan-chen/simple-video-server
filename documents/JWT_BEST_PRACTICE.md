# JWT 最佳实践：双Token设计

## 🎯 核心问题

### 单Token的困境

```
方案A：Token有效期长（7天）
  ✅ 用户体验好（不用频繁登录）
  ❌ 被盗后7天内无法撤销，安全风险高

方案B：Token有效期短（30分钟）
  ✅ 被盗后影响小
  ❌ 用户每30分钟就要重新登录，体验极差
```

**矛盾**：安全性 vs 用户体验，无法兼得！

---

## ✅ 主流方案：双Token机制

### 架构设计

```
┌─────────────────────────────────────────────────────────┐
│  Access Token (短期，15分钟)                             │
│  - 用于API调用                                          │
│  - 存储用户信息（user_id, username）                    │
│  - 不可撤销（JWT无状态特性）                            │
│  - 过期后需要refresh                                    │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│  Refresh Token (长期，7天)                              │
│  - 用于刷新Access Token                                 │
│  - 存储在Redis中（可撤销）                              │
│  - 只能调用 /refresh 接口                               │
│  - 登出时从Redis删除                                    │
└─────────────────────────────────────────────────────────┘
```

---

## 📊 完整流程

### 1. 登录流程

```
用户登录
  ↓
后端验证密码
  ↓
生成 Access Token (15分钟)
生成 Refresh Token (7天)
  ↓
Refresh Token 存入 Redis (key: refresh_token, value: user_id)
  ↓
返回给客户端:
{
  "access_token": "eyJhbGc...",
  "refresh_token": "f8d3e2a1...",
  "expires_in": 900  // 15分钟
}
```

### 2. API调用流程

```
客户端请求 API
  ↓
Header: Authorization: Bearer <access_token>
  ↓
后端验证 Access Token
  ↓
if (token未过期) → 处理请求 ✅
if (token已过期) → 返回 401 + "token_expired" ❌
  ↓
客户端收到 401
  ↓
调用 /refresh 接口（带 refresh_token）
  ↓
获取新的 access_token
  ↓
重试原请求
```

### 3. Token刷新流程

```
POST /auth/refresh
Body: {
  "refresh_token": "f8d3e2a1..."
}
  ↓
后端验证 Refresh Token
  ↓
1. 解析 JWT 签名（验证未被篡改）
2. 检查 Redis 中是否存在（验证未被撤销）
3. 检查是否过期
  ↓
if (全部通过)
  生成新的 Access Token (15分钟)
  (可选) 生成新的 Refresh Token (7天) - Refresh Token轮换
  返回:
  {
    "access_token": "eyJhbGc...",
    "refresh_token": "新的refresh_token",  // 可选
    "expires_in": 900
  }
```

### 4. 登出流程

```
POST /auth/logout
  ↓
从 Redis 删除 Refresh Token
  ↓
客户端删除本地存储的 tokens
  ↓
✅ 即使 Access Token 还有效（最多15分钟），
   用户也无法refresh，被迫重新登录
```

---

## 🔑 核心优势

| 特性 | 实现方式 | 优势 |
|------|---------|------|
| **用户体验** | Access Token短期自动续期 | 用户感觉不到过期 |
| **安全性** | Access Token 15分钟过期 | 被盗后影响窗口小 |
| **可撤销** | Refresh Token存Redis | 登出/风控可立即撤销 |
| **性能** | Access Token无状态 | API调用不查数据库 |

---

## 🛡️ 安全加固

### 1. Refresh Token轮换（Rotation）

```
每次refresh时，生成新的refresh_token，旧的立即失效

时间线:
T0: 登录，获得 refresh_token_1
T1: 15分钟后，用 refresh_token_1 换 access_token_2 + refresh_token_2
    → refresh_token_1 立即失效
T2: 30分钟后，用 refresh_token_2 换 access_token_3 + refresh_token_3
    → refresh_token_2 立即失效

优势：
  - 如果 refresh_token_1 被盗，黑客用它时，合法用户的 refresh_token_2 仍有效
  - 检测到同一个旧token被多次使用 → 判定为token泄露 → 撤销所有token
```

### 2. Refresh Token绑定设备

```go
// 生成时绑定
type RefreshTokenData struct {
    UserId      int    `json:"user_id"`
    DeviceID    string `json:"device_id"`  // 设备指纹
    UserAgent   string `json:"user_agent"`
    IP          string `json:"ip"`
}

// 验证时检查
if token.DeviceID != currentDeviceID {
    return errors.New("token被盗用")
}
```

### 3. 滑动过期时间

```
用户活跃时自动延长 Refresh Token 有效期

if (距离上次活跃 < 24小时) {
    延长 Refresh Token 到 7天
}

结果：
  - 活跃用户永远不会被登出
  - 不活跃用户7天后自动登出
```

---

## 💾 存储位置

| Token | 客户端存储 | 服务端存储 |
|-------|-----------|-----------|
| **Access Token** | localStorage / Cookie | ❌ 不存储（无状态） |
| **Refresh Token** | httpOnly Cookie（推荐）/ localStorage | ✅ Redis |

**httpOnly Cookie优势**：
- JS无法访问（防XSS攻击）
- 自动发送（无需手动管理）
- 可设置 Secure 标志（仅HTTPS）

---

## 🔥 常见问题

### Q1: 为什么不把Access Token也存Redis？
**A**: 性能问题。每个API请求都要查Redis，延迟+负载都会显著增加。双Token方案实现了：
- 高频操作（API调用）无状态（快）
- 低频操作（refresh）有状态（可控）

### Q2: Refresh Token被盗怎么办？
**A**:
1. 轮换机制：旧token立即失效
2. 异常检测：同一token多次使用 → 撤销所有token
3. 设备绑定：非法设备无法使用
4. 最坏情况：7天后自动过期

### Q3: Access Token 15分钟会不会太短？
**A**: 不会，因为refresh是自动的：
```javascript
// 客户端自动refresh逻辑
axios.interceptors.response.use(
  response => response,
  async error => {
    if (error.response.status === 401 && error.response.data.error === 'token_expired') {
      // 自动refresh
      const newToken = await refreshAccessToken()
      // 重试原请求
      error.config.headers['Authorization'] = `Bearer ${newToken}`
      return axios.request(error.config)
    }
    return Promise.reject(error)
  }
)
```

### Q4: 为什么不用Session？
**A**:
- Session需要服务器存储，不适合分布式部署
- JWT无状态，水平扩展容易
- 但双Token方案保留了Session的可撤销性（通过Redis存Refresh Token）

---

## 📝 实现检查清单

### 后端
- [ ] 生成 Access Token (15分钟)
- [ ] 生成 Refresh Token (7天，随机字符串)
- [ ] Refresh Token 存入 Redis (key: `refresh:<token>`, value: user_id)
- [ ] `/auth/refresh` 接口
- [ ] `/auth/logout` 删除 Redis 中的 Refresh Token
- [ ] Refresh Token 轮换（可选，推荐）
- [ ] 异常检测：重复使用旧token（可选）

### 前端
- [ ] 存储 access_token 和 refresh_token
- [ ] API调用带 `Authorization: Bearer <access_token>`
- [ ] 拦截401错误，自动调用 `/auth/refresh`
- [ ] 登出时删除本地tokens

---

## 🎯 你的项目改进方案

### 当前问题

```go
// 1. 只有一个token，30分钟过期（用户体验差）
ttl := 30 * time.Minute

// 2. 无法撤销（用户登出后token仍然有效）
// 3. Secret硬编码
jwtSecret = []byte("your_secret_key")
```

### 改进后

```go
// 1. Access Token: 15分钟
// 2. Refresh Token: 7天，存储在Redis
// 3. 登录返回双token
// 4. /auth/refresh 接口刷新
// 5. /auth/logout 删除Redis中的refresh token
// 6. Secret从环境变量读取
```

---

## 🚀 迁移策略

### 渐进式升级（推荐）

```
阶段1: 保持现有逻辑，添加 Refresh Token 支持
  - 登录时额外返回 refresh_token
  - 添加 /auth/refresh 接口
  - 客户端可选升级

阶段2: 缩短 Access Token 有效期
  - 从 30分钟 → 15分钟
  - 监控 /auth/refresh 调用量

阶段3: 强制使用双Token
  - Access Token 15分钟
  - 移除旧的长期token支持
```

---

**生成时间**: 2026-03-15
**适用项目**: video-server
