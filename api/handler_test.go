package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCreateOpenEventHandler 이메일 열람 추적 이벤트 핸들러 테스트
// DB 연결이 필요하므로 픽셀 이미지 반환 검증만 수행
func TestCreateOpenEventHandler(t *testing.T) {
	tests := []struct {
		name           string // 테스트 케이스 이름
		requestID      string // 쿼리 파라미터로 전달할 requestId
		expectedStatus int    // 예상 HTTP 상태 코드
		expectedType   string // 예상 Content-Type
	}{
		{
			name:           "requestId가 없는 경우에도 픽셀 반환",
			requestID:      "",
			expectedStatus: http.StatusOK,
			expectedType:   "image/png",
		},
		{
			name:           "잘못된 형식의 requestId도 픽셀 반환",
			requestID:      "invalid",
			expectedStatus: http.StatusOK,
			expectedType:   "image/png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// HTTP 요청 생성
			url := "/v1/events/open"
			if tt.requestID != "" {
				url += "?requestId=" + tt.requestID
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rr := httptest.NewRecorder()

			// 핸들러 실행 (DB 연결 실패해도 픽셀은 반환됨)
			createOpenEventHandler(rr, req)

			// 상태 코드 검증
			if rr.Code != tt.expectedStatus {
				t.Errorf("핸들러 상태 코드 = %v, 예상 = %v", rr.Code, tt.expectedStatus)
			}

			// Content-Type 검증
			contentType := rr.Header().Get("Content-Type")
			if contentType != tt.expectedType {
				t.Errorf("Content-Type = %v, 예상 = %v", contentType, tt.expectedType)
			}

			// 응답 본문이 비어있지 않은지 확인 (픽셀 이미지)
			if rr.Body.Len() == 0 {
				t.Error("응답 본문이 비어있음 (픽셀 이미지 예상)")
			}

			// Cache-Control 헤더 검증
			cacheControl := rr.Header().Get("Cache-Control")
			if cacheControl != "no-cache, no-store, must-revalidate" {
				t.Errorf("Cache-Control 헤더 = %v, 예상 = 'no-cache, no-store, must-revalidate'", cacheControl)
			}
		})
	}
}

