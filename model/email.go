package model

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

const (
	EmailMsgStatusCreated    = iota // 생성 완료
	EmailMsgStatusProcessing        // 처리 중
	EmailMsgStatusSent              // 발송 완료
	EmailMsgStatusFailed            // 발송 실패
	EmailMsgStatusStopped           // 중지됨
)

// Content 이메일 컨텐츠
type Content struct {
	gorm.Model
	Subject string `json:"subject" gorm:"not null;type:varchar(255);index:idx_subject"`
	Content string `json:"content" gorm:"not null;type:text"`
}

func (Content) TableName() string {
	return "email_contents"
}

// Request 이메일 발송 요청
type Request struct {
	gorm.Model
	TopicId     string     `json:"topic_id" gorm:"index:idx_topic_status;default:'';type:varchar(50)"`
	MessageId   string     `json:"message_id" gorm:"type:varchar(100);index:idx_message_id"`
	To          string     `json:"to" gorm:"not null;type:varchar(255);index:idx_recipient"`
	ContentId   uint       `json:"content_id" gorm:"index;not null"`
	Content     Content    `json:"content" gorm:"foreignKey:ContentId;references:ID"`
	ScheduledAt *time.Time `json:"scheduled_at" gorm:"not null;index:idx_scheduled_status;type:timestamp"`
	Status      int        `json:"status" gorm:"default:0;index:idx_topic_status,idx_scheduled_status;not null;type:smallint"`
	Error       string     `json:"error" gorm:"type:varchar(255)"`
}

func (Request) TableName() string {
	return "email_requests"
}

// Validate 요청 필드 검증
func (r *Request) Validate() error {
	if r.To == "" {
		return fmt.Errorf("recipient email is required")
	}
	if r.ContentId == 0 {
		return fmt.Errorf("content_id is required")
	}
	if r.ScheduledAt == nil {
		return fmt.Errorf("scheduled_at is required")
	}
	return nil
}

// Result AWS SES 이메일 발송 결과
type Result struct {
	gorm.Model
	RequestId uint    `json:"request_id" gorm:"index:idx_request_status;not null"`
	Request   Request `json:"request" gorm:"foreignKey:RequestId;references:ID"`
	Status    string  `json:"status" gorm:"not null;index:idx_request_status;type:varchar(50)"`
	Raw       string  `json:"raw" gorm:"type:json"`
}

func (Result) TableName() string {
	return "email_results"
}

// AutoMigrate 데이터베이스 마이그레이션 실행
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&Content{}); err != nil {
		return fmt.Errorf("failed to migrate Content: %w", err)
	}
	if !db.Migrator().HasTable(&Content{}) {
		return fmt.Errorf("email_contents table was not created")
	}

	if err := db.AutoMigrate(&Request{}); err != nil {
		return fmt.Errorf("failed to migrate Request: %w", err)
	}
	if !db.Migrator().HasTable(&Request{}) {
		return fmt.Errorf("email_requests table was not created")
	}

	if err := db.AutoMigrate(&Result{}); err != nil {
		return fmt.Errorf("failed to migrate Result: %w", err)
	}
	if !db.Migrator().HasTable(&Result{}) {
		return fmt.Errorf("email_results table was not created")
	}

	// WAL 체크포인트를 강제 실행하여 데이터를 메인 DB 파일에 기록
	if err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)").Error; err != nil {
		return fmt.Errorf("failed to execute WAL checkpoint: %w", err)
	}

	log.Println("Database migrations completed successfully")
	return nil
}
