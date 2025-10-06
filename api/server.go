package api

import (
	"aws-ses-sender-go/config"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	jsoniter "github.com/json-iterator/go"
)

func Run(ctx context.Context) {
	port := config.GetEnv("SERVER_PORT", "3000")

	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Timeout(30 * time.Second))

	if config.GetEnv("ENV", "dev") == "dev" {
		r.Mount("/debug", middleware.Profiler())
	}

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"service": "aws-ses-sender",
			"time":    time.Now().UTC().Format(time.RFC3339),
		})
	})

	setV1Routes(r)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	srvErr := make(chan error, 1)
	go func() {
		log.Printf("Starting HTTP server on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			srvErr <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("Shutdown signal received, stopping HTTP server...")
	case err := <-srvErr:
		log.Printf("Server error: %v", err)
		return
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	} else {
		log.Println("HTTP server stopped gracefully")
	}
}

// writeJSON JSON 응답 반환
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	jsoniter.NewEncoder(w).Encode(data)
}

// writeError 에러 응답 반환
func writeError(w http.ResponseWriter, r *http.Request, status int, message string) {
	log.Printf("Error: %s, Path: %s, Method: %s", message, r.URL.Path, r.Method)
	writeJSON(w, status, map[string]interface{}{
		"error":     message,
		"path":      r.URL.Path,
		"method":    r.Method,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
