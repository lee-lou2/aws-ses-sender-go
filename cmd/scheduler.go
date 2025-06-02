package cmd

import (
	"aws-ses-sender-go/config"
	"aws-ses-sender-go/model"
	"context"
	"log"
	"strconv"
	"time"
)

// RunScheduler runs the scheduler
// It schedules the email sending requests to be processed by the sender
func RunScheduler(ctx context.Context) {
	db := config.GetDB()
	sendPerSecStr := config.GetEnv("EMAIL_RATE", "14")
	sendPerSec, err := strconv.Atoi(sendPerSecStr)
	if err != nil {
		log.Fatalf("Invalid EMAIL_RATE: %v", err)
	}
	sendPerMin := sendPerSec * 60
	batchSize := 1000

	ticker := time.NewTicker(1 * time.Minute)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			contents := make(map[uint]*model.Content)
			for i := 0; i < sendPerMin; i += batchSize {
				now := time.Now().UTC()
				reqs := make([]*model.Request, 0, batchSize)
				err := db.Raw(`
				WITH locked_requests AS (
					SELECT id
					FROM email_requests
					WHERE status = ? AND (scheduled_at <= ? OR scheduled_at IS NULL) AND deleted_at IS NULL
					ORDER BY id ASC
					LIMIT ?
					FOR UPDATE SKIP LOCKED
				)
				UPDATE email_requests
				SET status = ?, updated_at = ?
				FROM locked_requests
				WHERE email_requests.id = locked_requests.id
				RETURNING email_requests.*;
			`,
					model.EmailMessageStatusCreated,
					now,
					batchSize,
					model.EmailMessageStatusProcessing,
					now,
				).Scan(&reqs).Error

				if err != nil {
					log.Printf("Update Returning Error: %v", err)
				} else if len(reqs) > 0 {
					for _, req := range reqs {
						if _, ok := contents[req.ContentId]; !ok {
							content := &model.Content{}
							db.First(content, req.ContentId)
							contents[req.ContentId] = content
						}
						req.Content = *contents[req.ContentId]
						reqChan <- req
					}
				} else {
					break
				}
			}
		}
	}
}
