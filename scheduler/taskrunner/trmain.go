package taskrunner

import "time"

type Worker struct {
	ticker *time.Ticker
	runner *Runner
}

func NewWorker(inteval time.Duration, r *Runner) *Worker {
	return &Worker{
		ticker: time.NewTicker(inteval * time.Second),
		runner: r,
	}
}

func (w *Worker) startWorker() {
	for {
		select {
		case <-w.ticker.C:
			go w.runner.startAll()
		}
	}
}

func Start() {
	r := NewRunner(30, false, VideoClearDispatcher, VideoClearExecutor)
	w := NewWorker(30, r)
	w.startWorker()

}
