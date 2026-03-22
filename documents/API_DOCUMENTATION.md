# API文档 - Swagger集成

## 📋 概述

已为项目集成 **Swagger API文档**，解决 CODE_REVIEW 问题11"API文档：不存在"。

## 🎯 解决的问题

### 改进前
```
前端：这个接口怎么调用？
后端：你看代码吧

前端：参数是什么？
后端：你看代码吧

前端：返回格式是什么？
后端：你看代码吧
```

### 改进后
- ✅ **在线文档**：http://localhost:8000/swagger/index.html
- ✅ **交互式测试**：可以直接在浏览器中测试API
- ✅ **自动更新**：代码修改后，重新生成即可
- ✅ **标准格式**：OpenAPI 2.0规范

## 🚀 使用步骤

### 1. 安装Swagger工具

由于Go版本依赖问题，推荐手动安装指定版本：

```bash
# 方式1：使用go install（推荐）
GOPROXY=https://goproxy.cn go install github.com/swaggo/swag/cmd/swag@v1.8.12

# 方式2：下载预编译二进制
# 访问 https://github.com/swaggo/swag/releases
# 下载对应平台的swag二进制文件，放到$PATH中

# 验证安装
swag --version
```

### 2. 添加Swagger依赖到项目

已经在handlers中添加了Swagger注解，现在需要安装运行时依赖：

```bash
# 添加到go.mod（手动方式，避免版本冲突）
cat >> go.mod << 'EOF'

require (
    github.com/swaggo/swag v1.8.12
    github.com/swaggo/gin-swagger v1.5.3
    github.com/swaggo/files v1.0.1
)
EOF

# 然后运行
go mod tidy
```

### 3. 生成Swagger文档

```bash
# 在项目根目录执行
swag init --dir ./api --output ./api/docs --parseDependency --parseInternal

# 参数说明：
# --dir          指定扫描的目录（api目录）
# --output       生成文档的输出目录
# --parseDependency  解析依赖的模块
# --parseInternal    解析internal包
```

**生成的文件**：
```
api/docs/
├── docs.go        # Go代码，自动导入
├── swagger.json   # OpenAPI JSON格式
└── swagger.yaml   # OpenAPI YAML格式
```

### 4. 注册Swagger路由

在 `api/api.go` 中添加：

```go
import (
    swaggerFiles "github.com/swaggo/files"
    ginSwagger "github.com/swaggo/gin-swagger"
    _ "video-server/api/docs"  // 导入生成的docs包
)

func RegisterHandlers() *gin.Engine {
    router := gin.Default()

    // ... 其他中间件和路由 ...

    // Swagger文档路由（放在最后）
    router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

    return router
}
```

### 5. 访问API文档

启动服务后访问：
```
http://localhost:8000/swagger/index.html
```

## 📝 Swagger注解说明

### 主配置注解（api/docs.go）

```go
// @title           Video Server API
// @version         1.0
// @description     视频服务器API文档 - 微服务架构

// @host      localhost:8000
// @BasePath  /

// @securityDefinitions.apikey  Bearer
// @in                          header
// @name                        X-Session-Id
```

### Handler注解示例

```go
// CreateUser 用户注册
// @Summary      用户注册
// @Description  创建新用户账号
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Param        user  body      api.UserDTO  true  "用户信息"
// @Success      201   {object}  map[string]string
// @Failure      400   {object}  utils.AppError
// @Router       /user [post]
func CreateUser(c *gin.Context) {
    // ...
}
```

### 注解字段说明

| 注解 | 说明 | 示例 |
|------|------|------|
| @Summary | 简短描述 | "用户注册" |
| @Description | 详细描述 | "创建新用户账号" |
| @Tags | 分组标签 | "用户管理" |
| @Accept | 接受的Content-Type | "json" |
| @Produce | 返回的Content-Type | "json" |
| @Param | 参数定义 | `user body api.UserDTO true "用户信息"` |
| @Success | 成功响应 | `200 {object} api.User` |
| @Failure | 失败响应 | `400 {object} utils.AppError` |
| @Router | 路由路径 | `/user [post]` |
| @Security | 安全认证 | "Bearer" |

## 📊 已添加文档的接口

### 用户管理
- `POST /user` - 用户注册
- `POST /user/{user_name}` - 用户登录
- `GET /user/{user_name}` - 获取用户信息
- `POST /user/{user_name}/logout` - 用户登出

### 认证
- `POST /auth/refresh` - 刷新Token

### 视频管理
- `POST /user/{user_name}/videos` - 添加视频
- `GET /user/{user_name}/videos` - 获取用户视频列表
- `DELETE /user/{user_name}/videos/{vid}` - 删除视频
- `GET /videos` - 获取所有视频

