# 前后端分离改造总结

## 📅 改造日期
2026-04-12

## 🎯 改造目标
将原有的后端渲染单体应用改造为前后端完全分离的架构。

## ✅ 完成的工作

### 1. 前端项目创建
创建了完整的 Vue 3 前端项目（位于 `/home/yang/Desktop/self_learn/simple_video_server_front`）

**技术栈**：
- Vue 3 (Composition API)
- Vite 5
- Vue Router 4
- Pinia 状态管理
- Element Plus UI 组件库
- Axios HTTP 客户端

**功能实现**：
- ✅ 用户注册/登录（JWT认证）
- ✅ 视频列表（全部/我的）
- ✅ 视频上传（拖拽 + 进度条）
- ✅ 视频播放
- ✅ 视频删除
- ✅ 评论功能

### 2. 前端代理配置修正

**修改文件**：`simple_video_server_front/vite.config.js`

**修改前**（错误）：
```javascript
server: {
  port: 5173,
  proxy: {
    '/api': {
      target: 'http://localhost:8000',  // ❌ 直接访问 API 服务
      changeOrigin: true,
      rewrite: (path) => path.replace(/^\/api/, '')
    },
    '/stream': {
      target: 'http://localhost:9090',  // ❌ 直接访问 Stream 服务
      changeOrigin: true,
      rewrite: (path) => path.replace(/^\/stream/, '')
    }
  }
}
```

**修改后**（正确）：
```javascript
server: {
  port: 80,
  proxy: {
    '/api': {
      target: 'http://localhost:8080',  // ✅ 统一访问 Web 网关
      changeOrigin: true
      // 保持 /api 前缀，由网关转发
    },
    '/stream': {
      target: 'http://localhost:8080',  // ✅ 统一访问 Web 网关
      changeOrigin: true
      // 保持 /stream 前缀，由网关转发
    }
  }
}
```

**改进点**：
- ✅ 所有前端请求统一打到 Web 网关（8080）
- ✅ 由网关负责转发到对应的后端服务
- ✅ 避免跨域问题
- ✅ 符合微服务网关架构

### 3. 后端 Web 模块改造

**改造原则**：Web 模块转变为纯网关，删除所有前端渲染相关代码。

#### 3.1 修改文件：`web/start.go`

**删除的内容**：
```go
// ❌ 删除：模板加载
router.LoadHTMLGlob("templates/*.html")
router.Static("/statics/", "./templates")

// ❌ 删除：页面路由
router.GET("/", homeHandler)
router.POST("/", homeHandler)
router.GET("/userhome", userHomeHandler)
router.POST("/userhome", userHomeHandler)

// ❌ 删除：旧的 API 透传路由
router.POST("/api", middleware.APIProxyRateLimiter(), apiHandler)

// ❌ 删除：旧的视频代理路由
router.GET("/videos/:vid-id", middleware.VideoProxyRateLimiter(), proxyVideoViewHandler)
router.POST("/videos/upload/:vid-id", middleware.VideoProxyRateLimiter(), proxyUploadHandler)
```

**新增的内容**：
```go
// ✅ 新增：API 代理路由组（匹配所有 /api/* 请求）
apiGroup := router.Group("/api", middleware.APIProxyRateLimiter())
{
    apiGroup.Any("/*path", proxyToAPIHandler)
}

// ✅ 新增：Stream 代理路由组（匹配所有 /stream/* 请求）
streamGroup := router.Group("/stream", middleware.VideoProxyRateLimiter())
{
    streamGroup.Any("/*path", proxyToStreamHandler)
}
```

