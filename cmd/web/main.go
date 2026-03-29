package main

import (
	"context"
	"net/http"
	"time"

	"video-server/api/utils"
	"video-server/internal/config"
	"video-server/internal/health"
	"video-server/internal/shutdown"
	"video-server/web"

	"go.uber.org/zap"
)

func main() {
	// ========== 第1步：加载配置 ==========
	configPath := "./config/config.json"
	config.MustLoad(configPath)

	serviceName := "web-service"
	config.AppConfig.ServiceName = serviceName

	// ========== 第2步：初始化日志 ==========
	utils.InitLogging()
	utils.Logger.Info("服务启动中", zap.String("service", serviceName))

	// ========== 第3步：创建HTTP服务器 ==========
	router := web.RegisterHandlers()

	// 注册健康检查
	healthChecker := health.NewHealthChecker(serviceName)
	healthChecker.RegisterRoutes(router)

	// 启动限流器清理任务（后台运行）
	go web.CleanupRateLimiters()

	addr := config.AppConfig.WebAddr
	if addr == "" {
		addr = ":8080"
	}
	server := &http.Server{
		Addr:           addr,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// ========== 第4步：启动服务器 ==========
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

	time.Sleep(100 * time.Millisecond)
	healthChecker.SetReady()
	utils.Logger.Info("服务已就绪",
		zap.String("service", serviceName))

	// ========== 第5步：优雅关闭 ==========
	shutdownManager := shutdown.New(serviceName, 30*time.Second)

	shutdownManager.Register(func(ctx context.Context) error {
		healthChecker.SetNotReady()
		utils.Logger.Info("已标记为未就绪",
			zap.String("service", serviceName))
		return nil
	})

	shutdownManager.Register(func(ctx context.Context) error {
		utils.Logger.Info("正在关闭HTTP服务器",
			zap.String("service", serviceName))
		return server.Shutdown(ctx)
	})

	shutdownManager.Wait()
	utils.Logger.Info("服务已停止",
		zap.String("service", serviceName))
}