### 评论管理
- `POST /videos/{vid}/comments` - 发表评论
- `GET /videos/{vid}/comments` - 获取评论列表

## 🎨 Swagger UI 功能

### 1. 查看所有接口
- 按Tag分组（用户管理、视频管理、评论管理等）
- 查看请求参数、响应格式
- 查看错误码

### 2. 在线测试
```
1. 点击接口 → "Try it out"
2. 填写参数
3. 点击 "Execute"
4. 查看响应结果
```

### 3. 认证测试
```
1. 先调用 /user/{user_name} 登录，获取 access_token
2. 点击页面右上角 "Authorize"
3. 输入 access_token
4. 现在可以测试需要认证的接口
```

## 🔄 开发流程

### 添加新接口时

1. **写Handler函数**
```go
func NewAPI(c *gin.Context) {
    // 实现逻辑
}
```

2. **添加Swagger注解**
```go
// NewAPI 新接口
// @Summary      新接口
// @Description  详细描述
// @Tags         分组名称
// @Accept       json
// @Produce      json
// @Param        xxx  path  string  true  "参数说明"
// @Success      200  {object}  ResponseType
// @Router       /path [method]
func NewAPI(c *gin.Context) {
    // ...
}
```

3. **重新生成文档**
```bash
swag init --dir ./api --output ./api/docs --parseDependency --parseInternal
```

4. **查看更新**
访问 http://localhost:8000/swagger/index.html

## 💡 最佳实践

### 1. 注解编写

✅ **应该做**：
- Summary简短（1-3个词）
- Description详细说明用途和注意事项
- 所有参数都加上说明
- 列出常见的错误码

❌ **不应该做**：
- 注解和实际代码不一致
- 缺少必需参数的说明
- 没有说明认证要求

### 2. 响应结构

✅ **使用结构体**：
```go
// @Success 200 {object} api.User
```

✅ **使用map（简单响应）**：
```go
// @Success 200 {object} map[string]string
```

### 3. 错误文档

详细列出所有可能的错误码：
```go
// @Failure 400 {object} utils.AppError  "请求参数错误"
// @Failure 401 {object} utils.AppError  "未认证"
// @Failure 403 {object} utils.AppError  "无权限"
// @Failure 404 {object} utils.AppError  "资源不存在"
// @Failure 429 {object} utils.AppError  "请求过于频繁"
// @Failure 500 {object} utils.AppError  "服务器内部错误"
```

## 🆚 对比Spring Boot

### Spring Boot（Java）

```java
@ApiOperation(value = "用户注册", notes = "创建新用户")
@ApiResponses({
    @ApiResponse(code = 201, message = "注册成功"),
    @ApiResponse(code = 400, message = "参数错误")
})
@PostMapping("/user")
public Response createUser(@RequestBody @Valid UserDTO user) {
    // ...
}
```

### Gin + Swaggo（Go）

```go
// @Summary      用户注册
// @Description  创建新用户账号
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Param        user  body      api.UserDTO  true  "用户信息"
// @Success      201   {object}  map[string]string
// @Failure      400   {object}  utils.AppError
// @Router       /user [post]
func CreateUser(c *gin.Context) {
    // ...
}
```

**相似点**：
- ✅ 都是基于注解/注释
- ✅ 都支持在线测试
- ✅ 都符合OpenAPI规范

**差异点**：
- Go使用注释（`//`），Java使用注解（`@`）
- Go需要手动运行`swag init`生成文档
- Java通过SpringFox自动生成

**复杂度**：**一样简单！** 🎉

## 📦 CI/CD集成

### Makefile添加生成命令

```makefile
# 生成Swagger文档
.PHONY: swagger
swagger:
	@echo "生成Swagger文档..."
	@swag init --dir ./api --output ./api/docs --parseDependency --parseInternal
	@echo "✅ Swagger文档生成完成！访问 http://localhost:8000/swagger/index.html"

# 构建前先生成文档
build: swagger
	@echo "构建所有服务..."
	# ...
```

### Git忽略生成的文档

```bash
# .gitignore
api/docs/
```

**原因**：文档是自动生成的，不应该提交到Git。

## 🎉 总结

通过集成Swagger：
- ✅ **开发效率提升**：前后端不再需要口头沟通接口
- ✅ **减少沟通成本**：文档自动生成，永远最新
- ✅ **提升专业度**：标准化API文档
- ✅ **便于测试**：在线测试所有接口

**从CODE_REVIEW的"不存在"提升到"生产级API文档"！** 🚀

## 📚 相关资源

- [Swaggo官方文档](https://github.com/swaggo/swag)
- [Swaggo注解说明](https://github.com/swaggo/swag#declarative-comments-format)
- [OpenAPI规范](https://swagger.io/specification/)
