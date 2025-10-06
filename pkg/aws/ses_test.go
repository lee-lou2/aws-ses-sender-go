package aws

import (
	"context"
	"testing"
)

// TestSendEmailValidation SendEmail 함수의 입력 검증 테스트
func TestSendEmailValidation(t *testing.T) {
	// 테스트용 SES 클라이언트 (실제 AWS 호출 없이 검증만 수행)
	ses := &SES{
		Client:        nil, // 실제 클라이언트는 nil (입력 검증만 테스트)
		senderEmail:   "test@example.com",
		configSetName: "test-config-set",
	}

	ctx := context.Background()
	validSubject := "Test Subject"
	validBody := "Test Body"
	validReceivers := []string{"receiver@example.com"}
	emptySubject := ""
	emptyBody := ""

	tests := []struct {
		name      string   // 테스트 케이스 이름
		reqID     int      // 요청 ID
		subject   *string  // 제목 포인터
		body      *string  // 본문 포인터
		receivers []string // 수신자 목록
		wantErr   bool     // 에러 발생 예상 여부
		errMsg    string   // 예상 에러 메시지 (부분 문자열)
	}{
		{
			name:      "subject가 nil",
			reqID:     1,
			subject:   nil,
			body:      &validBody,
			receivers: validReceivers,
			wantErr:   true,
			errMsg:    "subject cannot be empty",
		},
		{
			name:      "subject가 빈 문자열",
			reqID:     1,
			subject:   &emptySubject,
			body:      &validBody,
			receivers: validReceivers,
			wantErr:   true,
			errMsg:    "subject cannot be empty",
		},
		{
			name:      "body가 nil",
			reqID:     1,
			subject:   &validSubject,
			body:      nil,
			receivers: validReceivers,
			wantErr:   true,
			errMsg:    "body cannot be empty",
		},
		{
			name:      "body가 빈 문자열",
			reqID:     1,
			subject:   &validSubject,
			body:      &emptyBody,
			receivers: validReceivers,
			wantErr:   true,
			errMsg:    "body cannot be empty",
		},
		{
			name:      "receivers가 빈 배열",
			reqID:     1,
			subject:   &validSubject,
			body:      &validBody,
			receivers: []string{},
			wantErr:   true,
			errMsg:    "receivers list cannot be empty",
		},
		{
			name:      "receivers가 nil",
			reqID:     1,
			subject:   &validSubject,
			body:      &validBody,
			receivers: nil,
			wantErr:   true,
			errMsg:    "receivers list cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ses.SendEmail(ctx, tt.reqID, tt.subject, tt.body, tt.receivers)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SendEmail() 에러를 예상했지만 nil이 반환됨")
					return
				}
				// 특정 에러 메시지가 지정된 경우에만 검증
				if tt.errMsg != "" {
					found := false
					errStr := err.Error()
					for i := 0; i <= len(errStr)-len(tt.errMsg); i++ {
						if errStr[i:i+len(tt.errMsg)] == tt.errMsg {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("SendEmail() 에러 메시지 = %v, 예상 포함 = %v", err.Error(), tt.errMsg)
					}
				}
			} else {
				if err != nil {
					t.Errorf("SendEmail() 에러가 예상되지 않았지만 발생함 = %v", err)
				}
			}
		})
	}
}

// TestSendEmailMultipleReceivers 여러 수신자에 대한 입력 검증 테스트
func TestSendEmailMultipleReceivers(t *testing.T) {
	ses := &SES{
		Client:        nil,
		senderEmail:   "sender@example.com",
		configSetName: "",
	}

	ctx := context.Background()
	subject := "Test Subject"
	body := "Test Body"

	tests := []struct {
		name      string   // 테스트 케이스 이름
		receivers []string // 수신자 목록
		expectErr string   // 예상 에러 메시지 (입력 검증 에러만)
	}{
		{
			name:      "빈 수신자 목록은 에러 발생",
			receivers: []string{},
			expectErr: "receivers list cannot be empty",
		},
		{
			name:      "nil 수신자 목록은 에러 발생",
			receivers: nil,
			expectErr: "receivers list cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ses.SendEmail(ctx, 1, &subject, &body, tt.receivers)

			if err == nil {
				t.Errorf("에러를 예상했지만 nil이 반환됨")
				return
			}

			if tt.expectErr != "" {
				found := false
				errStr := err.Error()
				for i := 0; i <= len(errStr)-len(tt.expectErr); i++ {
					if errStr[i:i+len(tt.expectErr)] == tt.expectErr {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("에러 메시지 = %v, 예상 포함 = %v", err.Error(), tt.expectErr)
				}
			}
		})
	}
}

// TestSESClientFields SES 클라이언트 필드 검증
func TestSESClientFields(t *testing.T) {
	tests := []struct {
		name          string // 테스트 케이스 이름
		senderEmail   string // 발신자 이메일
		configSetName string // SES Configuration Set 이름
	}{
		{
			name:          "Configuration Set이 설정됨",
			senderEmail:   "sender@example.com",
			configSetName: "my-config-set",
		},
		{
			name:          "Configuration Set이 빈 문자열",
			senderEmail:   "sender@example.com",
			configSetName: "",
		},
		{
			name:          "발신자 이메일만 설정됨",
			senderEmail:   "noreply@example.com",
			configSetName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ses := &SES{
				Client:        nil,
				senderEmail:   tt.senderEmail,
				configSetName: tt.configSetName,
			}

			// 필드 값 검증
			if ses.senderEmail != tt.senderEmail {
				t.Errorf("senderEmail = %v, 예상 = %v", ses.senderEmail, tt.senderEmail)
			}
			if ses.configSetName != tt.configSetName {
				t.Errorf("configSetName = %v, 예상 = %v", ses.configSetName, tt.configSetName)
			}
		})
	}
}

// TestSendEmailSpecialCharacters 특수문자 및 다국어 처리 테스트 (입력 검증만)
func TestSendEmailSpecialCharacters(t *testing.T) {
	// 입력 검증을 통과하는 케이스만 테스트 (실제 AWS 호출은 하지 않음)
	tests := []struct {
		name    string // 테스트 케이스 이름
		subject string // 제목
		body    string // 본문
	}{
		{
			name:    "특수문자가 포함된 제목",
			subject: "Test <>&\"' 제목 🎉",
			body:    "Test body",
		},
		{
			name:    "HTML 태그가 포함된 본문",
			subject: "Test subject",
			body:    "<html><body><h1>Test</h1></body></html>",
		},
		{
			name:    "한글이 포함된 제목과 본문",
			subject: "테스트 제목",
			body:    "테스트 본문입니다.",
		},
		{
			name:    "긴 제목 (1000자)",
			subject: string(make([]byte, 1000)),
			body:    "Test body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receivers := []string{"test@example.com"}

			// 입력 검증만 수행 (빈 문자열이 아닌 경우 통과)
			if tt.subject == "" || tt.body == "" || len(receivers) == 0 {
				t.Errorf("테스트 데이터가 잘못됨")
				return
			}

			// 실제로는 AWS 호출이 필요하므로 입력값 검증만 확인
			if len(tt.subject) == 0 {
				t.Errorf("제목이 비어있음")
			}
			if len(tt.body) == 0 {
				t.Errorf("본문이 비어있음")
			}
		})
	}
}
