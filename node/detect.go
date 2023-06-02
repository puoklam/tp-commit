package node

import (
	"sync"
	"time"

	commit "github.com/puoklam/tp-commit"
)

type Detector interface {
	Detect(c commit.Interface) <-chan any
}

type Timeout interface {
	Duration() time.Duration
}

type TimeoutDetector struct {
	mu     sync.Mutex
	timers []*time.Timer
}

// Detect implements Detector
func (td *TimeoutDetector) Detect(c commit.Interface) <-chan any {
	ch := make(chan any, 1)
	t, ok := c.(Timeout)
	if !ok {
		return nil
	}
	timer := time.AfterFunc(t.Duration(), func() {
		// not voted
		diff := c.Participants().Diff(c.Votes())
		ch <- diff
	})
	td.mu.Lock()
	defer td.mu.Unlock()
	td.timers = append(td.timers, timer)
	return ch
}

func (td *TimeoutDetector) Close() error {
	td.mu.Lock()
	defer td.mu.Unlock()
	for _, t := range td.timers {
		t.Stop()
	}
	return nil
}
