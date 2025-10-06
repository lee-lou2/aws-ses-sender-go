package config

import (
	"os"
	"testing"
	"time"
)

// TestGetEnvAsInt 정수형 환경 변수 파싱 테스트
func TestGetEnvAsInt(t *testing.T) {
	tests := []struct {
		name        string // 테스트 케이스 이름
		key         string // 환경 변수 키
		envValue    string // 설정할 환경 변수 값
		defaultVal  int    // 기본값
		setupEnv    bool   // 환경 변수를 설정할지 여부
		expectedVal int    // 예상 반환값
	}{
		{
			name:        "정상적인 정수값 파싱",
			key:         "TEST_INT_1",
			envValue:    "42",
			defaultVal:  10,
			setupEnv:    true,
			expectedVal: 42,
		},
		{
			name:        "환경 변수가 없는 경우 기본값 반환",
			key:         "TEST_INT_2",
			envValue:    "",
			defaultVal:  100,
			setupEnv:    false,
			expectedVal: 100,
		},
		{
			name:        "잘못된 정수 형식 (문자열) - 기본값 반환",
			key:         "TEST_INT_3",
			envValue:    "not_a_number",
			defaultVal:  50,
			setupEnv:    true,
			expectedVal: 50,
		},
		{
			name:        "음수 정수값 파싱",
			key:         "TEST_INT_4",
			envValue:    "-15",
			defaultVal:  10,
			setupEnv:    true,
			expectedVal: -15,
		},
		{
			name:        "0 값 파싱",
			key:         "TEST_INT_5",
			envValue:    "0",
			defaultVal:  10,
			setupEnv:    true,
			expectedVal: 0,
		},
		{
			name:        "매우 큰 정수값 파싱",
			key:         "TEST_INT_6",
			envValue:    "1000000",
			defaultVal:  10,
			setupEnv:    true,
			expectedVal: 1000000,
		},
		{
			name:        "소수점이 포함된 값 (잘못된 형식) - 기본값 반환",
			key:         "TEST_INT_7",
			envValue:    "42.5",
			defaultVal:  20,
			setupEnv:    true,
			expectedVal: 20,
		},
		{
			name:        "공백이 포함된 정수 (잘못된 형식) - 기본값 반환",
			key:         "TEST_INT_8",
			envValue:    " 42 ",
			defaultVal:  30,
			setupEnv:    true,
			expectedVal: 30,
		},
		{
			name:        "빈 문자열 - 기본값 반환",
			key:         "TEST_INT_9",
			envValue:    "",
			defaultVal:  25,
			setupEnv:    true,
			expectedVal: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 테스트 전 환경 변수 정리
			os.Unsetenv(tt.key)

			// 필요한 경우 환경 변수 설정
			if tt.setupEnv {
				os.Setenv(tt.key, tt.envValue)
			}
			defer os.Unsetenv(tt.key)

			// GetEnvAsInt 함수 호출
			result := GetEnvAsInt(tt.key, tt.defaultVal)

			// 결과 검증
			if result != tt.expectedVal {
				t.Errorf("GetEnvAsInt() = %v, 예상 = %v", result, tt.expectedVal)
			}
		})
	}
}

