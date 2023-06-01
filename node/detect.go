package node

import (
	"time"

	commit "github.com/puoklam/tp-commit"
)

type Detector interface {
	Detect(c commit.Interface) <-chan any
}

type Timeout interface {
	Duration() time.Duration
}

type TimeoutDetector struct{}

// Detect implements Detector
func (td *TimeoutDetector) Detect(c commit.Interface) <-chan any {
	ch := make(chan any, 1)
	t, ok := c.(Timeout)
	if !ok {
		return nil
	}
	time.AfterFunc(t.Duration(), func() {
		// not voted
		diff := c.Participants().Diff(c.Votes())
		ch <- diff
	})
	return ch
}
