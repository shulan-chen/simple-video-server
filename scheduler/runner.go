package scheduler

import (
	"video-server/api/utils"

	"go.uber.org/zap"
)

type Runner struct {
	Controller controlChannel
	Error      controlChannel
	Data       dataChannel
	dataSize   int
	longLived  bool
	Dispatcher fn
	Executor   fn
}

func NewRunner(dataSize int, longLived bool, d fn, e fn) *Runner {
	return &Runner{
		Controller: make(controlChannel, 1),
		Error:      make(controlChannel, 1),
		Data:       make(dataChannel, dataSize),
		dataSize:   dataSize,
		longLived:  longLived,
		Dispatcher: d,
		Executor:   e,
	}
}

func (r *Runner) startDispatch() {
	defer func() {
		if !r.longLived {
			close(r.Data)
			close(r.Controller)
			close(r.Error)
		}
	}()

	for {
		select {
		case c := <-r.Controller:
			switch c {
			case READY_TO_DISPATCH:
				err := r.Dispatcher(r.Data)
				if err != nil {
					utils.Logger.Error("Dispatcher执行失败", zap.Error(err))
					return
				}
				r.Controller <- READY_TO_EXECUTE
			case READY_TO_EXECUTE:
				err := r.Executor(r.Data)
				if err != nil {
					utils.Logger.Error("Executor执行失败", zap.Error(err))
					return
				}
				r.Controller <- READY_TO_DISPATCH
			}
		case e := <-r.Error:
			if e == CLOSE {
				utils.Logger.Info("Runner收到关闭信号")
				return
			}
		}
	}
}

func (r *Runner) Start() {
	r.Controller <- READY_TO_DISPATCH
	r.startDispatch()
}
