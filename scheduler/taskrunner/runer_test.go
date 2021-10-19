package taskrunner

import (
	"log"
	"testing"
	"time"
)

func TestRunner(t *testing.T) {
	d := func(dc dataChannel) error {
		for i := 0; i < 30; i++ {
			dc <- i
			log.Printf("Dispacher sent: %v", i)
		}
		return nil
	}

	e := func(dc dataChannel) error {
	forloop:
		for {
			select {
			case d := <-dc:
				log.Printf("Excetor received: %v", d)
			default:
				break forloop
			}

		}
		return nil
	}

	runner := NewRunner(30, false, d, e)
	go runner.startAll()
	time.Sleep(3 * time.Second)
}
