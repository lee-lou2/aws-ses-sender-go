package cmd

import (
	"aws-ses-sender-go/config"
	"aws-ses-sender-go/model"
	"context"
	"log"
	"strconv"
	"time"
)

// RunScheduler 스케줄러 실행 (이메일 발송 요청을 처리 대기열에 추가)
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
			totalQueued := 0

			for i := 0; i < sendPerMin; i += batchSize {
				select {
				case <-ctx.Done():
					log.Printf("Scheduler interrupted, queued %d emails", totalQueued)
					return
				default:
				}

				now := time.Now().UTC()
				reqs := make([]*model.Request, 0, batchSize)

				// 상태 업데이트 및 처리 대상 조회 (SQLite3 RETURNING)
				err := db.Raw(`
					UPDATE email_requests
					SET status = ?, updated_at = ?
					WHERE id IN (
						SELECT id FROM email_requests
						WHERE status = ?
						  AND (scheduled_at <= ? OR scheduled_at IS NULL)
						  AND deleted_at IS NULL
						ORDER BY id ASC
						LIMIT ?
					)
					RETURNING *
				`,
					model.EmailMsgStatusProcessing,
					now,
					model.EmailMsgStatusCreated,
					now,
					batchSize,
				).Scan(&reqs).Error

				if err != nil {
					log.Printf("Update Returning Error: %v", err)
					break
				}

				if len(reqs) == 0 {
					break
				}

				for _, req := range reqs {
					if _, ok := contents[req.ContentId]; !ok {
						content := &model.Content{}
						db.First(content, req.ContentId)
						contents[req.ContentId] = content
					}
					req.Content = *contents[req.ContentId]

					select {
					case reqChan <- req:
						totalQueued++
					case <-ctx.Done():
						log.Printf("Scheduler interrupted while queueing, queued %d emails", totalQueued)
						return
					}
				}
			}

			if totalQueued > 0 {
				log.Printf("Queued %d emails for sending", totalQueued)
			}
		}
	}
}
