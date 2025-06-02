package model

import (
	"aws-ses-sender-go/config"
	"time"

	"gorm.io/gorm"
)

const (
	EmailMessageStatusCreated    = iota // Creation complete
	EmailMessageStatusProcessing        // Processing
	EmailMessageStatusSent              // Sent
	EmailMessageStatusFailed            // Failed
	EmailMessageStatusStopped           // Stopped
)

type Content struct {
	gorm.Model
	Subject string `json:"subject" gorm:"not null;type:varchar(255)"`
	Content string `json:"content" gorm:"not null;type:text"`
}

func (m *Content) TableName() string {
	return "email_contents"
}

type Request struct {
	gorm.Model
	TopicId     string     `json:"topic_id" gorm:"index;default:'';type:varchar(50)"`
	MessageId   string     `json:"message_id" gorm:"null;type:varchar(100)"`
	To          string     `json:"to" gorm:"not null;type:varchar(255)"`
	ContentId   uint       `json:"content_id" gorm:"index;not null"`
	Content     Content    `json:"content" gorm:"foreignKey:ContentId;references:ID"`
	ScheduledAt *time.Time `json:"scheduled_at" gorm:"not null;index;type:timestamp"`
	Status      int        `json:"status" gorm:"default:0;index;not null;type:smallint"`
	Error       string     `json:"error" gorm:"null;type:varchar(255)"`
}

func (m *Request) TableName() string {
	return "email_requests"
}

type Result struct {
	gorm.Model
	RequestId uint    `json:"request_id" gorm:"index;not null"`
	Request   Request `json:"request" gorm:"foreignKey:RequestId;references:ID"`
	Status    string  `json:"status" gorm:"not null;index;type:varchar(50)"`
	Raw       string  `json:"raw" gorm:"type:json"`
}

func (m *Result) TableName() string {
	return "email_results"
}

func init() {
	db := config.GetDB()
	_ = db.AutoMigrate(&Content{})
	_ = db.AutoMigrate(&Request{})
	_ = db.AutoMigrate(&Result{})
}
