# aws-ses-sender-go

[한국어](README.md) | [English](README.en.md)

AWS SES 기반 Golang 이메일 발송 서비스

## 개요

`aws-ses-sender-go`는 AWS Simple Email Service(SES)를 활용하여 이메일을 효율적으로 발송하는 Golang 기반 서비스입니다. 이메일 발송 요청 처리, 상태 추적, 예약 발송 및 결과 분석 기능을 제공합니다.

## 주요 기능

- **AWS SES 연동 이메일 발송**: AWS SES를 통한 안정적인 대량 이메일 발송
- **이메일 스케줄링**: 특정 시간에 이메일 발송 예약 기능
- **안정적인 발송**: AWS SES의 초당 발송 수를 고려하여 안정적으로 균등하게 발송
- **발송 상태 관리**: 이메일 생성, 처리, 발송, 실패, 중단 등 상태 관리
- **결과 추적 및 분석**:
  - 이메일 오픈 이벤트 추적 (1x1 픽셀 이미지 활용)
  - AWS SES 발송 결과(전달, 실패, 바운스) 저장 및 분석
  - 토픽별/시간별 발송 결과 및 통계 조회 API
- **Sentry 연동**: 에러 모니터링 지원

## 아키텍처

### 동작 흐름

1. **API 서버**가 이메일 발송 요청 수신
2. **스케줄러**가 DB에서 발송 대기 요청 주기적 조회
3. **센더**가 AWS SES를 통해 이메일 발송
4. **AWS SNS**를 통해 발송 결과 수신 및 DB 저장
5. **오픈 트래킹**으로 이메일 열람 이벤트 기록
6. **API**를 통한 발송 결과 및 통계 조회

```mermaid
flowchart TD
    A[클라이언트] -->|이메일 발송 요청| B[API 서버]
    B -->|DB 저장| C[스케줄러]
    C -->|발송 대기 조회| D[센더]
    D -->|SES 발송| E[AWS SES]
    E -->|이메일 수신| F[수신자]
    E -->|SNS 콜백| G[API 서버]
    G -->|DB 저장| H[결과/통계]
    F -->|오픈 트래킹| I[API 서버]
    I -->|DB 저장| J[오픈 이벤트]
    H -->|통계/결과 조회| K[클라이언트]
    J -->|통계/결과 조회| K[클라이언트]
```

## 데이터베이스 모델

### Content 테이블
| 필드 | 타입 | 설명 |
|------|------|------|
| ID | uint (PK) | 내용 고유 식별자 |
| TopicId | string (index) | 이메일 주제 식별자 |
| Subject | string (not null) | 이메일 제목 |
| Content | text (not null) | 이메일 내용 |

### Request 테이블
| 필드 | 타입 | 설명 |
|------|------|------|
| ID | uint (PK) | 요청 고유 식별자 |
| TopicId | string (index) | 이메일 주제 식별자 |
| MessageId | string (index) | SES 메시지 식별자 |
| To | string (not null) | 수신자 이메일 |
| ContentId | uint (FK, index) | Content ID 참조 |
| ScheduledAt | timestamp (index) | 예약 발송 시간 |
| Status | smallint (not null) | 상태 코드 |
| Error | string | 오류 메시지 |
| CreatedAt | timestamp | 생성 시간 |
| UpdatedAt | timestamp | 수정 시간 |
| DeletedAt | timestamp | 삭제 시간 |

### Result 테이블
| 필드 | 타입 | 설명 |
|------|------|------|
| ID | uint (PK) | 결과 고유 식별자 |
| RequestId | uint (FK, index) | Request ID 참조 |
| Status | string (not null, index) | 발송 결과 상태 |
| Raw | json | 원시 결과 데이터 |
| CreatedAt | timestamp | 생성 시간 |
| UpdatedAt | timestamp | 수정 시간 |
| DeletedAt | timestamp | 삭제 시간 |

