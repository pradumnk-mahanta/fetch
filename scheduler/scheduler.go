package scheduler

import (
	"context"
	"fetch/adapters"
	"sync"
	"time"
)

type Scheduler struct {
	cancel     context.CancelFunc
	jobRunning bool
	mu         sync.Mutex
}

func (s *Scheduler) Start(parentCtx context.Context) {
	ctx, cancel := context.WithCancel(parentCtx)
	s.cancel = cancel

	ticker := time.NewTicker(10 * time.Second)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.run()

			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *Scheduler) run() {
	s.mu.Lock()
	if s.jobRunning {
		s.mu.Unlock()
		return
	}
	s.jobRunning = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.jobRunning = false
		s.mu.Unlock()
	}()

	adapters.ProcessDownloadsQueue()
	adapters.ProcessDownloadItemsQueue()

	adapters.ProcessArchivedDownloadsQueue()
}

func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}
