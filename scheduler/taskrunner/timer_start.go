package taskrunner

import "time"

type Worker struct {
	Runner *Runner
	ticker *time.Ticker
}

func NewWorker(interval time.Duration, r *Runner) *Worker {
	return &Worker{
		Runner: r,
		ticker: time.NewTicker(interval),
	}
}

func (w *Worker) startWorker() {
	for {
		select {
		case <-w.ticker.C:
			go w.Runner.Start()
		}
	}
}

func Start() {
	r := NewRunner(3, true, VideoClearDispatcher, VideoClearExecutor)
	w := NewWorker(30*time.Second, r)
	w.startWorker()
}
