package scheduler

import (
	"video-server/api/utils"

	"go.uber.org/zap"
)

// Runner 任务执行器，协调 Dispatcher（任务分发）和 Executor（任务执行）
type Runner struct {
	controller controlChannel
	errorChan  controlChannel
	data       dataChannel
	dataSize   int
	longLived  bool
	dispatcher fn
	executor   fn
}

// NewRunner 创建新的 Runner 实例
// dataSize: 数据 channel 的缓冲区大小
// longLived: 是否长期运行（true 表示不自动关闭 channel）
// dispatcher: 任务分发函数（从数据源读取任务）
// executor: 任务执行函数（处理具体任务）
func NewRunner(dataSize int, longLived bool, dispatcher fn, executor fn) *Runner {
	return &Runner{
		controller: make(controlChannel, 1),
		errorChan:  make(controlChannel, 1),
		data:       make(dataChannel, dataSize),
		dataSize:   dataSize,
		longLived:  longLived,
		dispatcher: dispatcher,
		executor:   executor,
	}
}

// startDispatch 启动任务调度循环
func (r *Runner) startDispatch() {
	defer func() {
		if !r.longLived {
			close(r.data)
			close(r.controller)
			close(r.errorChan)
		}
	}()

	for {
		select {
		case cmd := <-r.controller:
			switch cmd {
			case READY_TO_DISPATCH:
				// 执行任务分发
				if err := r.dispatcher(r.data); err != nil {
					// 真正的错误（不包括"暂无任务"）
					if err.Error() == "暂无待删除视频" {
						utils.Logger.Info("暂无任务可分发，等待下次触发")
					} else {
						utils.Logger.Error("任务分发失败", zap.Error(err))
					}
					return
				}
				// 分发成功（包括暂无任务的情况），切换到执行阶段
				r.controller <- READY_TO_EXECUTE

			case READY_TO_EXECUTE:
				// 执行任务
				if err := r.executor(r.data); err != nil {
					utils.Logger.Error("任务执行失败", zap.Error(err))
					// 执行失败，结束本轮任务（等待下次定时触发）
					return
				}
				// 执行成功,ji继续下一轮分发
				r.controller <- READY_TO_DISPATCH
			}

		case signal := <-r.errorChan:
			if signal == CLOSE {
				utils.Logger.Info("Runner 收到关闭信号")
				return
			}
		}
	}
}

// Start 启动 Runner（启动任务调度循环）
func (r *Runner) Start() {
	r.controller <- READY_TO_DISPATCH
	r.startDispatch()
}

// Stop 停止 Runner（发送关闭信号）
func (r *Runner) Stop() {
	r.errorChan <- CLOSE
}
