package utils

import "net/http"

// AppError 统一错误结构
type AppError struct {
	Code    int    `json:"code"`               // 错误码
	Message string `json:"message"`            // 用户友好的错误信息
	Detail  string `json:"detail,omitempty"`   // 详细错误（仅dev环境）
	TraceID string `json:"trace_id,omitempty"` // 链路追踪ID
}

// 错误码设计：XXYYZZ
// XX: 服务类型（10=API, 20=Stream, 30=Scheduler, 40=Web）
// YY: 错误类别（01=参数, 02=认证, 03=数据库, 04=业务逻辑, 05=外部服务, 06=限流, 99=内部错误）
// ZZ: 具体错误序号

const (
	// ========== API服务错误 (10YYZZ) ==========

	// 参数错误 (1001XX)
	ErrAPIInvalidRequest     = 100101 // 请求体解析失败
	ErrAPIInvalidJSON        = 100102 // JSON格式错误
	ErrAPIMissingParam       = 100103 // 缺少必需参数
	ErrAPIInvalidParam       = 100104 // 参数格式错误
	ErrAPIInvalidVideoName   = 100105 // 视频名称非法
	ErrAPIInvalidCommentLen  = 100106 // 评论长度非法
	ErrAPIInvalidPasswordLen = 100107 // 密码长度非法

	// 认证/鉴权错误 (1002XX)
	ErrAPIUnauthorized        = 100201 // 未认证
	ErrAPITokenExpired        = 100202 // Token过期
	ErrAPITokenInvalid        = 100203 // Token无效
	ErrAPIUserNotFound        = 100204 // 用户不存在
	ErrAPIPasswordWrong       = 100205 // 密码错误
	ErrAPIForbidden           = 100206 // 无权限访问
	ErrAPIRefreshTokenInvalid = 100207 // Refresh Token无效

	// 数据库错误 (1003XX)
	ErrAPIDBConnection = 100301 // 数据库连接失败
	ErrAPIDBQuery      = 100302 // 数据库查询失败
	ErrAPIDBInsert     = 100303 // 数据库插入失败
	ErrAPIDBUpdate     = 100304 // 数据库更新失败
	ErrAPIDBDelete     = 100305 // 数据库删除失败
	ErrAPIDBTimeout    = 100306 // 数据库超时
	ErrAPIDBDeadlock   = 100307 // 数据库死锁

	// 业务逻辑错误 (1004XX)
	ErrAPIUserExisted          = 100401 // 用户已存在
	ErrAPIVideoNotExisted      = 100402 // 视频不存在
	ErrAPIVideoNotBelongToUser = 100403 // 视频不属于该用户
	ErrAPICommentNotExisted    = 100404 // 评论不存在
	ErrAPISelfOperation        = 100405 // 不能操作自己（如关注自己）

	// 外部服务错误 (1005XX)
	ErrAPIRedisConnection = 100501 // Redis连接失败
	ErrAPIRedisOperation  = 100502 // Redis操作失败
	ErrAPICacheExpired    = 100503 // 缓存过期

	// 限流错误 (1006XX)
	ErrAPIRateLimitExceeded = 100601 // 超过请求频率限制

	// 内部错误 (1099XX)
	ErrAPIInternal      = 109901 // 内部错误
	ErrAPIUUIDGenerate  = 109902 // UUID生成失败
	ErrAPIEncryptFailed = 109903 // 加密失败
	ErrAPITokenGenerate = 109904 // Token生成失败

	// ========== Stream服务错误 (20YYZZ) ==========

	// 参数错误 (2001XX)
	ErrStreamInvalidRequest = 200101 // 请求参数错误
	ErrStreamFileTooLarge   = 200102 // 文件过大
	ErrStreamInvalidFormat  = 200103 // 文件格式错误
	ErrStreamMissingFile    = 200104 // 缺少文件

	// 文件操作错误 (2003XX)
	ErrStreamFileRead   = 200301 // 文件读取失败
	ErrStreamFileWrite  = 200302 // 文件写入失败
	ErrStreamFileCreate = 200303 // 文件创建失败
	ErrStreamFileDelete = 200304 // 文件删除失败

	// OSS错误 (2005XX)
	ErrStreamOSSConnection = 200501 // OSS连接失败
	ErrStreamOSSUpload     = 200502 // OSS上传失败
	ErrStreamOSSDownload   = 200503 // OSS下载失败
	ErrStreamOSSDelete     = 200504 // OSS删除失败
	ErrStreamOSSSignURL    = 200505 // OSS签名URL生成失败

	// 限流错误 (2006XX)
	ErrStreamRateLimitExceeded = 200601 // 超过上传频率限制
	ErrStreamConcurrentLimit   = 200602 // 超过并发连接限制

	// 内部错误 (2099XX)
	ErrStreamInternal = 209901 // 内部错误

	// ========== Scheduler服务错误 (30YYZZ) ==========

	// 业务逻辑错误 (3004XX)
	ErrSchedulerTaskFailed = 300401 // 定时任务执行失败

	// 数据库错误 (3003XX)
	ErrSchedulerDBError = 300301 // 数据库操作失败

	// 内部错误 (3099XX)
	ErrSchedulerInternal = 309901 // 内部错误

	// ========== Web服务错误 (40YYZZ) ==========

	// 请求错误 (4001XX)
	ErrWebInvalidRequest     = 400101 // 请求格式错误
	ErrWebRequestBodyInvalid = 400102 // 请求体解析失败
	ErrWebMethodNotAllowed   = 400103 // 请求方法不支持

	// 代理错误 (4002XX)
	ErrWebProxyFailed     = 400201 // 代理请求失败
	ErrWebProxyTimeout    = 400202 // 代理请求超时
	ErrWebInvalidProxyURL = 400203 // 代理URL无效

	// 模板错误 (4003XX)
	ErrWebTemplateRender = 400301 // 模板渲染失败

	// 限流错误 (4006XX)
	ErrWebRateLimitExceeded = 400601 // 请求频率超限

	// 熔断错误 (4007XX)
	ErrWebCircuitBreakerOpen = 400701 // 熔断器打开

	// 内部错误 (4099XX)
	ErrWebInternalFault = 409901 // 内部服务错误
)

