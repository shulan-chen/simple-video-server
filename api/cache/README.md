# Cache 目录说明

## 📁 目录结构

```
api/cache/
├── cache.go              # 数据缓存（用户、视频、评论等业务数据）
└── session_cache.go      # 会话缓存（Session和Refresh Token）
```

## 🎯 职责划分

### cache.go - 业务数据缓存

**管理的缓存类型**：
- 用户信息（User）
- 视频信息（Video）
- 视频列表（全部/用户）
- 评论列表（Comments）

**缓存键前缀**：
```go
data:user:<userId>           // 用户信息（使用userId避免重名）
data:video:<vid>             // 视频信息
data:videos:all:latest       // 全部视频列表
data:videos:user:<userId>    // 用户视频列表
data:comments:<vid>          // 评论列表
```

**缓存时间**：
- 用户信息：1小时
- 视频信息：30分钟
- 视频列表：10分钟（热门数据）
- 评论列表：5分钟

### session_cache.go - 会话缓存

**管理的缓存类型**：
- Session会话（已废弃，保留代码兼容性）
- Refresh Token（JWT刷新令牌）

**缓存键前缀**：
```go
session:<sid>                // 单个Session（已废弃）
session:list:<username>      // Session列表（已废弃）
session:refresh:<token>      // Refresh Token（当前使用）
```

**缓存时间**：
- Session：15分钟（已废弃）
- Refresh Token：7天

## 🔄 历史变更

### 原有结构（已废弃）
```
api/session/
├── ops.go              # Session管理（sync.Map + Redis混合）
├── redisCache.go       # Redis操作（已删除）
└── refresh_token.go    # Refresh Token（已删除）
```

### 新结构（当前）
```
api/cache/
├── cache.go            # 数据缓存
└── session_cache.go    # 会话缓存
```

**迁移原因**：
1. ✅ **统一Redis操作**：所有Redis操作集中在cache目录
2. ✅ **命名清晰**：通过文件名区分不同职责
3. ✅ **避免重复连接**：共用同一个redisClient实例
4. ✅ **版本统一**：统一使用 go-redis v9

## 🔧 使用示例

### 数据缓存

```go
import "video-server/api/cache"

// 用户信息（使用userId）
user, err := cache.GetUserById(ctx, userId)
cache.SetUser(ctx, user)
cache.DeleteUserById(ctx, userId)

// 视频列表
videos, err := cache.GetAllVideos(ctx)
cache.SetAllVideos(ctx, videos)
cache.InvalidateAllVideos(ctx)

// 评论列表
comments, err := cache.GetComments(ctx, vid)
cache.SetComments(ctx, vid, comments)
cache.InvalidateComments(ctx, vid)
```

### 会话缓存

```go
import "video-server/api/cache"

// Refresh Token管理
cache.SaveRefreshToken(ctx, token, userId)
userId, err := cache.ValidateRefreshToken(ctx, token)
cache.DeleteRefreshToken(ctx, token)
cache.RotateRefreshToken(ctx, oldToken, newToken, userId)
```

## 📊 性能指标

| 操作 | 延迟 | 吞吐量 |
|------|------|--------|
| Redis GET | ~1ms | 10万QPS+ |
| Redis SET | ~1ms | 10万QPS+ |
| MySQL查询 | ~50ms | 5千QPS |

**缓存命中率目标**：80%+

## ⚠️ 注意事项

1. **UserInfo缓存key使用userId**：避免重名问题
2. **统一使用cache包**：不要直接操作redisClient
3. **缓存失效策略**：写操作必须使相关缓存失效
4. **降级策略**：Redis失败时不阻塞请求，降级到DB

## 🔗 相关文档

- [PERFORMANCE_OPTIMIZATION.md](../../documents/PERFORMANCE_OPTIMIZATION.md) - 性能优化总览
- [JWT_BEST_PRACTICE.md](../../documents/JWT_BEST_PRACTICE.md) - JWT实践（包含Refresh Token）
