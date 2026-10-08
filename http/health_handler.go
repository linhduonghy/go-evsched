package http

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthHandler struct {
	db *pgxpool.Pool
}

func NewHealthHandle(db *pgxpool.Pool,
) *HealthHandler {
	return &HealthHandler{
		db: db,
	}
}

func (h HealthHandler) Live(writer http.ResponseWriter, request *http.Request) {
	_ = WriteJSON(writer, http.StatusOK, map[string]string{"status": "UP"})
}

func (h HealthHandler) Ready(writer http.ResponseWriter, request *http.Request) {

	ctx, cancel := context.WithTimeout(request.Context(), 5*time.Second)
	defer cancel()

	services := map[string]string{}
	ready := true

	// DB
	if err := h.db.Ping(ctx); err != nil {
		services["postgres"] = "DOWN"
		ready = false
	} else {
		services["postgres"] = "UP"
	}

	status := http.StatusOK
	state := "UP"

	if !ready {
		status = http.StatusServiceUnavailable
		state = "DOWN"
	}

	resp := map[string]any{
		"status":   state,
		"services": services,
	}

	_ = WriteJSON(writer, status, resp)

}
