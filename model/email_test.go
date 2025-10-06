package model

import (
	"testing"
	"time"
)

// TestRequestValidate Request 모델의 Validate 함수 테스트
func TestRequestValidate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string        // 테스트 케이스 이름
		request *Request      // 테스트할 Request 객체
		wantErr bool          // 에러 발생 예상 여부
		errMsg  string        // 예상되는 에러 메시지 (부분 문자열)
	}{
		{
			name: "정상적인 요청",
			request: &Request{
				To:          "test@example.com",
				ContentId:   1,
				ScheduledAt: &now,
				Status:      EmailMsgStatusCreated,
			},
			wantErr: false,
		},
		{
			name: "수신자 이메일이 빈 문자열",
			request: &Request{
				To:          "",
				ContentId:   1,
				ScheduledAt: &now,
			},
			wantErr: true,
			errMsg:  "recipient email is required",
		},
		{
			name: "ContentId가 0",
			request: &Request{
				To:          "test@example.com",
				ContentId:   0,
				ScheduledAt: &now,
			},
			wantErr: true,
			errMsg:  "content_id is required",
		},
		{
			name: "ScheduledAt이 nil",
			request: &Request{
				To:          "test@example.com",
				ContentId:   1,
				ScheduledAt: nil,
			},
			wantErr: true,
			errMsg:  "scheduled_at is required",
		},
		{
			name: "모든 필수 필드가 누락됨",
			request: &Request{
				To:          "",
				ContentId:   0,
				ScheduledAt: nil,
			},
			wantErr: true,
			errMsg:  "recipient email is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() 에러를 예상했지만 nil이 반환됨")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Validate() 에러 메시지 = %v, 예상 = %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() 에러가 예상되지 않았지만 발생함 = %v", err)
				}
			}
		})
	}
}

// TestEmailMsgStatus 이메일 메시지 상태 상수값 검증
func TestEmailMsgStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		expected int
	}{
		{"생성 완료 상태", EmailMsgStatusCreated, 0},
		{"처리 중 상태", EmailMsgStatusProcessing, 1},
		{"발송 완료 상태", EmailMsgStatusSent, 2},
		{"발송 실패 상태", EmailMsgStatusFailed, 3},
		{"중지됨 상태", EmailMsgStatusStopped, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.status != tt.expected {
				t.Errorf("%s = %d, 예상 = %d", tt.name, tt.status, tt.expected)
			}
		})
	}
}

// TestContentTableName Content 모델의 테이블명 검증
func TestContentTableName(t *testing.T) {
	c := Content{}
	expected := "email_contents"

	if c.TableName() != expected {
		t.Errorf("TableName() = %v, 예상 = %v", c.TableName(), expected)
	}
}

// TestRequestTableName Request 모델의 테이블명 검증
func TestRequestTableName(t *testing.T) {
	r := Request{}
	expected := "email_requests"

	if r.TableName() != expected {
		t.Errorf("TableName() = %v, 예상 = %v", r.TableName(), expected)
	}
}

// TestResultTableName Result 모델의 테이블명 검증
func TestResultTableName(t *testing.T) {
	r := Result{}
	expected := "email_results"

	if r.TableName() != expected {
		t.Errorf("TableName() = %v, 예상 = %v", r.TableName(), expected)
	}
}
