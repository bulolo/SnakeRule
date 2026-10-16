package httpapi

import (
	"encoding/json"
	"errors"
	"github.com/szplanner/snakerule/internal/domain"
	"github.com/szplanner/snakerule/internal/repo"
	"net/http"
	"strings"
	"time"
)

func (s *Server) listProjectInvocations(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if _, err := s.projects.Get(r.Context(), projectID); err != nil {
		s.writeServiceError(w, err)
		return
	}
	filter := repo.InvocationLogFilter{
		RulesetID: r.URL.Query().Get("ruleset_id"), RequestID: r.URL.Query().Get("request_id"),
		Source: r.URL.Query().Get("source"), Outcome: r.URL.Query().Get("outcome"),
		Page: positiveQueryInt(r, "page", 1), Size: positiveQueryInt(r, "size", 20),
		Unpaged: r.URL.Query().Get("is_pager") == "0",
	}
	var err error
	if raw := strings.TrimSpace(r.URL.Query().Get("since")); raw != "" {
		parsed, parseErr := time.Parse(time.RFC3339, raw)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, errors.New("since must be an RFC3339 timestamp"))
			return
		}
		filter.Since = &parsed
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("until")); raw != "" {
		parsed, parseErr := time.Parse(time.RFC3339, raw)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, errors.New("until must be an RFC3339 timestamp"))
			return
		}
		filter.Until = &parsed
	}
	if filter.Outcome != "" && filter.Outcome != "matched" && filter.Outcome != "unmatched" && filter.Outcome != "error" {
		writeError(w, http.StatusBadRequest, errors.New("outcome must be matched, unmatched, or error"))
		return
	}
	page := repo.InvocationLogPage{Items: []domain.InvocationLog{}}
	if s.invocationLogs != nil {
		page, err = s.invocationLogs.Query(r.Context(), repo.OrganizationIDFrom(r.Context()), projectID, filter)
		if err != nil {
			s.writeServiceError(w, err)
			return
		}
	}
	pagination := PaginationInfo{IsPager: 1, Page: filter.Page, Size: filter.Size, Total: page.Total}
	if filter.Unpaged {
		pagination.IsPager = 0
		pagination.Total = page.Total
	}
	writeJSON(w, http.StatusOK, map[string]any{"list": page.Items, "pagination": pagination})
}

type responseStatusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
	body        []byte
}

const invocationPayloadLimit = 8 * 1024

func (w *responseStatusRecorder) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseStatusRecorder) Write(data []byte) (int, error) {
	if remaining := invocationPayloadLimit + 1 - len(w.body); remaining > 0 {
		if remaining > len(data) {
			remaining = len(data)
		}
		w.body = append(w.body, data[:remaining]...)
	}
	return w.ResponseWriter.Write(data)
}

func captureInvocationBody(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var value any
	if json.Unmarshal(raw, &value) == nil {
		redactInvocationValue(value)
		if formatted, err := json.MarshalIndent(value, "", "  "); err == nil {
			raw = formatted
		}
	}
	if len(raw) <= invocationPayloadLimit {
		return string(raw)
	}
	preview, _ := json.Marshal(string(raw[:invocationPayloadLimit]))
	return "{\n  \"_truncated\": true,\n  \"_preview\": " + string(preview) + "\n}"
}

func redactInvocationValue(value any) {
	switch item := value.(type) {
	case map[string]any:
		for key, nested := range item {
			normalized := strings.NewReplacer("_", "", "-", "").Replace(strings.ToLower(key))
			if strings.Contains(normalized, "password") || strings.Contains(normalized, "passwd") || strings.Contains(normalized, "secret") || strings.Contains(normalized, "token") || strings.Contains(normalized, "authorization") || strings.Contains(normalized, "apikey") || strings.Contains(normalized, "privatekey") {
				item[key] = "[已隐藏]"
				continue
			}
			redactInvocationValue(nested)
		}
	case []any:
		for _, nested := range item {
			redactInvocationValue(nested)
		}
	}
}
