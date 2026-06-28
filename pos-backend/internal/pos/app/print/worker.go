package printqueue

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// WorkerConfig задает in-process print worker loop.
type WorkerConfig struct {
	WorkerID     string
	PollInterval time.Duration
	SendTimeout  time.Duration
}

// Worker выполняет due print_jobs последовательно в процессе POS Edge.
type Worker struct {
	service interface {
		ProcessNextPrintJob(context.Context, string) (bool, error)
	}
	config WorkerConfig
	logger *slog.Logger
}

func NewWorker(service interface {
	ProcessNextPrintJob(context.Context, string) (bool, error)
}, config WorkerConfig, logger *slog.Logger) *Worker {
	if strings.TrimSpace(config.WorkerID) == "" {
		config.WorkerID = fmt.Sprintf("pos-print-worker-%d", time.Now().UnixNano())
	}
	if config.PollInterval <= 0 {
		config.PollInterval = 2 * time.Second
	}
	if config.SendTimeout <= 0 {
		config.SendTimeout = 10 * time.Second
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{service: service, config: config, logger: logger}
}

func (w *Worker) Run(ctx context.Context) {
	w.logger.InfoContext(ctx, "print worker started", "operation", "print.worker", "worker_id", w.config.WorkerID)
	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			w.logger.InfoContext(ctx, "print worker stopped", "operation", "print.worker", "worker_id", w.config.WorkerID)
			return
		default:
		}
		if err := w.RunOnce(ctx); err != nil {
			w.logger.WarnContext(ctx, "print worker step failed", "operation", "print.worker", "worker_id", w.config.WorkerID, "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *Worker) RunOnce(ctx context.Context) error {
	sendCtx, cancel := context.WithTimeout(ctx, w.config.SendTimeout)
	defer cancel()
	_, err := w.service.ProcessNextPrintJob(sendCtx, w.config.WorkerID)
	return err
}
