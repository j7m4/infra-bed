package model

import (
	"context"
	"time"

	"github.com/infra-bed/go-spikes/pkg/logger"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type Runner interface {
	Start(ctx context.Context, job Job)
}

func NewRunner() Runner {
	tracer := otel.Tracer("Runner")
	return &runnerImpl{
		tracer: tracer,
	}
}

type runnerImpl struct {
	tracer trace.Tracer
}

func (r *runnerImpl) Start(ctx context.Context, job Job) {
	if job.GetPlugin().GetRunDuration() <= 0 {
		logger.Ctx(ctx).Error().
			Str("job-name", job.GetPlugin().GetName()).
			Msg("runner has no run duration, skipping execution")
	}

	if job.GetPlugin().GetInitialDelayDuration() > 0 {
		select {
		case <-delayTimer(job.GetPlugin().GetInitialDelayDuration()):
			log.Trace().
				Str("job-name", job.GetPlugin().GetName()).
				Msg("initial-delay")
		}
		log.Trace().
			Str("job-name", job.GetPlugin().GetName()).
			Msg("post-initial-delay")
	}

	var span trace.Span
	var cancel context.CancelFunc

	ctx, cancel = context.WithTimeout(ctx, job.GetPlugin().GetRunDuration())
	ctx, span = r.tracer.Start(ctx, job.GetPlugin().GetName())
	execId := ExecutionRepo.Add(job, cancel)
	go func(ctx context.Context) {
		defer span.End()
		defer cancel()
		defer ExecutionRepo.Close(execId)
		job.Run(ctx)
	}(ctx)
}

func delayTimer(duration time.Duration) <-chan time.Time {
	var result <-chan time.Time
	if duration > 0 {
		result = time.After(duration)
	}
	return result
}
