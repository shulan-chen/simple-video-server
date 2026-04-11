package scheduler

import (
	"time"
	"video-server/api/utils"
	"video-server/internal/config"

	"go.uber.org/zap"
)

// Worker 定时任务执行器，周期性触发 Runner 执行任务
type Worker struct {
	runner    *Runner
	ticker    *time.Ticker
	done      chan struct{}
	interval  time.Duration
	startTime time.Time
}

// NewWorker 创建新的 Worker 实例
// interval: 任务执行间隔
// runner: 要执行的 Runner 实例
func NewWorker(interval time.Duration, runner *Runner) *Worker {
	return &Worker{
		runner:    runner,
		ticker:    time.NewTicker(interval),
		done:      make(chan struct{}),
		interval:  interval,
		startTime: time.Now(),
	}
}

// Start 启动定时任务（阻塞方法）
func (w *Worker) Start() {
	utils.Logger.Info("定时任务启动",
		zap.Duration("interval", w.interval),
		zap.Time("start_time", w.startTime))

	for {
		select {
		case <-w.ticker.C:
			// 每次定时触发时，在新协程中执行 Runner
			// 这样即使 Runner 执行时间超过 interval，也不会阻塞定时器
			go w.runner.Start()

		case <-w.done:
			utils.Logger.Info("定时任务停止",
				zap.Duration("运行时长", time.Since(w.startTime)))
			w.ticker.Stop()
			return
		}
	}
}

// Stop 停止定时任务
func (w *Worker) Stop() {
	close(w.done)
	w.runner.Stop()
}

// StartVideoClearWorker 启动视频清理定时任务（兼容旧的 Start 方法）
func StartVideoClearWorker() {
	// 从配置读取延迟时间
	interval := time.Duration(config.AppConfig.VideoDeleteDelayTime) * time.Second

	// 从配置读取批处理大小
	batchSize := config.AppConfig.VideoDeleteBatchSize
	if batchSize <= 0 {
		batchSize = 5 // 默认值
	}

	// 创建 Runner（数据缓冲区大小从配置读取，长期运行模式）
	runner := NewRunner(batchSize, true, VideoClearDispatcher, VideoClearExecutor)

	// 创建 Worker 并启动
	worker := NewWorker(interval, runner)
	worker.Start()
}
