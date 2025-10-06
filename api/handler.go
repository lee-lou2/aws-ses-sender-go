package api

import (
	"aws-ses-sender-go/config"
	"aws-ses-sender-go/model"
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// createMessageHandler 이메일 발송 요청을 받아 처리
func createMessageHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var reqBody struct {
		Messages []struct {
			TopicId     string   `json:"topicId"`
			Emails      []string `json:"emails"`
			Subject     string   `json:"subject"`
			Content     string   `json:"content"`
			ScheduledAt string   `json:"scheduledAt"`
		} `json:"messages"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeError(w, r, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	if len(reqBody.Messages) == 0 {
		writeError(w, r, http.StatusBadRequest, "messages array cannot be empty")
		return
	}

	db := config.GetDB()
	var totalCreated int

	// 각 메시지를 트랜잭션으로 처리
	for _, msg := range reqBody.Messages {
		scheduledAt := time.Now().UTC()
		if msg.ScheduledAt != "" {
			t, err := time.Parse(time.RFC3339, msg.ScheduledAt)
			if err != nil {
				writeError(w, r, http.StatusBadRequest, fmt.Sprintf("invalid scheduledAt format: %v", err))
				return
			}
			scheduledAt = t.UTC()
			if scheduledAt.Before(time.Now().UTC()) {
				log.Printf("Warning: Scheduled time %v is in the past, skipping", scheduledAt)
				continue
			}
		}

		trimmedSubject := strings.TrimSpace(msg.Subject)
		if trimmedSubject == "" {
			writeError(w, r, http.StatusBadRequest, "subject cannot be empty")
			return
		}

		trimmedContent := strings.TrimSpace(msg.Content)
		if trimmedContent == "" {
			writeError(w, r, http.StatusBadRequest, "content cannot be empty")
			return
		}

		if len(msg.Emails) == 0 {
			writeError(w, r, http.StatusBadRequest, "emails array cannot be empty")
			return
		}

		validEmails := make([]string, 0, len(msg.Emails))
		for _, email := range msg.Emails {
			trimmedEmail := strings.TrimSpace(email)
			if _, err := mail.ParseAddress(trimmedEmail); err != nil {
				writeError(w, r, http.StatusBadRequest, fmt.Sprintf("invalid email address: %s", trimmedEmail))
				return
			}
			validEmails = append(validEmails, trimmedEmail)
		}

		err := db.Transaction(func(tx *gorm.DB) error {
			content := &model.Content{
				Subject: trimmedSubject,
				Content: trimmedContent,
			}
			if err := tx.Create(content).Error; err != nil {
				return fmt.Errorf("failed to create content: %w", err)
			}

			reqs := make([]*model.Request, 0, len(validEmails))
			for _, email := range validEmails {
				req := &model.Request{
					TopicId:     msg.TopicId,
					To:          email,
					ContentId:   content.ID,
					ScheduledAt: &scheduledAt,
					Status:      model.EmailMsgStatusCreated,
				}
				reqs = append(reqs, req)
			}

			const chunkSize = 1000
			for i := 0; i < len(reqs); i += chunkSize {
				end := i + chunkSize
				if end > len(reqs) {
					end = len(reqs)
				}
				batch := reqs[i:end]
				if err := tx.Create(&batch).Error; err != nil {
					return fmt.Errorf("failed to create requests batch: %w", err)
				}
				totalCreated += len(batch)
			}
			return nil
		})

		if err != nil {
			log.Printf("Transaction failed: %v", err)
			writeError(w, r, http.StatusInternalServerError, "failed to create email requests")
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"count":   totalCreated,
		"elapsed": time.Since(start).String(),
	})
}

// createOpenEventHandler 이메일 열람 추적 이벤트 처리
func createOpenEventHandler(w http.ResponseWriter, r *http.Request) {
	reqId := r.URL.Query().Get("requestId")

	defer func() {
		img := image.NewRGBA(image.Rect(0, 0, 1, 1))
		img.Set(0, 0, color.RGBA{R: 0, G: 0, B: 0, A: 0})
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			log.Printf("Failed to encode tracking pixel: %v", err)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Write(buf.Bytes())
	}()

	if reqId == "" {
		return
	}

	reqIdInt, err := strconv.Atoi(reqId)
	if err != nil {
		log.Printf("Invalid requestId format: %s, error: %v", reqId, err)
		return
	}

	db := config.GetDB()
	result := &model.Result{
		RequestId: uint(reqIdInt),
		Status:    "Open",
		Raw:       "{}",
	}

	if err := db.Create(result).Error; err != nil {
		log.Printf("Failed to create open event for requestId %d: %v", reqIdInt, err)
	}
}

// createResultEventHandler AWS SES 이벤트 결과 처리
func createResultEventHandler(w http.ResponseWriter, r *http.Request) {
	msgType := r.Header.Get("x-amz-sns-message-type")
	if msgType == "" {
		writeError(w, r, http.StatusBadRequest, "missing x-amz-sns-message-type header")
		return
	}

	if msgType != "Notification" && msgType != "SubscriptionConfirmation" {
		writeError(w, r, http.StatusBadRequest, "invalid SNS message type")
		return
	}

	var reqBody struct {
		Type         string `json:"Type"`
		Message      string `json:"Message"`
		MessageId    string `json:"MessageId"`
		SubscribeURL string `json:"SubscribeURL"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeError(w, r, http.StatusBadRequest, fmt.Sprintf("failed to parse SNS message: %v", err))
		return
	}

	if reqBody.Type == "SubscriptionConfirmation" {
		log.Printf("SNS Subscription confirmation required. URL: %s", reqBody.SubscribeURL)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": "subscription confirmation required",
			"url":     reqBody.SubscribeURL,
		})
		return
	}

	if reqBody.Type != "Notification" {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": fmt.Sprintf("unsupported message type: %s", reqBody.Type),
		})
		return
	}

	var sesNoti struct {
		NotiType string `json:"notificationType"`
		Mail     struct {
			MsgId   string `json:"messageId"`
			Headers []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"headers"`
		} `json:"mail"`
	}

	if err := json.Unmarshal([]byte(reqBody.Message), &sesNoti); err != nil {
		log.Printf("Failed to parse SES notification: %v", err)
		writeError(w, r, http.StatusBadRequest, "invalid SES notification format")
		return
	}

	if sesNoti.Mail.MsgId == "" {
		writeError(w, r, http.StatusBadRequest, "SES message_id not found")
		return
	}

	log.Printf("Received %s notification for message %s",
		sesNoti.NotiType, sesNoti.Mail.MsgId)

	var reqId uint
	for _, header := range sesNoti.Mail.Headers {
		if strings.EqualFold(header.Name, "X-Request-ID") {
			reqIdInt, err := strconv.Atoi(header.Value)
			if err != nil {
				log.Printf("Invalid X-Request-ID header value: %s, error: %v", header.Value, err)
				continue
			}
			reqId = uint(reqIdInt)
			break
		}
	}

	if reqId == 0 {
		writeError(w, r, http.StatusBadRequest, "X-Request-ID header not found or invalid")
		return
	}

	db := config.GetDB()
	result := &model.Result{
		RequestId: reqId,
		Status:    sesNoti.NotiType,
		Raw:       reqBody.Message,
	}

	if err := db.Create(result).Error; err != nil {
		log.Printf("Failed to save SES result event (requestId=%d): %v", reqId, err)
		writeError(w, r, http.StatusInternalServerError, "failed to save event")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"message": "ok"})
}

// getResultCntHandler 토픽별 이메일 발송 결과 집계 조회
func getResultCntHandler(w http.ResponseWriter, r *http.Request) {
	topicID := chi.URLParam(r, "topicId")
	if topicID == "" {
		writeError(w, r, http.StatusBadRequest, "topicId is required")
		return
	}

	db := config.GetDB()

	// 토픽ID에 해당하는 요청 존재 여부 확인
	var reqCnt int64
	if err := db.Model(&model.Request{}).Where("topic_id = ?", topicID).Count(&reqCnt).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if reqCnt == 0 {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"request": map[string]interface{}{"total": 0, "created": 0, "sent": 0, "failed": 0, "stopped": 0},
			"result":  map[string]interface{}{"total": 0, "statuses": map[string]int{}},
		})
		return
	}

	var reqResults []struct {
		Status int
		Count  int
	}
	if err := db.Model(&model.Request{}).
		Select("status, COUNT(*) as count").
		Where("topic_id = ?", topicID).
		Group("status").
		Scan(&reqResults).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	reqCnts := struct {
		Total   int `json:"total"`
		Created int `json:"created"`
		Sent    int `json:"sent"`
		Failed  int `json:"failed"`
		Stopped int `json:"stopped"`
	}{Total: int(reqCnt)}

	for _, r := range reqResults {
		switch r.Status {
		case model.EmailMsgStatusCreated:
			reqCnts.Created = r.Count
		case model.EmailMsgStatusSent:
			reqCnts.Sent = r.Count
		case model.EmailMsgStatusFailed:
			reqCnts.Failed = r.Count
		case model.EmailMsgStatusStopped:
			reqCnts.Stopped = r.Count
		}
	}

	subQuery := db.Model(&model.Request{}).Select("id").Where("topic_id = ?", topicID)

	var resultResults []struct {
		Status string
		Count  int
	}
	if err := db.Model(&model.Result{}).
		Select("status, COUNT(DISTINCT request_id) as count").
		Where("request_id IN (?)", subQuery).
		Group("status").
		Scan(&resultResults).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	resultCounts := make(map[string]int)
	for _, r := range resultResults {
		resultCounts[r.Status] = r.Count
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"request": reqCnts,
		"result": map[string]interface{}{
			"statuses": resultCounts,
		},
	})
}

// getSentCntHandler 지정된 시간 내 발송된 이메일 수 조회
func getSentCntHandler(w http.ResponseWriter, r *http.Request) {
	hoursStr := r.URL.Query().Get("hours")
	if hoursStr == "" {
		hoursStr = "24"
	}
	hours, err := strconv.Atoi(hoursStr)
	if err != nil || hours <= 0 || hours > 168 {
		writeError(w, r, http.StatusBadRequest, "hours must be between 1 and 168")
		return
	}

	startTime := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)

	db := config.GetDB()
	var cnt int64
	err = db.Model(&model.Request{}).
		Where("updated_at >= ?", startTime).
		Where("status = ?", model.EmailMsgStatusSent).
		Count(&cnt).Error

	if err != nil {
		log.Printf("Failed to count sent emails: %v", err)
		writeError(w, r, http.StatusInternalServerError, "failed to retrieve sent count")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"count": cnt,
		"hours": hours,
		"since": startTime.Format(time.RFC3339),
	})
}
