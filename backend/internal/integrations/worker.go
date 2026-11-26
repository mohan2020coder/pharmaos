package integrations

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type Worker struct {
	service  *Service
	interval time.Duration
	logger   *zap.Logger
}

func NewWorker(service *Service, interval time.Duration, logger *zap.Logger) *Worker {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	return &Worker{service: service, interval: interval, logger: logger}
}

func (w *Worker) Run(ctx context.Context) {
	if w.logger != nil {
		w.logger.Info("webhook dispatcher started", zap.Duration("interval", w.interval))
	}
	w.dispatch(ctx)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			if w.logger != nil {
				w.logger.Info("webhook dispatcher stopped")
			}
			return
		case <-ticker.C:
			w.dispatch(ctx)
		}
	}
}

func (w *Worker) dispatch(ctx context.Context) {
	claimed, err := w.service.DispatchBatch(ctx)
	if err != nil && w.logger != nil {
		w.logger.Warn("webhook dispatch batch failed", zap.Error(err))
		return
	}
	if claimed > 0 && w.logger != nil {
		w.logger.Debug("webhook deliveries claimed", zap.Int("count", claimed))
	}
}