### 상태 코드 (Status)
- **0**: 생성 완료 (Created)
- **1**: 처리 중 (Processing)
- **2**: 발송 완료 (Sent)
- **3**: 실패 (Failed)
- **4**: 중단 (Stopped)

## 프로젝트 구조

```
aws-ses-sender-go/
├── main.go              # 애플리케이션 진입점
├── api/                 # HTTP API 관련 코드
│   ├── handler.go       # API 핸들러 함수
│   ├── route.go         # API 라우팅 설정
│   ├── server.go        # HTTP 서버 설정/실행
│   └── middlewares.go   # API 인증 미들웨어
├── cmd/                 # 백그라운드 작업 코드
│   ├── scheduler.go     # 발송 대기 이메일 스케줄러
│   └── sender.go        # SES 이메일 발송 처리
├── config/              # 애플리케이션 설정
│   ├── env.go           # 환경 변수 관리
│   └── db.go            # 데이터베이스 연결 설정
├── model/               # 데이터베이스 모델
│   └── email.go         # GORM 모델 정의
└── pkg/aws/             # AWS 서비스 연동
    └── ses.go           # SES 이메일 발송
```

## 시작하기

### 사전 준비 사항

- Go 언어 개발 환경
- AWS 계정 및 SES 서비스 설정
  - 발신자 이메일/도메인 인증
  - IAM 사용자 생성 및 SES 권한 부여
- PostgreSQL 데이터베이스
- (선택) Sentry DSN

### 설정

`.env` 파일을 프로젝트 루트에 생성하여 다음 환경 변수를 설정하세요:

```env
# AWS 관련
AWS_ACCESS_KEY_ID=your_access_key
AWS_SECRET_ACCESS_KEY=your_secret_key
AWS_REGION=ap-northeast-2
EMAIL_SENDER=sender@example.com

# 서버 및 API
SERVER_PORT=3000
API_KEY=your_api_key
SERVER_HOST=http://localhost:3000

# 데이터베이스 (PostgreSQL)
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=postgres

# 초당 발송 수
EMAIL_RATE=14

# Sentry (선택)
SENTRY_DSN=your_sentry_dsn
```

### 설치 및 실행

1. 저장소 복제:
   ```bash
   git clone <저장소_URL>
   cd aws-ses-sender-go
   ```

2. 의존성 설치:
   ```bash
   go mod tidy
   ```

3. 애플리케이션 실행:
   ```bash
   go run main.go
   ```

## API 엔드포인트

모든 요청에는 `x-api-key` 헤더가 필요합니다(일부 예외).

### 이메일 발송 요청
```
POST /v1/messages
```

요청 본문 예시:
```json
{
  "messages": [
    {
      "topicId": "promotion-event-2024",
      "emails": ["recipient1@example.com", "recipient2@example.com"],
      "subject": "특별 프로모션 안내",
      "content": "<h1>안녕하세요!</h1><p>특별 프로모션 내용을 확인하세요.</p>",
      "scheduledAt": "2024-12-25T10:00:00+09:00"
    }
  ]
}
```

### 토픽별 발송 통계 조회
```
GET /v1/topics/:topicId
```

### 이메일 오픈 추적
```
GET /v1/events/open?requestId={requestId}
```

### 발송 통계 조회
```
GET /v1/events/counts/sent?hours={hours}
```

### 발송 결과 수신 (AWS SNS)
```
POST /v1/events/results
```

## 기여하기

1. 저장소 포크
2. 기능 브랜치 생성 (`git checkout -b feature/기능명`)
3. 변경사항 커밋 (`git commit -m '기능 추가'`)
4. 브랜치에 푸시 (`git push origin feature/기능명`)
5. Pull Request 생성

## 라이센스

이 프로젝트는 MIT 라이센스 하에 배포됩니다. 자세한 내용은 [LICENSE](LICENSE) 파일을 참조하세요.