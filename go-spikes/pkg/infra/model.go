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
	return time.After(plugin.GetInitialDelayDuration())
}

func StartRunTimer(ctx context.Context, plugin Plugin) <-chan time.Time {
	return time.After(plugin.GetRunDuration())
}

func StartIntervalTicker(ctx context.Context, plugin Plugin) *time.Ticker {
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
	return ticker
}
