package httpapi

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/szplanner/snakerule/internal/domain"
	"github.com/szplanner/snakerule/internal/repo"
	"io"
	"net/http"
	"strings"
	"time"
)

func (s *Server) listRulesets(w http.ResponseWriter, r *http.Request) {
	items, err := s.rulesets.List(r.Context())
	if err != nil {
		writeError(w, 500, err)
		return
	}
	writeJSON(w, 200, paginatedListData(r, items, nil))
}
func (s *Server) getRuleset(w http.ResponseWriter, r *http.Request) {
	item, err := s.rulesets.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	writeJSON(w, 200, item)
}
func (s *Server) deleteRuleset(w http.ResponseWriter, r *http.Request) {
	if err := s.rulesets.Delete(r.Context(), r.PathValue("id")); err != nil {
		s.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (s *Server) saveRuleset(w http.ResponseWriter, r *http.Request) {
	var item domain.Ruleset
	if err := decodeBody(w, r, &item); err != nil {
		writeError(w, 400, err)
		return
	}
	if id := r.PathValue("id"); id != "" && item.ID != id {
		writeError(w, 400, errors.New("path id must match ruleset id"))
		return
	}
	channels, err := s.alertChannels.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	enabledChannels := make(map[string]bool, len(channels))
	for _, channel := range channels {
		enabledChannels[channel.ID] = channel.Enabled
	}
	validateAlert := func(alert *domain.RuleAlert, owner string) error {
		if alert == nil || !alert.Enabled {
			return nil
		}
		if len(alert.ChannelIDs) == 0 {
			return fmt.Errorf("%s 启用告警时至少选择一个渠道", owner)
		}
		for _, channelID := range alert.ChannelIDs {
			if !enabledChannels[channelID] {
				return fmt.Errorf("告警渠道 %q 不存在或未启用", channelID)
			}
		}
		return nil
	}
	if err := validateAlert(item.Alert, "规则集"); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	for _, rule := range item.Rules {
		if err := validateAlert(rule.Alert, fmt.Sprintf("规则 %q", rule.Name)); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	if err := s.rulesets.Save(r.Context(), item); err != nil {
		s.writeServiceError(w, err)
		return
	}
	w.Header().Set("Location", "/api/v1/rulesets/"+item.ID)
	writeJSON(w, 200, item)
}
func (s *Server) evaluate(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()
	recorder := &responseStatusRecorder{ResponseWriter: w, status: http.StatusOK}
	w = recorder
	var requestPayload bytes.Buffer
	r.Body = io.NopCloser(io.TeeReader(r.Body, &requestPayload))
	var rulesetID string
	var decision domain.Decision
	defer func() {
		if s.invocationLogs == nil {
			return
		}
		credential := invocationCredentialFrom(r.Context())
		source := "management"
		if credential.Type == "apikey" || credential.Type == "jwt" {
			source = "project_api"
		} else if strings.EqualFold(strings.TrimSpace(r.Header.Get("X-SnakeRule-Source")), "designer") {
			source = "designer_simulation"
		}
		hits := make([]domain.InvocationHitRule, 0, len(decision.HitRules))
		for _, hit := range decision.HitRules {
			actions := make([]string, 0, len(hit.Actions))
			for _, action := range hit.Actions {
				actions = append(actions, action.Type)
			}
			hits = append(hits, domain.InvocationHitRule{RuleID: hit.RuleID, Name: hit.Name, Priority: hit.Priority, Actions: actions})
		}
		_ = s.invocationLogs.Add(domain.InvocationLog{
			OrganizationID: repo.OrganizationIDFrom(r.Context()), ProjectID: repo.ProjectIDFrom(r.Context()),
			RulesetID: rulesetID, Method: r.Method, Path: r.URL.Path, Source: source, CredentialType: credential.Type, StatusCode: recorder.status,
			Matched:  recorder.status >= 200 && recorder.status < 300 && decision.Matched,
			HitRules: hits, RequestBody: captureInvocationBody(requestPayload.Bytes()), ResponseBody: captureInvocationBody(recorder.body),
			DurationMS: time.Since(startedAt).Milliseconds(), CreatedAt: startedAt.UTC(),
		})
	}()
	var req struct {
		RulesetID string          `json:"ruleset_id"`
		Facts     map[string]any  `json:"facts"`
		Ruleset   *domain.Ruleset `json:"ruleset,omitempty"`
	}
	if err := decodeBody(w, r, &req); err != nil {
		writeError(w, 400, err)
		return
	}
	rulesetID = strings.TrimSpace(req.RulesetID)
	if strings.TrimSpace(req.RulesetID) == "" {
		writeError(w, 400, errors.New("ruleset_id is required"))
		return
	}
	if req.Ruleset != nil && isProjectKeyRequest(r.Context()) {
		writeError(w, http.StatusForbidden, errors.New("项目 API Key 不能运行未保存的规则草稿"))
		return
	}
	var err error
	if req.Ruleset != nil {
		if req.Ruleset.ID != req.RulesetID {
			writeError(w, http.StatusBadRequest, errors.New("ruleset_id must match ruleset.id"))
			return
		}
		decision, err = s.engine.EvaluateDraft(*req.Ruleset, req.Facts)
	} else {
		decision, err = s.engine.Evaluate(r.Context(), req.RulesetID, req.Facts)
	}
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	if req.Ruleset == nil {
		s.dispatchProjectAlert(r.Context(), decision, nil, false)
	} else if strings.EqualFold(strings.TrimSpace(r.Header.Get("X-SnakeRule-Source")), "designer") {
		s.dispatchProjectAlert(r.Context(), decision, req.Ruleset, true)
	}
	writeJSON(w, 200, decision)
}
