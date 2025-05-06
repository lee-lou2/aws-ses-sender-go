package scheduler

import (
	"aws-ses-sender-go/cmd/sender"
	"aws-ses-sender-go/config"
	"aws-ses-sender-go/model"
	"gorm.io/gorm/clause"
	"log"
	"time"
)

func Run() {
	db := config.GetDB()
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		now := time.Now()
		reqs := make([]*model.Request, 0, 1000)
		err := db.Model(&model.Request{}).
			Clauses(clause.Returning{}).
			Where("status = ? AND scheduled_at <= ?", model.EmailMessageStatusCreated, now).
			Limit(1000).
			Updates(model.Request{Status: model.EmailMessageStatusProcessing}).
			Scan(&reqs).Error

		if err != nil {
			log.Printf("UPDATE RETURNING 오류: %v", err)
		} else if len(reqs) > 0 {
			for _, req := range reqs {
				sender.Request(*req, nil)
			}
		}
	}
}
