package bootstrap

import (
	"context"
	"log/slog"
	"time"

	"github.com/szplanner/snakerule/internal/repo"
)

func startInvocationLogCleanup(ctx context.Context, logs *repo.PersistentInvocationLogRepository, retentionDays, intervalSeconds, batchSize int, logger *slog.Logger) {
	if retentionDays < 1 {
		logger.Info("invocation log cleanup disabled", "retention_days", retentionDays)
		return
	}
	if intervalSeconds < 60 {
		intervalSeconds = 3600
	}
	if batchSize < 1 {
		batchSize = 500
	}
	go func() {
		run := func() {
			cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
			deletedTotal := 0
			var lastErr error
			for batch := 0; batch < 10; batch++ {
				deleted, err := logs.DeleteBefore(ctx, cutoff, batchSize)
				deletedTotal += deleted
				if err != nil {
					lastErr = err
					break
				}
				if deleted < batchSize {
					break
				}
			}
			logs.RecordCleanup(deletedTotal, lastErr)
			if lastErr != nil {
				logger.Error("invocation log retention cleanup failed", "deleted", deletedTotal, "error", lastErr)
			} else {
				logger.Info("invocation log retention cleanup completed", "deleted", deletedTotal, "retention_days", retentionDays)
			}
		}
		run()
		ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
}