// TestCreateResultEventHandler AWS SES 이벤트 결과 핸들러 테스트
func TestCreateResultEventHandler(t *testing.T) {
	tests := []struct {
		name           string            // 테스트 케이스 이름
		messageType    string            // x-amz-sns-message-type 헤더 값
		requestBody    map[string]string // 요청 본문
		expectedStatus int               // 예상 HTTP 상태 코드
		expectError    bool              // 에러 응답 예상 여부
	}{
		{
			name:        "x-amz-sns-message-type 헤더 없음",
			messageType: "",
			requestBody: map[string]string{
				"Type":    "Notification",
				"Message": "{}",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:        "잘못된 SNS 메시지 타입",
			messageType: "InvalidType",
			requestBody: map[string]string{
				"Type":    "InvalidType",
				"Message": "{}",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:        "SubscriptionConfirmation 메시지",
			messageType: "SubscriptionConfirmation",
			requestBody: map[string]string{
				"Type":         "SubscriptionConfirmation",
				"Message":      "{}",
				"MessageId":    "test-id",
				"SubscribeURL": "https://sns.amazonaws.com/confirm",
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "잘못된 JSON 형식의 요청 본문",
			messageType:    "Notification",
			requestBody:    nil, // invalid JSON
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 요청 본문 생성
			var bodyBytes []byte
			if tt.requestBody != nil {
				bodyBytes, _ = json.Marshal(tt.requestBody)
			} else {
				bodyBytes = []byte("invalid json")
			}

			// HTTP 요청 생성
			req := httptest.NewRequest(http.MethodPost, "/v1/events/result", bytes.NewReader(bodyBytes))
			if tt.messageType != "" {
				req.Header.Set("x-amz-sns-message-type", tt.messageType)
			}
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			// 핸들러 실행
			createResultEventHandler(rr, req)

			// 상태 코드 검증
			if rr.Code != tt.expectedStatus {
				t.Errorf("핸들러 상태 코드 = %v, 예상 = %v", rr.Code, tt.expectedStatus)
			}

			// 에러 응답 검증
			if tt.expectError {
				var response map[string]interface{}
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err == nil {
					if _, hasError := response["error"]; !hasError {
						t.Error("에러 응답이 예상되었지만 'error' 필드가 없음")
					}
				}
			}
		})
	}
}

// TestGetSentCntHandler 발송된 이메일 수 조회 핸들러 테스트
// DB 연결이 필요하므로 입력 검증만 테스트
func TestGetSentCntHandler(t *testing.T) {
	tests := []struct {
		name           string // 테스트 케이스 이름
		hoursParam     string // hours 쿼리 파라미터
		expectedStatus int    // 예상 HTTP 상태 코드
		expectError    bool   // 에러 응답 예상 여부
	}{
		{
			name:           "잘못된 hours 파라미터 (0)",
			hoursParam:     "0",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "잘못된 hours 파라미터 (음수)",
			hoursParam:     "-5",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "잘못된 hours 파라미터 (범위 초과)",
			hoursParam:     "200",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "잘못된 hours 파라미터 (문자열)",
			hoursParam:     "invalid",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// HTTP 요청 생성
			url := "/v1/stats/sent"
			if tt.hoursParam != "" {
				url += "?hours=" + tt.hoursParam
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rr := httptest.NewRecorder()

			// 핸들러 실행
			getSentCntHandler(rr, req)

			// 상태 코드 검증
			if rr.Code != tt.expectedStatus {
				t.Errorf("핸들러 상태 코드 = %v, 예상 = %v", rr.Code, tt.expectedStatus)
			}

			// 에러 응답 검증
			if tt.expectError {
				var response map[string]interface{}
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err == nil {
					if _, hasError := response["error"]; !hasError {
						t.Error("에러 응답이 예상되었지만 'error' 필드가 없음")
					}
				}
			}
		})
	}
}

// TestWriteJSON writeJSON 헬퍼 함수 테스트
func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name           string      // 테스트 케이스 이름
		status         int         // HTTP 상태 코드
		data           interface{} // 응답 데이터
		expectedStatus int         // 예상 상태 코드
		expectedType   string      // 예상 Content-Type
	}{
		{
			name:           "정상적인 JSON 응답",
			status:         http.StatusOK,
			data:           map[string]string{"message": "success"},
			expectedStatus: http.StatusOK,
			expectedType:   "application/json",
		},
		{
			name:           "에러 응답",
			status:         http.StatusBadRequest,
			data:           map[string]string{"error": "bad request"},
			expectedStatus: http.StatusBadRequest,
			expectedType:   "application/json",
		},
		{
			name:           "빈 객체 응답",
			status:         http.StatusOK,
			data:           map[string]interface{}{},
			expectedStatus: http.StatusOK,
			expectedType:   "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			writeJSON(rr, tt.status, tt.data)

			// 상태 코드 검증
			if rr.Code != tt.expectedStatus {
				t.Errorf("상태 코드 = %v, 예상 = %v", rr.Code, tt.expectedStatus)
			}

			// Content-Type 검증
			contentType := rr.Header().Get("Content-Type")
			if contentType != tt.expectedType {
				t.Errorf("Content-Type = %v, 예상 = %v", contentType, tt.expectedType)
			}

			// JSON 파싱 가능 여부 검증
			var result map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
				t.Errorf("JSON 파싱 실패: %v", err)
			}
		})
	}
}

// TestWriteError writeError 헬퍼 함수 테스트
func TestWriteError(t *testing.T) {
	tests := []struct {
		name           string // 테스트 케이스 이름
		status         int    // HTTP 상태 코드
		message        string // 에러 메시지
		expectedStatus int    // 예상 상태 코드
	}{
		{
			name:           "400 Bad Request 에러",
			status:         http.StatusBadRequest,
			message:        "invalid input",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "401 Unauthorized 에러",
			status:         http.StatusUnauthorized,
			message:        "unauthorized access",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "500 Internal Server Error",
			status:         http.StatusInternalServerError,
			message:        "server error",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rr := httptest.NewRecorder()

			writeError(rr, req, tt.status, tt.message)

			// 상태 코드 검증
			if rr.Code != tt.expectedStatus {
				t.Errorf("상태 코드 = %v, 예상 = %v", rr.Code, tt.expectedStatus)
			}

			// 응답 본문 파싱
			var response map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatalf("JSON 파싱 실패: %v", err)
			}

			// error 필드 검증
			if errorMsg, ok := response["error"].(string); !ok || errorMsg != tt.message {
				t.Errorf("에러 메시지 = %v, 예상 = %v", errorMsg, tt.message)
			}

			// path 필드 존재 여부 검증
			if _, ok := response["path"]; !ok {
				t.Error("응답에 'path' 필드가 없음")
			}

			// method 필드 존재 여부 검증
			if _, ok := response["method"]; !ok {
				t.Error("응답에 'method' 필드가 없음")
			}

			// timestamp 필드 존재 여부 검증
			if _, ok := response["timestamp"]; !ok {
				t.Error("응답에 'timestamp' 필드가 없음")
			}
		})
	}
}