// 错误码到HTTP状态码的映射
var errorHTTPStatus = map[int]int{
	// API - 参数错误 -> 400
	ErrAPIInvalidRequest:     http.StatusBadRequest,
	ErrAPIInvalidJSON:        http.StatusBadRequest,
	ErrAPIMissingParam:       http.StatusBadRequest,
	ErrAPIInvalidParam:       http.StatusBadRequest,
	ErrAPIInvalidVideoName:   http.StatusBadRequest,
	ErrAPIInvalidCommentLen:  http.StatusBadRequest,
	ErrAPIInvalidPasswordLen: http.StatusBadRequest,

	// API - 认证错误 -> 401
	ErrAPIUnauthorized:        http.StatusUnauthorized,
	ErrAPITokenExpired:        http.StatusUnauthorized,
	ErrAPITokenInvalid:        http.StatusUnauthorized,
	ErrAPIPasswordWrong:       http.StatusUnauthorized,
	ErrAPIRefreshTokenInvalid: http.StatusUnauthorized,

	// API - 资源不存在 -> 404
	ErrAPIUserNotFound:      http.StatusNotFound,
	ErrAPIVideoNotExisted:   http.StatusNotFound,
	ErrAPICommentNotExisted: http.StatusNotFound,

	// API - 权限错误 -> 403
	ErrAPIForbidden:            http.StatusForbidden,
	ErrAPIVideoNotBelongToUser: http.StatusForbidden,

	// API - 业务逻辑错误 -> 409 (Conflict)
	ErrAPIUserExisted:   http.StatusConflict,
	ErrAPISelfOperation: http.StatusConflict,

	// API - 数据库/外部服务错误 -> 500
	ErrAPIDBConnection:    http.StatusInternalServerError,
	ErrAPIDBQuery:         http.StatusInternalServerError,
	ErrAPIDBInsert:        http.StatusInternalServerError,
	ErrAPIDBUpdate:        http.StatusInternalServerError,
	ErrAPIDBDelete:        http.StatusInternalServerError,
	ErrAPIDBTimeout:       http.StatusServiceUnavailable,
	ErrAPIDBDeadlock:      http.StatusInternalServerError,
	ErrAPIRedisConnection: http.StatusInternalServerError,
	ErrAPIRedisOperation:  http.StatusInternalServerError,
	ErrAPICacheExpired:    http.StatusInternalServerError,

	// API - 限流 -> 429
	ErrAPIRateLimitExceeded: http.StatusTooManyRequests,

	// API - 内部错误 -> 500
	ErrAPIInternal:      http.StatusInternalServerError,
	ErrAPIUUIDGenerate:  http.StatusInternalServerError,
	ErrAPIEncryptFailed: http.StatusInternalServerError,
	ErrAPITokenGenerate: http.StatusInternalServerError,

	// Stream - 参数错误 -> 400
	ErrStreamInvalidRequest: http.StatusBadRequest,
	ErrStreamFileTooLarge:   http.StatusRequestEntityTooLarge,
	ErrStreamInvalidFormat:  http.StatusBadRequest,
	ErrStreamMissingFile:    http.StatusBadRequest,

	// Stream - 文件操作 -> 500
	ErrStreamFileRead:   http.StatusInternalServerError,
	ErrStreamFileWrite:  http.StatusInternalServerError,
	ErrStreamFileCreate: http.StatusInternalServerError,
	ErrStreamFileDelete: http.StatusInternalServerError,

	// Stream - OSS -> 500/503
	ErrStreamOSSConnection: http.StatusServiceUnavailable,
	ErrStreamOSSUpload:     http.StatusInternalServerError,
	ErrStreamOSSDownload:   http.StatusInternalServerError,
	ErrStreamOSSDelete:     http.StatusInternalServerError,
	ErrStreamOSSSignURL:    http.StatusInternalServerError,

	// Stream - 限流 -> 429
	ErrStreamRateLimitExceeded: http.StatusTooManyRequests,
	ErrStreamConcurrentLimit:   http.StatusTooManyRequests,

	// Stream - 内部错误 -> 500
	ErrStreamInternal: http.StatusInternalServerError,

	// Scheduler
	ErrSchedulerTaskFailed: http.StatusInternalServerError,
	ErrSchedulerDBError:    http.StatusInternalServerError,
	ErrSchedulerInternal:   http.StatusInternalServerError,

	// Web - 请求错误 -> 400
	ErrWebInvalidRequest:     http.StatusBadRequest,
	ErrWebRequestBodyInvalid: http.StatusBadRequest,
	ErrWebMethodNotAllowed:   http.StatusMethodNotAllowed,

	// Web - 代理错误 -> 502/504
	ErrWebProxyFailed:     http.StatusBadGateway,
	ErrWebProxyTimeout:    http.StatusGatewayTimeout,
	ErrWebInvalidProxyURL: http.StatusBadRequest,

	// Web - 模板错误 -> 500
	ErrWebTemplateRender: http.StatusInternalServerError,

	// Web - 限流错误 -> 429
	ErrWebRateLimitExceeded: http.StatusTooManyRequests,

	// Web - 熔断错误 -> 503
	ErrWebCircuitBreakerOpen: http.StatusServiceUnavailable,

	// Web - 内部错误 -> 500
	ErrWebInternalFault: http.StatusInternalServerError,
}

