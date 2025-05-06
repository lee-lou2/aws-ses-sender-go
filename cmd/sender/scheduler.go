package sender

import (
	"aws-ses-sender-go/config"
	"aws-ses-sender-go/model"
	"log"
	"time"
)

func Run() {
	db := config.GetDB()
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		now := time.Now()
		reqs := make([]*model.Request, 0, 1000)
		sql := `
            UPDATE email_requests
            SET status = ?, updated_at = ?
            WHERE id IN (
                SELECT id
                FROM email_requests
                WHERE status = ? AND (scheduled_at <= ? OR scheduled_at IS NULL) AND deleted_at IS NULL
                ORDER BY id
                LIMIT 1000
                FOR UPDATE SKIP LOCKED
            )
            RETURNING *;
        `
		err := db.Raw(sql,
			model.EmailMessageStatusProcessing,
			now,
			model.EmailMessageStatusCreated,
			now,
		).Scan(&reqs).Error

		if err != nil {
			log.Printf("Update Returning Error: %v", err)
		} else if len(reqs) > 0 {
			for _, req := range reqs {
				reqChan <- req
			}
		}
	}
}
