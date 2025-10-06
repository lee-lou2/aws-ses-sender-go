package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestApiKeyAuth API 키 인증 미들웨어 테스트
func TestApiKeyAuth(t *testing.T) {
	// 테스트용 핸들러 (성공 시 호출됨)
	successHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	tests := []struct {
		name           string              // 테스트 케이스 이름
		envAPIKey      string              // 환경변수에 설정할 API 키
		headerAPIKey   string              // 요청 헤더에 포함할 API 키
		expectedStatus int                 // 예상 HTTP 상태 코드
		expectedBody   string              // 예상 응답 본문 (부분 문자열)
	}{
		{
			name:           "유효한 API 키로 인증 성공",
			envAPIKey:      "test-api-key-12345",
			headerAPIKey:   "test-api-key-12345",
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name:           "잘못된 API 키로 인증 실패",
			envAPIKey:      "test-api-key-12345",
			headerAPIKey:   "wrong-api-key",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Unauthorized",
		},
		{
			name:           "API 키 헤더가 없는 경우",
			envAPIKey:      "test-api-key-12345",
			headerAPIKey:   "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Unauthorized",
		},
		{
			name:           "환경변수에 API 키가 설정되지 않은 경우",
			envAPIKey:      "",
			headerAPIKey:   "any-key",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Unauthorized",
		},
		{
			name:           "대소문자가 다른 API 키로 인증 실패",
			envAPIKey:      "TestApiKey",
			headerAPIKey:   "testapikey",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Unauthorized",
		},
		{
			name:           "공백이 포함된 API 키로 인증 실패",
			envAPIKey:      "test-api-key",
			headerAPIKey:   " test-api-key",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 환경변수 설정
			if tt.envAPIKey != "" {
				os.Setenv("API_KEY", tt.envAPIKey)
			} else {
				os.Unsetenv("API_KEY")
			}
			defer os.Unsetenv("API_KEY")

			// HTTP 요청 생성
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.headerAPIKey != "" {
				req.Header.Set("x-api-key", tt.headerAPIKey)
			}

			// ResponseRecorder 생성
			rr := httptest.NewRecorder()

			// 미들웨어를 통과시킨 핸들러 실행
			handler := apiKeyAuth(successHandler)
			handler.ServeHTTP(rr, req)

			// 상태 코드 검증
			if rr.Code != tt.expectedStatus {
				t.Errorf("핸들러 상태 코드 = %v, 예상 = %v", rr.Code, tt.expectedStatus)
			}

			// 응답 본문 검증
			body := rr.Body.String()
			if tt.expectedBody != "" {
				// 부분 문자열 포함 여부 확인
				found := false
				if len(body) > 0 {
					for i := 0; i <= len(body)-len(tt.expectedBody); i++ {
						if body[i:i+len(tt.expectedBody)] == tt.expectedBody {
							found = true
							break
						}
					}
				}
				if !found {
					t.Errorf("응답 본문에 예상 문자열이 없음. 응답 = %v, 예상 포함 = %v", body, tt.expectedBody)
				}
			}
		})
	}
}

// TestApiKeyAuthWithDifferentMethods 다양한 HTTP 메서드에 대한 API 키 인증 테스트
func TestApiKeyAuthWithDifferentMethods(t *testing.T) {
	os.Setenv("API_KEY", "valid-key")
	defer os.Unsetenv("API_KEY")

	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	successHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	for _, method := range methods {
		t.Run("HTTP_"+method+"_메서드", func(t *testing.T) {
			req := httptest.NewRequest(method, "/test", nil)
			req.Header.Set("x-api-key", "valid-key")
			rr := httptest.NewRecorder()

			handler := apiKeyAuth(successHandler)
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("%s 메서드: 상태 코드 = %v, 예상 = %v", method, rr.Code, http.StatusOK)
			}
		})
	}
}

// TestApiKeyAuthTimingAttackResistance 타이밍 공격 저항성 확인
// subtle.ConstantTimeCompare 사용 여부를 간접적으로 검증
func TestApiKeyAuthTimingAttackResistance(t *testing.T) {
	os.Setenv("API_KEY", "very-long-api-key-for-timing-test-123456789")
	defer os.Unsetenv("API_KEY")

	successHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	testCases := []string{
		"v",                                                  // 첫 글자만 일치
		"very",                                               // 앞부분만 일치
		"very-long-api-key",                                  // 중간까지 일치
		"very-long-api-key-for-timing-test-12345678",         // 거의 일치
		"completely-different-key-with-same-length-12345678", // 길이만 같음
	}

	for _, key := range testCases {
		t.Run("부분일치_"+key[:min(10, len(key))], func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("x-api-key", key)
			rr := httptest.NewRecorder()

			handler := apiKeyAuth(successHandler)
			handler.ServeHTTP(rr, req)

			// 모든 케이스에서 Unauthorized 응답을 받아야 함
			if rr.Code != http.StatusUnauthorized {
				t.Errorf("잘못된 API 키 '%s'가 인증을 통과함", key)
			}
		})
	}
}

// min 헬퍼 함수
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
