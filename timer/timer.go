package timer

import (
	"log"
	"time"
)

type RefreshTimer struct {
	Interval time.Duration
	t        *time.Timer
	C        <-chan time.Time
}

func NewRefreshTimer(d time.Duration) *RefreshTimer {
	log.Println("starting refresh timer:", d)

	timer := time.NewTimer(d)

	return &RefreshTimer{
		Interval: d,
		t:        timer,
		C:        timer.C,
	}
}

func (rt *RefreshTimer) Next() {
	rt.Reset(rt.Interval)
}

func (rt *RefreshTimer) Reset(d time.Duration) {
	rt.t.Reset(d)
	log.Println("next refresh:", d)
}

func (rt *RefreshTimer) Stop() {
	if !rt.t.Stop() {
		log.Println("refresh timer already stopped or expired, draining channel")
		<-rt.t.C
		log.Println("refresh timer chanel drained")

		return
	}
}
