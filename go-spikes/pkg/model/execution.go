package model

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/infra-bed/go-spikes/pkg/logger"
)

type ExecutionRepoManager interface {
	Add(job Job, cancelFunc context.CancelFunc) string
	List() []string
	Close(id string)
}

var ExecutionRepo ExecutionRepoManager = &executionRepo{
	runningJobs: make(map[string]*jobExecutionImpl),
	mutex:       sync.RWMutex{},
}

type executionRepo struct {
	runningJobs map[string]*jobExecutionImpl
	mutex       sync.RWMutex
}

func (e *executionRepo) Add(job Job, cancelFunc context.CancelFunc) string {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	jobExecution := newJobExecution(job, cancelFunc)
	e.runningJobs[jobExecution.id] = jobExecution
	return jobExecution.id
}

func (e *executionRepo) List() []string {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	var execIds []string
	for _, exec := range e.runningJobs {
		execIds = append(execIds, exec.id)
	}
	return execIds
}

func (e *executionRepo) Close(id string) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if _, exists := e.runningJobs[id]; exists {
		e.runningJobs[id].cancel()
		delete(e.runningJobs, id)
	} else {
		logger.Get().Warn().
			Str("id", id).
			Msg("JobExecution not found in the repository, cannot remove")
	}
}

func newJobExecution(job Job, cancelFunc context.CancelFunc) *jobExecutionImpl {
	return &jobExecutionImpl{
		id:        uuid.New().String(),
		startTime: time.Now(),
		jobName:   job.GetPlugin().GetName(),
		cancel:    cancelFunc,
	}
}

type jobExecutionImpl struct {
	id        string
	jobName   string
	startTime time.Time
	cancel    context.CancelFunc
}
