package queue

import (
	"context"
	"log"
	"time"

	"github.com/its-asif/job-portal/internal/repository"
)

func StartOutboxRetryWorker(ctx context.Context, outboxRepo *repository.OutboxRepository, publisher *Client, interval time.Duration) {
	if outboxRepo == nil || publisher == nil {
		return
	}
	if interval <= 0 {
		interval = 10 * time.Second
	}

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				retryPendingEvents(ctx, outboxRepo, publisher)
			}
		}
	}()
}

func retryPendingEvents(ctx context.Context, outboxRepo *repository.OutboxRepository, publisher *Client) {
	events, err := outboxRepo.ListPending(50)
	if err != nil {
		log.Printf("outbox fetch failed: %v", err)
		return
	}

	for _, event := range events {
		if err := publisher.Publish(event.QueueName, string(event.Payload)); err != nil {
			retryDelay := time.Duration((event.Attempts+1)*5) * time.Second
			if retryDelay > 5*time.Minute {
				retryDelay = 5 * time.Minute
			}
			if markErr := outboxRepo.MarkRetry(event.ID, err.Error(), retryDelay); markErr != nil {
				log.Printf("outbox mark retry failed for event %d: %v", event.ID, markErr)
			}
			continue
		}

		if err := outboxRepo.MarkPublished(event.ID); err != nil {
			log.Printf("outbox mark published failed for event %d: %v", event.ID, err)
		}
	}

	_ = ctx
}