// TestGetEnvAsDuration Duration 형식 환경 변수 파싱 테스트
func TestGetEnvAsDuration(t *testing.T) {
	tests := []struct {
		name        string        // 테스트 케이스 이름
		key         string        // 환경 변수 키
		envValue    string        // 설정할 환경 변수 값
		defaultVal  time.Duration // 기본값
		setupEnv    bool          // 환경 변수를 설정할지 여부
		expectedVal time.Duration // 예상 반환값
	}{
		{
			name:        "정상적인 초 단위 Duration 파싱",
			key:         "TEST_DUR_1",
			envValue:    "30s",
			defaultVal:  10 * time.Second,
			setupEnv:    true,
			expectedVal: 30 * time.Second,
		},
		{
			name:        "정상적인 분 단위 Duration 파싱",
			key:         "TEST_DUR_2",
			envValue:    "5m",
			defaultVal:  1 * time.Minute,
			setupEnv:    true,
			expectedVal: 5 * time.Minute,
		},
		{
			name:        "정상적인 시간 단위 Duration 파싱",
			key:         "TEST_DUR_3",
			envValue:    "2h",
			defaultVal:  1 * time.Hour,
			setupEnv:    true,
			expectedVal: 2 * time.Hour,
		},
		{
			name:        "환경 변수가 없는 경우 기본값 반환",
			key:         "TEST_DUR_4",
			envValue:    "",
			defaultVal:  15 * time.Second,
			setupEnv:    false,
			expectedVal: 15 * time.Second,
		},
		{
			name:        "잘못된 Duration 형식 - 기본값 반환",
			key:         "TEST_DUR_5",
			envValue:    "invalid",
			defaultVal:  20 * time.Second,
			setupEnv:    true,
			expectedVal: 20 * time.Second,
		},
		{
			name:        "밀리초 단위 Duration 파싱",
			key:         "TEST_DUR_6",
			envValue:    "500ms",
			defaultVal:  100 * time.Millisecond,
			setupEnv:    true,
			expectedVal: 500 * time.Millisecond,
		},
		{
			name:        "복합 단위 Duration 파싱",
			key:         "TEST_DUR_7",
			envValue:    "1h30m",
			defaultVal:  1 * time.Hour,
			setupEnv:    true,
			expectedVal: 90 * time.Minute,
		},
		{
			name:        "0 Duration 파싱",
			key:         "TEST_DUR_8",
			envValue:    "0s",
			defaultVal:  10 * time.Second,
			setupEnv:    true,
			expectedVal: 0,
		},
		{
			name:        "단위 없는 숫자 (잘못된 형식) - 기본값 반환",
			key:         "TEST_DUR_9",
			envValue:    "30",
			defaultVal:  5 * time.Second,
			setupEnv:    true,
			expectedVal: 5 * time.Second,
		},
		{
			name:        "빈 문자열 - 기본값 반환",
			key:         "TEST_DUR_10",
			envValue:    "",
			defaultVal:  25 * time.Second,
			setupEnv:    true,
			expectedVal: 25 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 테스트 전 환경 변수 정리
			os.Unsetenv(tt.key)

			// 필요한 경우 환경 변수 설정
			if tt.setupEnv {
				os.Setenv(tt.key, tt.envValue)
			}
			defer os.Unsetenv(tt.key)

			// getEnvAsDuration 함수 호출 (내부 함수이지만 GetEnvAsInt를 통해 간접 테스트)
			// 직접 테스트를 위해 함수 호출
			result := getEnvAsDuration(tt.key, tt.defaultVal)

			// 결과 검증
			if result != tt.expectedVal {
				t.Errorf("getEnvAsDuration() = %v, 예상 = %v", result, tt.expectedVal)
			}
		})
	}
}

// TestGetEnvAsIntBoundaryValues 경계값 테스트
func TestGetEnvAsIntBoundaryValues(t *testing.T) {
	tests := []struct {
		name        string // 테스트 케이스 이름
		envValue    string // 환경 변수 값
		expectedVal int    // 예상 반환값
		defaultVal  int    // 기본값
	}{
		{
			name:        "최대 int 값",
			envValue:    "2147483647",
			expectedVal: 2147483647,
			defaultVal:  0,
		},
		{
			name:        "최소 int 값",
			envValue:    "-2147483648",
			expectedVal: -2147483648,
			defaultVal:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_BOUNDARY"
			os.Setenv(key, tt.envValue)
			defer os.Unsetenv(key)

			result := GetEnvAsInt(key, tt.defaultVal)
			if result != tt.expectedVal {
				t.Errorf("GetEnvAsInt() = %v, 예상 = %v", result, tt.expectedVal)
			}
		})
	}
}

// TestGetEnvAsDurationNegative 음수 Duration 처리 테스트
func TestGetEnvAsDurationNegative(t *testing.T) {
	key := "TEST_NEG_DURATION"
	envValue := "-10s"
	defaultVal := 5 * time.Second

	os.Setenv(key, envValue)
	defer os.Unsetenv(key)

	result := getEnvAsDuration(key, defaultVal)

	// 음수 Duration도 파싱되어야 함
	expectedVal := -10 * time.Second
	if result != expectedVal {
		t.Errorf("getEnvAsDuration() = %v, 예상 = %v", result, expectedVal)
	}
}

// TestCloseDB CloseDB 함수 테스트
func TestCloseDB(t *testing.T) {
	// dbInstance가 nil인 경우 에러 없이 반환되어야 함
	dbInstance = nil
	err := CloseDB()
	if err != nil {
		t.Errorf("CloseDB()가 nil dbInstance에 대해 에러 반환: %v", err)
	}
}
