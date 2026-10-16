package httpapi

import (
	"context"
	"net/http"
	"time"

	appversion "github.com/szplanner/snakerule/internal/version"
)

type systemInfoResponse struct {
	Version     string            `json:"version"`
	Environment string            `json:"environment"`
	Database    databaseStatusDTO `json:"database"`
}

type databaseStatusDTO struct {
	Backend   string `json:"backend"`
	Connected bool   `json:"connected"`
}

func (s *Server) getSystemInfo(w http.ResponseWriter, r *http.Request) {
	database := databaseStatusDTO{Backend: "PostgreSQL"}
	if s.database != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		database.Connected = s.database.PingContext(ctx) == nil
	}

	writeJSON(w, http.StatusOK, systemInfoResponse{
		Version:     appversion.Version,
		Environment: s.environment,
		Database:    database,
	})
}
