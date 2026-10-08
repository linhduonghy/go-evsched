package http

import (
	"context"
	"evsched/logging"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const (
	ContextRequestID contextKey = "requestID"
	HeaderXRequestID            = "X-Request"
)

func GetRequestID(ctx context.Context) string {
	id, ok := ctx.Value(ContextRequestID).(string)
	if !ok {
		return ""
	}
	return id
}

func Context(baseLogger *slog.Logger) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			requestID := r.Header.Get(HeaderXRequestID)
			if requestID == "" {
				requestID = GetRequestID(r.Context())
			}
			if requestID == "" {
				requestID = uuid.NewString()
			}

			// add context requestID
			ctx := context.WithValue(r.Context(), ContextRequestID, requestID)
			reqLogger := baseLogger.With(logging.FieldRequestID, requestID)

			// bind logger to current context (Context-Aware Logging)
			ctx = logging.BindContext(ctx, reqLogger)

			w.Header().Set(HeaderXRequestID, requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