// 错误码到用户友好消息的映射
var errorMessages = map[int]string{
	// API - 参数错误
	ErrAPIInvalidRequest:     "请求格式错误",
	ErrAPIInvalidJSON:        "JSON格式错误",
	ErrAPIMissingParam:       "缺少必需参数",
	ErrAPIInvalidParam:       "参数格式错误",
	ErrAPIInvalidVideoName:   "视频名称格式错误",
	ErrAPIInvalidCommentLen:  "评论长度不符合要求",
	ErrAPIInvalidPasswordLen: "密码长度不符合要求",

	// API - 认证错误
	ErrAPIUnauthorized:        "请先登录",
	ErrAPITokenExpired:        "登录已过期，请重新登录",
	ErrAPITokenInvalid:        "登录凭证无效",
	ErrAPIUserNotFound:        "用户不存在",
	ErrAPIPasswordWrong:       "密码错误",
	ErrAPIForbidden:           "无权限访问",
	ErrAPIRefreshTokenInvalid: "刷新令牌无效",

	// API - 数据库错误
	ErrAPIDBConnection: "数据库连接失败，请稍后重试",
	ErrAPIDBQuery:      "数据查询失败",
	ErrAPIDBInsert:     "数据保存失败",
	ErrAPIDBUpdate:     "数据更新失败",
	ErrAPIDBDelete:     "数据删除失败",
	ErrAPIDBTimeout:    "数据库响应超时",
	ErrAPIDBDeadlock:   "数据库操作冲突，请重试",

	// API - 业务逻辑错误
	ErrAPIUserExisted:          "用户名已存在",
	ErrAPIVideoNotExisted:      "视频不存在",
	ErrAPIVideoNotBelongToUser: "无权操作该视频",
	ErrAPICommentNotExisted:    "评论不存在",
	ErrAPISelfOperation:        "不能对自己执行该操作",

	// API - 外部服务错误
	ErrAPIRedisConnection: "缓存服务连接失败",
	ErrAPIRedisOperation:  "缓存操作失败",
	ErrAPICacheExpired:    "缓存已过期",

	// API - 限流错误
	ErrAPIRateLimitExceeded: "操作过于频繁，请稍后再试",

	// API - 内部错误
	ErrAPIInternal:      "服务器内部错误",
	ErrAPIUUIDGenerate:  "生成ID失败",
	ErrAPIEncryptFailed: "加密失败",
	ErrAPITokenGenerate: "生成令牌失败",

	// Stream - 参数错误
	ErrStreamInvalidRequest: "请求参数错误",
	ErrStreamFileTooLarge:   "文件过大，最大支持100MB",
	ErrStreamInvalidFormat:  "文件格式不支持",
	ErrStreamMissingFile:    "未选择文件",

	// Stream - 文件操作
	ErrStreamFileRead:   "文件读取失败",
	ErrStreamFileWrite:  "文件写入失败",
	ErrStreamFileCreate: "文件创建失败",
	ErrStreamFileDelete: "文件删除失败",

	// Stream - OSS
	ErrStreamOSSConnection: "对象存储连接失败",
	ErrStreamOSSUpload:     "视频上传失败",
	ErrStreamOSSDownload:   "视频下载失败",
	ErrStreamOSSDelete:     "视频删除失败",
	ErrStreamOSSSignURL:    "生成视频链接失败",

	// Stream - 限流
	ErrStreamRateLimitExceeded: "上传过于频繁，请稍后再试",
	ErrStreamConcurrentLimit:   "当前上传人数过多，请稍后再试",

	// Stream - 内部错误
	ErrStreamInternal: "服务器内部错误",

	// Scheduler
	ErrSchedulerTaskFailed: "定时任务执行失败",
	ErrSchedulerDBError:    "数据库操作失败",
	ErrSchedulerInternal:   "调度服务内部错误",

	// Web - 请求错误
	ErrWebInvalidRequest:     "请求格式错误",
	ErrWebRequestBodyInvalid: "请求体解析失败",
	ErrWebMethodNotAllowed:   "请求方法不支持",

	// Web - 代理错误
	ErrWebProxyFailed:     "后端服务请求失败",
	ErrWebProxyTimeout:    "后端服务响应超时",
	ErrWebInvalidProxyURL: "无效的请求地址",

	// Web - 模板错误
	ErrWebTemplateRender: "页面渲染失败",

	// Web - 限流错误
	ErrWebRateLimitExceeded: "请求过于频繁，请稍后再试",

	// Web - 熔断错误
	ErrWebCircuitBreakerOpen: "服务暂时不可用，请稍后再试",

	// Web - 内部错误
	ErrWebInternalFault: "内部服务错误",
}

// NewAppError 创建应用错误
func NewAppError(code int, detail string, traceID string) *AppError {
	message, ok := errorMessages[code]
	if !ok {
		message = "未知错误"
	}

	return &AppError{
		Code:    code,
		Message: message,
		Detail:  detail,
		TraceID: traceID,
	}
}

// GetHTTPStatus 获取错误对应的HTTP状态码
func GetHTTPStatus(code int) int {
	if status, ok := errorHTTPStatus[code]; ok {
		return status
	}
	return http.StatusInternalServerError
}
