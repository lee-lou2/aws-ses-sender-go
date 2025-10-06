package aws

import (
	"aws-ses-sender-go/config"
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

// SES AWS SES 클라이언트 래퍼
type SES struct {
	Client        *sesv2.Client
	senderEmail   string
	configSetName string
}

// NewSESClient SES 클라이언트 생성
func NewSESClient(ctx context.Context) (*SES, error) {
	region := config.GetEnv("AWS_REGION", "ap-northeast-2")
	accessKeyID := config.GetEnv("AWS_ACCESS_KEY_ID")
	secretAccessKey := config.GetEnv("AWS_SECRET_ACCESS_KEY")
	senderEmail := config.GetEnv("EMAIL_SENDER")

	if senderEmail == "" {
		return nil, fmt.Errorf("EMAIL_SENDER environment variable is required")
	}

	var cfgOpts []func(*awsConfig.LoadOptions) error
	cfgOpts = append(cfgOpts, awsConfig.WithRegion(region))

	if accessKeyID != "" && secretAccessKey != "" {
		cfgOpts = append(cfgOpts, awsConfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		))
	}

	cfg, err := awsConfig.LoadDefaultConfig(ctx, cfgOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	log.Printf("SES client initialized (region=%s, sender=%s)", region, senderEmail)

	return &SES{
		Client:        sesv2.NewFromConfig(cfg),
		senderEmail:   senderEmail,
		configSetName: config.GetEnv("SES_CONFIG_SET", ""),
	}, nil
}

// SendEmail AWS SES를 통한 이메일 발송
func (s *SES) SendEmail(ctx context.Context, reqID int, subject, body *string, receivers []string) (string, error) {
	if subject == nil || *subject == "" {
		return "", fmt.Errorf("subject cannot be empty")
	}
	if body == nil || *body == "" {
		return "", fmt.Errorf("body cannot be empty")
	}
	if len(receivers) == 0 {
		return "", fmt.Errorf("receivers list cannot be empty")
	}

	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(s.senderEmail),
		Destination: &types.Destination{
			ToAddresses: receivers,
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{
					Data:    aws.String(*subject),
					Charset: aws.String("UTF-8"),
				},
				Body: &types.Body{
					Html: &types.Content{
						Data:    aws.String(*body),
						Charset: aws.String("UTF-8"),
					},
				},
				Headers: []types.MessageHeader{
					{
						Name:  aws.String("X-Request-ID"),
						Value: aws.String(strconv.Itoa(reqID)),
					},
				},
			},
		},
	}

	if s.configSetName != "" {
		input.ConfigurationSetName = aws.String(s.configSetName)
	}

	result, err := s.Client.SendEmail(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to send email via SES: %w", err)
	}

	if result.MessageId == nil {
		return "", fmt.Errorf("SES returned nil message ID")
	}

	return *result.MessageId, nil
}
