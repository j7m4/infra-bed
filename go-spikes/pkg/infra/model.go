package infra

import (
	"context"
	"time"
)

type Engine interface {
	Run(ctx context.Context)
	Close()
}

type Plugin interface {
	GetInitialDelayDuration() time.Duration
	GetRunDuration() time.Duration
	GetIntervalDuration() time.Duration
}

func StartInitialDelayTimer(ctx context.Context, plugin Plugin) <-chan time.Time {
	var result <-chan time.Time
	if plugin.GetInitialDelayDuration() > 0 {
		result = time.After(plugin.GetInitialDelayDuration())
	}
	return result
}

func StartRunTimer(ctx context.Context, plugin Plugin) <-chan time.Time {
	var result <-chan time.Time
	if plugin.GetRunDuration() > 0 {
		result = time.After(plugin.GetRunDuration())
	}
	return result
}

type IntervalTimer interface {
	NextTickWait()
}

func NewIntervalTimer(ctx context.Context, plugin Plugin) IntervalTimer {
	if plugin.GetIntervalDuration() <= 0 {
		return &intervalTimer{
			ticker: nil,
		}
	}
	tickerCtx, cancel := context.WithCancel(ctx)
	ticker := time.NewTicker(plugin.GetIntervalDuration())
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-tickerCtx.Done():
				cancel()
				return
			}
		}
	}()
	return &intervalTimer{
		ticker: ticker.C,
	}
}

type intervalTimer struct {
	ticker <-chan time.Time
}

func (i *intervalTimer) NextTickWait() {
	if i.ticker != nil {
		<-i.ticker
	}
}
