package httpapi

import (
	"encoding/json"
	"github.com/szplanner/snakerule/internal/domain"
	"github.com/szplanner/snakerule/internal/repo"
	"net/http"
	"strings"
)

func (s *Server) listProjectInvocations(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if _, err := s.projects.Get(r.Context(), projectID); err != nil {
		s.writeServiceError(w, err)
		return
	}
	items := []domain.InvocationLog{}
	if s.invocationLogs != nil {
		items = s.invocationLogs.List(repo.OrganizationIDFrom(r.Context()), projectID)
	}
	if rulesetID := strings.TrimSpace(r.URL.Query().Get("ruleset_id")); rulesetID != "" {
		filtered := make([]domain.InvocationLog, 0, len(items))
		for _, item := range items {
			if item.RulesetID == rulesetID {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	writeJSON(w, http.StatusOK, paginatedListData(r, items, nil))
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
