package api

import (
	"aws-ses-sender-go/config"
	"crypto/subtle"
	"net/http"
)

// apiKeyAuth API 키 인증 미들웨어
func apiKeyAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("x-api-key")
		expectedAPIKey := config.GetEnv("API_KEY", "")
		if expectedAPIKey == "" || subtle.ConstantTimeCompare([]byte(apiKey), []byte(expectedAPIKey)) != 1 {
			writeError(w, r, http.StatusUnauthorized, "Unauthorized: Invalid API key")
			return
		}
		next(w, r)
	}
}
