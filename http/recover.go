package http

import (
	apperror "evsched/errors"
	"log/slog"
	"net/http"
	"runtime/debug"
)

func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error(
						"panic recovered",
						"panic", rec,
						"stack", string(debug.Stack()),
					)
					WriteErrorDetail(
						w,
						http.StatusInternalServerError,
						apperror.InternalServerErr.Error(),
					)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
