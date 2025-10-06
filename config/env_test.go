package config

import (
	"os"
	"testing"
)

// TestGetEnv 환경 변수 조회 함수 테스트
func TestGetEnv(t *testing.T) {
	tests := []struct {
		name        string   // 테스트 케이스 이름
		key         string   // 조회할 환경 변수 키
		defaults    []string // 기본값 (가변 인자)
		setupEnv    bool     // 환경 변수를 설정할지 여부
		envValue    string   // 설정할 환경 변수 값
		expectedVal string   // 예상 반환값
	}{
		{
			name:        "환경 변수가 설정되어 있고 기본값이 없는 경우",
			key:         "TEST_KEY_1",
			defaults:    nil,
			setupEnv:    true,
			envValue:    "test_value",
			expectedVal: "test_value",
		},
		{
			name:        "환경 변수가 없고 기본값이 제공된 경우",
			key:         "TEST_KEY_2",
			defaults:    []string{"default_value"},
			setupEnv:    false,
			envValue:    "",
			expectedVal: "default_value",
		},
		{
			name:        "환경 변수가 설정되어 있고 기본값도 제공된 경우 (환경 변수 우선)",
			key:         "TEST_KEY_3",
			defaults:    []string{"default_value"},
			setupEnv:    true,
			envValue:    "env_value",
			expectedVal: "env_value",
		},
		{
			name:        "환경 변수가 없고 기본값도 없는 경우 (빈 문자열 반환)",
			key:         "TEST_KEY_4",
			defaults:    nil,
			setupEnv:    false,
			envValue:    "",
			expectedVal: "",
		},
		{
			name:        "환경 변수가 빈 문자열이고 기본값이 제공된 경우 (기본값 반환)",
			key:         "TEST_KEY_5",
			defaults:    []string{"default_value"},
			setupEnv:    true,
			envValue:    "",
			expectedVal: "default_value",
		},
		{
			name:        "특수 문자가 포함된 환경 변수 값",
			key:         "TEST_KEY_6",
			defaults:    nil,
			setupEnv:    true,
			envValue:    "value-with-special@chars#123",
			expectedVal: "value-with-special@chars#123",
		},
		{
			name:        "공백이 포함된 환경 변수 값",
			key:         "TEST_KEY_7",
			defaults:    nil,
			setupEnv:    true,
			envValue:    "value with spaces",
			expectedVal: "value with spaces",
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

			// GetEnv 함수 호출
			result := GetEnv(tt.key, tt.defaults...)

			// 결과 검증
			if result != tt.expectedVal {
				t.Errorf("GetEnv() = %v, 예상 = %v", result, tt.expectedVal)
			}
		})
	}
}

// TestGetEnvMultipleCalls 동일 키에 대한 여러 번의 호출 테스트
func TestGetEnvMultipleCalls(t *testing.T) {
	testKey := "TEST_MULTIPLE_CALLS"
	testValue := "test_value"

	os.Setenv(testKey, testValue)
	defer os.Unsetenv(testKey)

	// 첫 번째 호출
	result1 := GetEnv(testKey)
	if result1 != testValue {
		t.Errorf("첫 번째 GetEnv() = %v, 예상 = %v", result1, testValue)
	}

	// 두 번째 호출 (같은 결과 반환되어야 함)
	result2 := GetEnv(testKey)
	if result2 != testValue {
		t.Errorf("두 번째 GetEnv() = %v, 예상 = %v", result2, testValue)
	}

	// 결과가 일관성 있는지 확인
	if result1 != result2 {
		t.Errorf("GetEnv 호출 결과가 일관성이 없음: %v != %v", result1, result2)
	}
}

// TestGetEnvDefaultValueOverride 기본값보다 환경 변수가 우선하는지 검증
func TestGetEnvDefaultValueOverride(t *testing.T) {
	testKey := "TEST_OVERRIDE"
	envValue := "from_env"
	defaultValue := "from_default"

	os.Setenv(testKey, envValue)
	defer os.Unsetenv(testKey)

	result := GetEnv(testKey, defaultValue)

	// 환경 변수가 기본값보다 우선해야 함
	if result != envValue {
		t.Errorf("환경 변수가 기본값보다 우선되지 않음: got %v, want %v", result, envValue)
	}
}

// TestGetEnvEmptyStringVsNil 빈 문자열과 nil 처리 차이 검증
func TestGetEnvEmptyStringVsNil(t *testing.T) {
	testKey := "TEST_EMPTY_STRING"
	defaultValue := "default"

	// 빈 문자열로 설정된 환경 변수
	os.Setenv(testKey, "")
	defer os.Unsetenv(testKey)

	result := GetEnv(testKey, defaultValue)

	// 빈 문자열인 경우 기본값 반환되어야 함
	if result != defaultValue {
		t.Errorf("빈 문자열 환경 변수에 대해 기본값이 반환되지 않음: got %v, want %v", result, defaultValue)
	}
}
