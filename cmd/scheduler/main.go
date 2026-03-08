package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"video-server/api/dbops"
	"video-server/api/utils"
	"video-server/internal/config"
	"video-server/internal/health"
	"video-server/internal/shutdown"
	"video-server/scheduler"

	"github.com/gin-gonic/gin"
)

// Scheduler服务的特殊之处：
// 1. 没有对外提供HTTP API（只有健康检查）
// 2. 主要工作是后台定时任务
// 3. 需要优雅关闭（等待正在执行的任务完成）

func main() {
	// ========== 第1步：加载配置 ==========
	configPath := "./config/config.json"
	config.MustLoad(configPath)

	serviceName := "scheduler-service"
	config.AppConfig.ServiceName = serviceName

	// ========== 第2步：初始化日志 ==========
	utils.InitLogging()
	log.Printf("[%s] 服务启动中...", serviceName)

	// ========== 第3步：初始化依赖 ==========
	// 初始化数据库连接
	if err := dbops.Init(); err != nil {
		log.Fatalf("[%s] 数据库初始化失败: %v", serviceName, err)
	}

	// 获取数据库实例（用于关闭）
	sqlDB, err := dbops.Db.DB()
	if err != nil {
		log.Fatalf("[%s] 获取数据库实例失败: %v", serviceName, err)
	}

	// ========== 第4步：启动定时任务 ==========
	runner := scheduler.NewRunner(3, true, scheduler.VideoClearDispatcher, scheduler.VideoClearExecutor)
	worker := scheduler.NewWorker(time.Duration(config.AppConfig.VideoDeleteDelayTime)*time.Second, runner)

	// 启动worker（非阻塞）
	go func() {
		log.Printf("[%s] 定时任务启动，间隔: %ds", serviceName, config.AppConfig.VideoDeleteDelayTime)
		worker.StartWorker()
	}()

	// ========== 第5步：启动健康检查服务器 ==========
	// 为什么Scheduler也需要HTTP服务器？
	// 1. 提供健康检查接口（K8s需要）
	// 2. 可以扩展管理接口（暂停/恢复任务）
	// 3. 可以暴露metrics（任务执行情况）

	router := gin.Default()
	healthChecker := health.NewHealthChecker(serviceName)
	healthChecker.RegisterRoutes(router)

	// 可选：添加任务管理接口
	router.GET("/tasks/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": serviceName,
			"status":  "running",
			"uptime":  time.Since(worker.StartTime).String(),
		})
	})

	addr := ":8001" // Scheduler的管理端口
	server := &http.Server{
		Addr:           addr,
		Handler:        router,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   5 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		log.Printf("[%s] 管理服务器启动在 %s", serviceName, addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[%s] 服务器启动失败: %v", serviceName, err)
		}
	}()

	time.Sleep(100 * time.Millisecond)
	healthChecker.SetReady()
	log.Printf("[%s] 服务已就绪", serviceName)

	// ========== 第6步：优雅关闭 ==========
	shutdownManager := shutdown.New(serviceName, 30*time.Second)

	shutdownManager.Register(func(ctx context.Context) error {
		healthChecker.SetNotReady()
		log.Printf("[%s] 已标记为未就绪，停止接收新任务", serviceName)
		return nil
	})

	shutdownManager.Register(func(ctx context.Context) error {
		log.Printf("[%s] 正在停止定时任务...", serviceName)
		worker.Stop() // 需要在scheduler包中添加Stop方法
		return nil
	})

	shutdownManager.Register(func(ctx context.Context) error {
		log.Printf("[%s] 正在关闭HTTP服务器...", serviceName)
		return server.Shutdown(ctx)
	})

	shutdownManager.Register(func(ctx context.Context) error {
		log.Printf("[%s] 正在关闭数据库连接...", serviceName)
		return sqlDB.Close()
	})

	shutdownManager.Wait()
	log.Printf("[%s] 服务已停止", serviceName)
}
