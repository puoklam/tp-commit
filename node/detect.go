package node

import (
	"time"
)

type Detector interface {
	Detect()
}

type TimeoutDetector struct {
	d time.Duration
}

func (d *TimeoutDetector) Detect() {
	// timer := time.AfterFunc(d.d, func() {
	// 	fmt.Println("timeout")
	// })
}

func NewTimeoutDetector(d time.Duration) *TimeoutDetector {
	return &TimeoutDetector{
		d: d,
	}
}
