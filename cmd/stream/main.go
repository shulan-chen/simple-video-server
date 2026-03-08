package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"video-server/api/utils"
	"video-server/internal/config"
	"video-server/internal/health"
	"video-server/internal/shutdown"
	"video-server/stream"
)

func main() {
	// ========== 第1步：加载配置 ==========
	configPath := "./config/config.json"
	config.MustLoad(configPath)

	serviceName := "stream-service"
	config.AppConfig.ServiceName = serviceName

	// ========== 第2步：初始化日志 ==========
	utils.InitLogging()
	log.Printf("[%s] 服务启动中...", serviceName)

	// ========== 第3步：创建HTTP服务器 ==========
	router := stream.RegisterHandlers()

	// 注册健康检查
	healthChecker := health.NewHealthChecker(serviceName)
	healthChecker.RegisterRoutes(router)

	addr := config.AppConfig.StreamAddr
	if addr == "" {
		addr = ":9090"
	}
	server := &http.Server{
		Addr:           addr,
		Handler:        router,
		ReadTimeout:    60 * time.Second,  // Stream服务需要更长的超时
		WriteTimeout:   60 * time.Second,  // 因为可能传输大文件
		MaxHeaderBytes: 1 << 20,
	}

	// ========== 第4步：启动服务器 ==========
	go func() {
		log.Printf("[%s] HTTP服务器启动在 %s", serviceName, addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[%s] 服务器启动失败: %v", serviceName, err)
		}
	}()

	time.Sleep(100 * time.Millisecond)
	healthChecker.SetReady()
	log.Printf("[%s] 服务已就绪", serviceName)

	// ========== 第5步：优雅关闭 ==========
	shutdownManager := shutdown.New(serviceName, 60*time.Second) // 更长的关闭超时

	shutdownManager.Register(func(ctx context.Context) error {
		healthChecker.SetNotReady()
		log.Printf("[%s] 已标记为未就绪，等待正在上传/下载的请求完成...", serviceName)
		return nil
	})

	shutdownManager.Register(func(ctx context.Context) error {
		log.Printf("[%s] 正在关闭HTTP服务器...", serviceName)
		return server.Shutdown(ctx)
	})

	shutdownManager.Wait()
	log.Printf("[%s] 服务已停止", serviceName)
}