**改进点**：
- ✅ 使用路由组（Group）统一管理
- ✅ 使用通配符（/*path）匹配所有子路径
- ✅ 支持所有 HTTP 方法（Any）
- ✅ 自动应用限流和熔断中间件

#### 3.2 修改文件：`web/handlers.go`

**删除的内容**：
```go
// ❌ 删除：模板渲染相关的数据结构
type HomePage struct { Name string }
type UserHomePage struct { Name string }

// ❌ 删除：页面处理器
func homeHandler(c *gin.Context) { ... }
func userHomeHandler(c *gin.Context) { ... }

// ❌ 删除：旧的 API 透传处理器
func apiHandler(c *gin.Context) { ... }

// ❌ 删除：旧的视频代理处理器
func proxyUploadHandler(c *gin.Context) { ... }
func proxyVideoViewHandler(c *gin.Context) { ... }
```

**新增的内容**：
```go
// ✅ 新增：通用的 API 代理处理器（支持所有路径和方法）
func proxyToAPIHandler(c *gin.Context) {
    // 获取路径参数
    path := c.Param("path")
    
    // 通过熔断器执行代理
    circuitBreaker.Call(func() error {
        return proxyToAPI(c, traceID, path)
    })
    
    // 记录 Prometheus 指标
}

// ✅ 新增：通用的 Stream 代理处理器（支持所有路径和方法）
func proxyToStreamHandler(c *gin.Context) {
    // 获取路径参数
    path := c.Param("path")
    
    // 通过熔断器执行代理
    circuitBreaker.Call(func() error {
        return proxyToStream(c, traceID, path)
    })
    
    // 记录 Prometheus 指标
}

// ✅ 新增：反向代理到 API 服务
func proxyToAPI(c *gin.Context, traceID string, path string) error {
    // 创建反向代理
    proxy := httputil.NewSingleHostReverseProxy(u)
    
    // 修改请求路径（去掉 /api 前缀）
    c.Request.URL.Path = path
    
    // 执行代理
    proxy.ServeHTTP(c.Writer, c.Request)
}

// ✅ 新增：反向代理到 Stream 服务
func proxyToStream(c *gin.Context, traceID string, path string) error {
    // 创建反向代理
    proxy := httputil.NewSingleHostReverseProxy(u)
    
    // 修改请求路径（去掉 /stream 前缀）
    c.Request.URL.Path = path
    
    // 执行代理
    proxy.ServeHTTP(c.Writer, c.Request)
}
```

**改进点**：
- ✅ 使用 Go 标准库的反向代理（httputil.ReverseProxy）
- ✅ 代码更简洁，性能更好
- ✅ 自动处理请求头和响应头
- ✅ 支持所有 HTTP 方法和路径

#### 3.3 修改文件：`web/client.go`

**删除的内容**：
```go
// ❌ 删除：手动构建 HTTP 请求的相关函数
func apiRequestProcess(apiBody *ApiBody, ...) { ... }
func doAPIRequest(apiBody *ApiBody, ...) { ... }
func writeErrorResponse(...) { ... }

// ❌ 删除：HTTP 客户端（不再需要）
var httpClient *http.Client
```

**保留的内容**：
```go
// ✅ 保留：获取后端服务地址的辅助函数
func getAPIAddr() string { ... }
func getStreamAddr() string { ... }
```

**改进点**：
- ✅ 不再手动构建 HTTP 请求
- ✅ 使用反向代理，更高效、更可靠
- ✅ 代码量大幅减少

#### 3.4 修改文件：`web/defs.go`

**删除的内容**：
```go
// ❌ 删除：API 透传相关的数据结构（不再需要）
type ApiBody struct {
    Url     string `json:"url"`
    Method  string `json:"method"`
    ReqBody string `json:"req_body"`
}
```

**改进点**：
- ✅ Web 模块不再定义业务数据结构
- ✅ 作为纯网关，只负责转发

## 📊 架构对比

### 改造前（后端渲染）

```
┌─────────┐
│ Browser │
└────┬────┘
     │
     ▼
┌──────────────┐
│ Web Service  │ ← 渲染 HTML 模板
│  (Port 8080) │ ← 处理页面路由
└────┬────┬────┘
     │    │
     │    └──────────┐
     ▼               ▼
┌──────────┐   ┌──────────┐
│   API    │   │  Stream  │
│  (8000)  │   │  (9090)  │
└──────────┘   └──────────┘
```

### 改造后（前后端分离）

```
┌─────────┐
│ Browser │ ← Vue 3 SPA
└────┬────┘
     │
     ▼
┌──────────────┐
│   Frontend   │ ← Vue + Vite
│  (Port 80)   │ ← SPA 应用
└────┬─────────┘
     │ (Proxy)
     ▼
┌──────────────┐
│ Web Gateway  │ ← 纯网关（无渲染）
│  (Port 8080) │ ← 限流、熔断、metrics
└────┬────┬────┘
     │    │
     │    └──────────┐
     ▼               ▼
┌──────────┐   ┌──────────┐
│   API    │   │  Stream  │
│  (8000)  │   │  (9090)  │
└──────────┘   └──────────┘
```

## 🚀 请求流程

### 前端 → 后端

1. **用户操作** → 前端 Vue 应用（80端口）
2. **API 调用** → 前端发送请求到 `/api/*` 或 `/stream/*`
3. **Vite 代理** → 请求被代理到 Web 网关（8080端口）
4. **网关处理** → Web 网关应用中间件（限流、熔断、CORS、metrics）
5. **路由匹配** → 根据前缀匹配到对应的路由组
   - `/api/*` → API 服务（8000）
   - `/stream/*` → Stream 服务（9090）
6. **反向代理** → 使用 httputil.ReverseProxy 转发请求
7. **返回响应** → 逐层返回到前端

### 示例

**前端请求**：
```javascript
// 前端代码
axios.get('/api/videos')
```

**请求链路**：
```
1. Frontend (80) 发起：GET /api/videos
2. Vite 代理转发：   GET http://localhost:8080/api/videos
3. Web Gateway 接收：GET /api/videos
4. 路由匹配：        apiGroup.Any("/*path") → path = "/videos"
5. 代理转发：        GET http://localhost:8000/videos
6. API Service 处理：返回视频列表 JSON
7. 逐层返回：        Web → Vite → Frontend
```

## 🎯 改进优势

### 1. 架构优势
- ✅ **职责分离**：前端负责 UI，后端负责 API
- ✅ **独立部署**：前后端可以独立开发、测试、部署
- ✅ **技术栈解耦**：前端可以自由选择技术栈
- ✅ **符合微服务**：Web 作为统一网关

### 2. 开发优势
- ✅ **开发效率**：Vite 热更新极快
- ✅ **代码组织**：Vue 组件化，代码清晰
- ✅ **状态管理**：Pinia 统一管理状态
- ✅ **类型安全**：可扩展到 TypeScript

### 3. 用户体验优势
- ✅ **现代化 UI**：Element Plus 美观易用
- ✅ **无刷新跳转**：SPA 路由体验流畅
- ✅ **响应式布局**：适配移动端
- ✅ **友好提示**：完善的加载和错误提示

### 4. 性能优势
- ✅ **反向代理**：比手动转发更高效
- ✅ **资源分离**：静态资源和 API 分离
- ✅ **缓存优化**：静态资源可以 CDN 加速
- ✅ **代码分割**：Vue 按需加载

### 5. 运维优势
- ✅ **Docker 支持**：前后端都有 Dockerfile
- ✅ **独立扩容**：可以独立扩展前端或后端
- ✅ **监控完善**：保留了 Prometheus 监控
- ✅ **日志追踪**：保留了 TraceID 链路追踪

## 📝 配置说明

### 前端配置（开发环境）
```bash
cd /home/yang/Desktop/self_learn/simple_video_server_front
npm install
npm run dev  # 启动在 80 端口
```

### 后端配置（不变）
```bash
cd /home/yang/Desktop/self_learn/video-server

# 启动所有服务
make start

# 或者单独启动
./bin/web-service     # 8080
./bin/api-service     # 8000
./bin/stream-service  # 9090
```

## 🔧 注意事项

1. **端口占用**
   - 前端开发服务器：80（需要 root 权限或修改为其他端口）
   - Web 网关：8080
   - API 服务：8000
   - Stream 服务：9090

2. **CORS 配置**
   - Web 网关已配置 CORS 中间件
   - 允许前端域名访问

3. **Token 认证**
   - 前端使用 JWT Token
   - Token 存储在 localStorage
   - 请求自动添加 `X-Session-Id` header

4. **生产部署**
   - 前端：`npm run build` 后部署 dist 目录到 Nginx
   - 后端：使用 Docker 或直接运行二进制文件
   - Nginx 配置示例在 `simple_video_server_front/nginx.conf`

## 🎓 总结

本次改造成功地将一个后端渲染的单体应用改造为现代化的前后端分离架构：

1. ✅ **前端独立**：完整的 Vue 3 项目
2. ✅ **网关优化**：Web 模块转为纯网关
3. ✅ **架构清晰**：前后端职责明确
4. ✅ **代码简化**：删除冗余代码
5. ✅ **功能完整**：保留所有原有功能
6. ✅ **体验提升**：现代化 UI 和交互

这是一个**生产可用**的前后端分离项目！🎉
