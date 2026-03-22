package scheduler

import (
	"time"
	"video-server/api/utils"
	"video-server/internal/config"

	"go.uber.org/zap"
)

type Worker struct {
	Runner    *Runner
	ticker    *time.Ticker
	done      chan struct{} // 用于通知停止
	StartTime time.Time     // 启动时间
}

func NewWorker(interval time.Duration, r *Runner) *Worker {
	return &Worker{
		Runner:    r,
		ticker:    time.NewTicker(interval),
		done:      make(chan struct{}),
		StartTime: time.Now(),
	}
}

// StartWorker 启动worker（导出方法）
func (w *Worker) StartWorker() {
	for {
		select {
		case <-w.ticker.C:
			go w.Runner.Start()
		case <-w.done:
			utils.Logger.Info("Worker收到停止信号，正在停止",
				zap.String("service", "scheduler"))
			w.ticker.Stop()
			return
		}
	}
}

// Stop 停止worker
func (w *Worker) Stop() {
	close(w.done)
}

// 旧的Start方法保持兼容性（但不推荐使用）
func Start() {
	r := NewRunner(3, true, VideoClearDispatcher, VideoClearExecutor)
	w := NewWorker(time.Duration(config.AppConfig.VideoDeleteDelayTime)*time.Second, r)
	w.StartWorker()
}
