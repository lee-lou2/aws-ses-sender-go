package cmd

import (
	"aws-ses-sender-go/config"
	"aws-ses-sender-go/model"
	"aws-ses-sender-go/pkg/aws"
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

// reqChan 스케줄러와 워커 간 요청 전달 채널
var reqChan = make(chan *model.Request, 1000)

// RunSender 이메일 발송 워커 실행
func RunSender(ctx context.Context) {
	rateStr := config.GetEnv("EMAIL_RATE", "14")
	emailRate, err := strconv.Atoi(rateStr)
	if err != nil {
		log.Fatalf("Invalid EMAIL_RATE: %v", err)
	}

	maxConcurrent := config.GetEnvAsInt("MAX_CONCURRENT", emailRate*2)

	sesClient, err := aws.NewSESClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create SES client: %v", err)
	}

	db := config.GetDB()

	// 발송 속도 제어
	limiter := rate.NewLimiter(rate.Limit(emailRate), emailRate)
	// 동시 실행 수 제한
	sem := semaphore.NewWeighted(int64(maxConcurrent))

	log.Printf("Sender started (rate=%d/sec, max_concurrent=%d)", emailRate, maxConcurrent)

	var sentCnt, failCnt atomic.Int64
	now := time.Now()

	// 발송 메트릭 주기적 출력
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if sentCnt.Load() == 0 && failCnt.Load() == 0 {
					continue
				}

				elapsed := time.Since(now).Seconds()
				currentRate := float64(sentCnt.Load()) / elapsed
				log.Printf("Metrics: sent=%d, failed=%d, rate=%.2f/sec, uptime=%.0fs",
					sentCnt.Load(), failCnt.Load(), currentRate, elapsed)

				sentCnt.Store(0)
				failCnt.Store(0)
				now = time.Now()
			}
		}
	}()

	var wg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			log.Println("Sender shutting down, waiting for in-flight requests...")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := sem.Acquire(shutdownCtx, int64(maxConcurrent)); err == nil {
				sem.Release(int64(maxConcurrent))
			}
			cancel()
			wg.Wait()
			log.Printf("Sender stopped (sent=%d, failed=%d)", sentCnt.Load(), failCnt.Load())
			return

		case req := <-reqChan:
			if req == nil {
				continue
			}

			if err := limiter.Wait(ctx); err != nil {
				log.Printf("Rate limiter error: %v", err)
				continue
			}

			if err := sem.Acquire(ctx, 1); err != nil {
				log.Printf("Semaphore acquire error: %v", err)
				continue
			}

			wg.Add(1)
			go func(r *model.Request) {
				defer wg.Done()
				defer sem.Release(1)
				defer func() {
					if rec := recover(); rec != nil {
						log.Printf("Panic recovered in sendEmail: %v", rec)
						failCnt.Add(1)
					}
				}()

				if err := sendEmail(ctx, r, sesClient, db); err != nil {
					failCnt.Add(1)
				} else {
					sentCnt.Add(1)
				}
			}(req)
		}
	}
}

// sendEmail 이메일 발송 처리
func sendEmail(ctx context.Context, req *model.Request, sesClient *aws.SES, db *gorm.DB) error {
	serverHost := config.GetEnv("SERVER_HOST", "http://localhost:3000")

	if req.Content.ID == 0 {
		log.Printf("Content not loaded for RequestID=%d", req.ID)
		return fmt.Errorf("content not loaded")
	}

	content := req.Content.Content
	trackingPixel := fmt.Sprintf(`<img src="%s/v1/events/open?requestId=%d" width="1" height="1" alt="" />`,
		serverHost, req.ID)
	content += trackingPixel

	sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	msgId, err := sesClient.SendEmail(
		sendCtx,
		int(req.ID),
		&req.Content.Subject,
		&content,
		[]string{req.To},
	)

	status := model.EmailMsgStatusSent
	errMsg := ""
	if err != nil {
		status = model.EmailMsgStatusFailed
		errMsg = err.Error()
		log.Printf("Failed to send email (RequestID=%d, To=%s): %v", req.ID, req.To, err)
	}

	updateErr := db.WithContext(ctx).Model(&model.Request{}).
		Where("id = ?", req.ID).
		Updates(map[string]interface{}{
			"message_id": msgId,
			"status":     status,
			"error":      errMsg,
		}).Error

	if updateErr != nil {
		log.Printf("Failed to update request status (RequestID=%d): %v", req.ID, updateErr)
		return updateErr
	}

	return err
}
