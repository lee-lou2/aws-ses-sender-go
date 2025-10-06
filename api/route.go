package api

import (
	"github.com/go-chi/chi/v5"
)

// setV1Routes API v1 라우트 설정
func setV1Routes(r chi.Router) {
	r.Route("/v1", func(r chi.Router) {
		r.Post("/messages", apiKeyAuth(createMessageHandler))
		r.Get("/topics/{topicId}", apiKeyAuth(getResultCntHandler))
		r.Get("/events/open", createOpenEventHandler)
		r.Get("/events/counts/sent", apiKeyAuth(getSentCntHandler))
		r.Post("/events/results", createResultEventHandler)
	})
}
