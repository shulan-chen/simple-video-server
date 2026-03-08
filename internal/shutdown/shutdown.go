package shutdown

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// 为什么需要优雅关闭？
// 1. 正在处理的请求要处理完（不能直接kill）
// 2. 数据库连接要关闭（防止连接泄露）
// 3. 正在写入的文件要关闭（防止数据损坏）
// 4. 告诉负载均衡器"我要下线了"（防止新流量进来）

// GracefulShutdown 优雅关闭管理器
type GracefulShutdown struct {
	serviceName string
	timeout     time.Duration
	callbacks   []func(ctx context.Context) error
}

// New 创建优雅关闭管理器
func New(serviceName string, timeout time.Duration) *GracefulShutdown {
	return &GracefulShutdown{
		serviceName: serviceName,
		timeout:     timeout,
		callbacks:   make([]func(ctx context.Context) error, 0),
	}
}

// Register 注册关闭回调
// 注册顺序就是执行顺序（先注册的先执行）
func (g *GracefulShutdown) Register(callback func(ctx context.Context) error) {
	g.callbacks = append(g.callbacks, callback)
}

// Wait 等待关闭信号
// 返回后，服务应该立即停止接收新请求
func (g *GracefulShutdown) Wait() {
	quit := make(chan os.Signal, 1)
	// 监听 SIGINT (Ctrl+C) 和 SIGTERM (kill)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	log.Printf("[%s] 收到关闭信号: %v, 开始优雅关闭...", g.serviceName, sig)

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	defer cancel()

	// 依次执行关闭回调
	for i, callback := range g.callbacks {
		log.Printf("[%s] 执行关闭回调 %d/%d", g.serviceName, i+1, len(g.callbacks))
		if err := callback(ctx); err != nil {
			log.Printf("[%s] 关闭回调执行失败: %v", g.serviceName, err)
		}
	}

	log.Printf("[%s] 优雅关闭完成", g.serviceName)
}

// 典型的使用方式：
//
// shutdown := shutdown.New("api-service", 30*time.Second)
//
// // 注册HTTP服务器关闭
// shutdown.Register(func(ctx context.Context) error {
//     return server.Shutdown(ctx)
// })
//
// // 注册数据库关闭
// shutdown.Register(func(ctx context.Context) error {
//     sqlDB, _ := db.DB()
//     return sqlDB.Close()
// })
//
// // 等待关闭信号
// shutdown.Wait()