// TestCreateMessageHandlerValidation createMessageHandler 입력 검증 테스트
// 주의: handler 내부에서 config.GetDB()를 호출하므로 DB 연결 전 검증 케이스만 테스트 가능
func TestCreateMessageHandlerValidation(t *testing.T) {
	tests := []struct {
		name           string                 // 테스트 케이스 이름
		requestBody    map[string]interface{} // 요청 본문
		expectedStatus int                    // 예상 HTTP 상태 코드
		expectError    bool                   // 에러 응답 예상 여부
		errorContains  string                 // 에러 메시지에 포함되어야 할 문자열
	}{
		{
			name: "messages 배열이 빈 경우",
			requestBody: map[string]interface{}{
				"messages": []interface{}{},
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
			errorContains:  "messages array cannot be empty",
		},
		{
			name:           "잘못된 JSON 형식",
			requestBody:    nil, // 잘못된 JSON을 시뮬레이션
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
			errorContains:  "invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 요청 본문 생성
			var bodyBytes []byte
			if tt.requestBody != nil {
				bodyBytes, _ = json.Marshal(tt.requestBody)
			} else {
				bodyBytes = []byte("invalid json")
			}

			// HTTP 요청 생성
			req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			// 핸들러 실행
			createMessageHandler(rr, req)

			// 상태 코드 검증
			if rr.Code != tt.expectedStatus {
				t.Errorf("핸들러 상태 코드 = %v, 예상 = %v", rr.Code, tt.expectedStatus)
			}

			// 에러 응답 검증
			if tt.expectError {
				var response map[string]interface{}
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err == nil {
					if errorMsg, hasError := response["error"].(string); hasError {
						if tt.errorContains != "" {
							found := false
							for i := 0; i <= len(errorMsg)-len(tt.errorContains); i++ {
								if errorMsg[i:i+len(tt.errorContains)] == tt.errorContains {
									found = true
									break
								}
							}
							if !found {
								t.Errorf("에러 메시지에 예상 문자열이 없음. 응답 = %v, 예상 포함 = %v", errorMsg, tt.errorContains)
							}
						}
					} else {
						t.Error("에러 응답이 예상되었지만 'error' 필드가 없음")
					}
				}
			}
		})
	}
}

// TestGetResultCntHandlerValidation getResultCntHandler 입력 검증 테스트
// DB 연결이 필요하므로 입력 검증만 테스트
func TestGetResultCntHandlerValidation(t *testing.T) {
	tests := []struct {
		name           string // 테스트 케이스 이름
		topicId        string // URL 파라미터로 전달할 topicId
		expectedStatus int    // 예상 HTTP 상태 코드
		expectError    bool   // 에러 응답 예상 여부
	}{
		{
			name:           "topicId가 빈 문자열",
			topicId:        "",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// HTTP 요청 생성
			url := "/v1/topics/" + tt.topicId
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rr := httptest.NewRecorder()

			// 핸들러 실행 (chi 라우터 없이 직접 호출하므로 빈 topicId가 전달됨)
			getResultCntHandler(rr, req)

			// 상태 코드 검증
			if rr.Code != tt.expectedStatus {
				t.Errorf("핸들러 상태 코드 = %v, 예상 = %v", rr.Code, tt.expectedStatus)
			}

			// 에러 응답 검증
			if tt.expectError {
				var response map[string]interface{}
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err == nil {
					if _, hasError := response["error"]; !hasError {
						t.Error("에러 응답이 예상되었지만 'error' 필드가 없음")
					}
				}
			}
		})
	}
}
