# 🐛 问题排查和解决方案

**日期**：2026-03-08
**排查时长**：约30分钟
**状态**：✅ 所有问题已解决

---

## 🎯 问题清单

### ❌ 问题1：API和Scheduler服务启动失败

**错误现象：**
```
panic: runtime error: invalid memory address or nil pointer dereference
goroutine 1 [running]:
go.uber.org/zap.(*Logger).Error(0x0, ...)
video-server/api/dbops.init.0()
```

**根本原因：**
```
执行顺序：
1. import dbops → dbops.init() 自动执行
2. 数据库连接失败 → 调用 utils.Logger.Error()
3. 此时 utils.Logger 还是 nil（main函数还没执行到InitLogging）
4. nil.Error() → panic
```

**解决方案：**
- 将 `init()` 改为手动初始化函数 `Init()`
- 在 main 函数中按顺序初始化：配置 → 日志 → 数据库
- 不在 init() 中使用未初始化的全局变量

---

### ❌ 问题2：API服务健康检查返回401

**错误现象：**
```bash
curl http://localhost:8000/health/ready
# 返回 401 Unauthorized
```

**根本原因：**
- `router.Use()` 注册的全局中间件会拦截所有路由
- 健康检查路由虽然后注册，但依然被认证中间件拦截
- K8s需要无认证访问健康检查接口

**解决方案：**
- 在认证中间件中跳过健康检查路径
- 添加路径判断：`/health/*` 直接放行

---

### ❌ 问题3：Docker Compose版本不兼容

**错误现象：**
```
ERROR: Version in "./docker-compose.yml" is unsupported.
```

**解决方案：**
- 将 `version: '3.8'` 降级到 `'3.3'`

---

### ❌ 问题4：Docker部署与已有服务冲突

**问题描述：**
- 用户已有独立的 MySQL 和 Redis
- docker-compose.yml 会启动新的 MySQL 和 Redis
- 导致端口冲突或服务重复

**解决方案：**
- 提供 `docker-compose.dev.yml`（本地开发，包含MySQL/Redis）
- 提供 `docker-compose.prod.yml`（生产环境，只部署服务）
- 默认 `docker-compose.yml` 使用生产配置

---

## ✅ 验证结果

```bash
$ ./verify.sh

✅ 所有服务运行正常！

服务地址：
  - API服务:       http://localhost:8000
  - Web服务:       http://localhost:8080
  - Stream服务:    http://localhost:9090
  - Scheduler服务: http://localhost:8001
```

---

## 🎓 核心教训

1. **不要在 init() 中做复杂初始化**
   - init() 执行顺序不可控
   - 依赖可能还没初始化
   - 使用手动初始化函数

2. **健康检查必须公开**
   - K8s需要无认证访问
   - 在中间件中明确跳过

3. **分离开发和生产配置**
   - 不同环境有不同需求
   - 使用环境变量覆盖配置

4. **连接池配置很重要**
   - SetMaxIdleConns
   - SetMaxOpenConns
   - SetConnMaxLifetime
