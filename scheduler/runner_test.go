package scheduler

import (
	"log"
	"testing"
	"time"
)

func TestRunner(t *testing.T) {
	d := func(dc dataChannel) error {
		for i := 0; i < 5; i++ {
			dc <- i
			log.Printf("Dispatcher sent:%v", i)
		}
		return nil
	}
	e := func(dc dataChannel) error {
	forloop:
		for {
			select {
			case d := <-dc:
				log.Printf("Executor received:%v", d)
			default:
				break forloop
			}
		}
		return nil
	}
	runner := NewRunner(5, false, d, e)
	go runner.Start()
	time.Sleep(50 * time.Microsecond)
}
