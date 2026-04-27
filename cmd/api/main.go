package main

import (
	"context"
	"net/http"
	"time"

	"video-server/api"
	"video-server/api/dbops"
	"video-server/api/utils"
	"video-server/internal/config"
	"video-server/internal/health"
	"video-server/internal/shutdown"
	"video-server/stream"

	"go.uber.org/zap"
)

// 为什么要独立的main.go？
// 1. 可以独立编译成独立的二进制文件
// 2. 可以独立部署（不影响其他服务）
// 3. 可以独立扩容（只扩API服务，不扩Stream）
// 4. 可以独立监控和日志
// 5. 故障隔离（API崩了不影响Stream）

func main() {
	// ========== 第1步：加载配置 ==========
	configPath := "./config/config.json"
	config.MustLoad(configPath)

	// 设置服务名称（用于日志和监控）
	serviceName := "api-service"
	config.AppConfig.ServiceName = serviceName

	// ========== 第2步：初始化日志 ==========
	utils.InitLogging()
	utils.Logger.Info("服务启动中", zap.String("service", serviceName))

	// ========== 第3步：初始化依赖 ==========
	// 初始化数据库连接
	stream.InitOSSClient()
	if err := dbops.Init(); err != nil {
		utils.Logger.Fatal("数据库初始化失败",
			zap.String("service", serviceName),
			zap.Error(err))
	}

	// 获取数据库实例（用于关闭）
	sqlDB, err := dbops.Db.DB()
	if err != nil {
		utils.Logger.Fatal("获取数据库实例失败",
			zap.String("service", serviceName),
			zap.Error(err))
	}

	// ========== 第4步：创建HTTP服务器 ==========
	router := api.RegisterHandlers()

	// 注册健康检查路由
	healthChecker := health.NewHealthChecker(serviceName)
	healthChecker.RegisterRoutes(router)

	// 创建HTTP服务器
	addr := config.AppConfig.APIAddr
	if addr == "" {
		addr = ":8000" // 默认端口
	}
	server := &http.Server{
		Addr:           addr,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// ========== 第5步：启动服务器（非阻塞） ==========
	go func() {
		utils.Logger.Info("HTTP服务器启动",
			zap.String("service", serviceName),
			zap.String("addr", addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			utils.Logger.Fatal("服务器启动失败",
				zap.String("service", serviceName),
				zap.Error(err))
		}
	}()

	// 等待服务器真正启动
	time.Sleep(100 * time.Millisecond)

	// 设置为就绪状态（开始接收流量）
	healthChecker.SetReady()
	utils.Logger.Info("服务已就绪，开始接收请求",
		zap.String("service", serviceName))

	// ========== 第6步：优雅关闭 ==========
	shutdownManager := shutdown.New(serviceName, 30*time.Second)

	// 注册关闭回调：先停止接收新请求
	shutdownManager.Register(func(ctx context.Context) error {
		healthChecker.SetNotReady()
		utils.Logger.Info("已标记为未就绪，停止接收新流量",
			zap.String("service", serviceName))
		return nil
	})

	// 注册关闭回调：关闭HTTP服务器（等待现有请求处理完）
	shutdownManager.Register(func(ctx context.Context) error {
		utils.Logger.Info("正在关闭HTTP服务器",
			zap.String("service", serviceName))
		return server.Shutdown(ctx)
	})

	// 注册关闭回调：关闭数据库连接
	shutdownManager.Register(func(ctx context.Context) error {
		utils.Logger.Info("正在关闭数据库连接",
			zap.String("service", serviceName))
		return sqlDB.Close()
	})

	// 等待关闭信号（阻塞）
	shutdownManager.Wait()

	utils.Logger.Info("服务已停止",
		zap.String("service", serviceName))
}

// 对比旧版本的差异：
//
// 旧版本（main.go）：
//   go api.Start()  // 直接启动，无法控制生命周期
//
// 新版本（cmd/api/main.go）：
//   1. 独立进程（可单独部署）
//   2. 配置加载（支持环境变量覆盖）
//   3. 健康检查（K8s可以探测服务状态）
//   4. 优雅关闭（不丢失正在处理的请求）
//   5. 超时控制（防止hang住）
//   6. 日志结构化（方便排查问题）
