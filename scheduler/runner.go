package scheduler

import "fmt"

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
				//fmt.Println("entered dispatch case")
				err := r.Dispatcher(r.Data)
				//fmt.Println("leave dispatch function")
				if err != nil {
					return
				}
				r.Controller <- READY_TO_EXECUTE
			case READY_TO_EXECUTE:
				//fmt.Println("entered execute case")
				err := r.Executor(r.Data)
				//fmt.Println("leave execute function")
				if err != nil {
					fmt.Println("execute function errored")
					return
				}
				r.Controller <- READY_TO_DISPATCH
			}
		case e := <-r.Error:
			if e == CLOSE {
				return
			}
		default:
		}
	}
}

func (r *Runner) Start() {
	r.Controller <- READY_TO_DISPATCH
	r.startDispatch()
}
