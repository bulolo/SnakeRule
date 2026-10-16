package httpapi

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/szplanner/snakerule/internal/domain"
	"github.com/szplanner/snakerule/internal/licensing"
	"github.com/szplanner/snakerule/internal/repo"
	"github.com/szplanner/snakerule/internal/service"
	appversion "github.com/szplanner/snakerule/internal/version"
)

type Server struct {
	rulesets       *service.Rulesets
	engine         *service.Engine
	logger         *slog.Logger
	auth           AuthConfig
	sessions       *sessionStore
	license        *licensing.Manager
	workspaces     *service.Workspaces
	alertChannels  repo.AlertChannelRepository
	projects       repo.ProjectRepository
	projectKeys    repo.ProjectKeyRepository
	notifications  *repo.MemoryNotificationRepository
	invocationLogs *repo.MemoryInvocationLogRepository
	jwtSigningKey  []byte
}

func New(rulesets *service.Rulesets, engine *service.Engine, logger *slog.Logger, auth AuthConfig, license *licensing.Manager, workspaces *service.Workspaces) *Server {
	return &Server{rulesets: rulesets, engine: engine, logger: logger, auth: auth, sessions: newSessionStore(auth.SessionTTL, auth.SessionStorePath), license: license, workspaces: workspaces, jwtSigningKey: auth.JWTSigningKey}
}

func (s *Server) SetAlertChannelRepository(repository repo.AlertChannelRepository) {
	s.alertChannels = repository
}

func (s *Server) SetProjectRepositories(projects repo.ProjectRepository, keys repo.ProjectKeyRepository) {
	s.projects, s.projectKeys = projects, keys
}

func (s *Server) SetNotificationRepository(repository *repo.MemoryNotificationRepository) {
	s.notifications = repository
}

func (s *Server) SetInvocationLogRepository(repository *repo.MemoryInvocationLogRepository) {
	s.invocationLogs = repository
}

func (s *Server) listNotifications(w http.ResponseWriter, _ *http.Request) {
	if s.notifications == nil {
		writeJSON(w, http.StatusOK, map[string]any{"items": []any{}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": s.notifications.List()})
}
func (s *Server) readNotification(w http.ResponseWriter, r *http.Request) {
	if s.notifications == nil {
		writeError(w, http.StatusNotFound, errors.New("通知不存在"))
		return
	}
	if err := s.notifications.MarkRead(r.PathValue("id")); err != nil {
		s.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"read": true})
}
func (s *Server) readAllNotifications(w http.ResponseWriter, _ *http.Request) {
	if s.notifications != nil {
		if err := s.notifications.MarkAllRead(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]bool{"read": true})
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.index)
	mux.Handle("GET /assets/", s.assets())
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("GET /api/v1/meta", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"version": appversion.Version})
	})
	mux.HandleFunc("POST /api/v1/auth/login", s.login)
	mux.HandleFunc("GET /api/v1/auth/session", s.currentSession)
	mux.HandleFunc("POST /api/v1/auth/logout", s.logout)
	mux.Handle("GET /api/v1/platform/license", s.requireSession(http.HandlerFunc(s.getLicense)))
	mux.Handle("POST /api/v1/platform/license", s.requireSession(http.HandlerFunc(s.uploadLicense)))
	mux.Handle("DELETE /api/v1/platform/license", s.requireSession(http.HandlerFunc(s.clearLicense)))
	mux.Handle("GET /api/v1/workspaces", s.requireSession(http.HandlerFunc(s.listWorkspaces)))
	mux.Handle("POST /api/v1/workspaces", s.requireSession(http.HandlerFunc(s.createWorkspace)))
	mux.Handle("PUT /api/v1/workspaces/active", s.requireSession(http.HandlerFunc(s.setActiveWorkspace)))
	mux.Handle("PUT /api/v1/workspaces/{id}", s.requireSession(http.HandlerFunc(s.updateWorkspace)))
	mux.Handle("DELETE /api/v1/workspaces/{id}", s.requireSession(http.HandlerFunc(s.deleteWorkspace)))
	mux.Handle("GET /api/v1/system/alert-channels", s.requireSession(http.HandlerFunc(s.listAlertChannels)))
	mux.Handle("POST /api/v1/system/alert-channels", s.requireSession(http.HandlerFunc(s.saveAlertChannel)))
	mux.Handle("PUT /api/v1/system/alert-channels/{id}", s.requireSession(http.HandlerFunc(s.saveAlertChannel)))
	mux.Handle("DELETE /api/v1/system/alert-channels/{id}", s.requireSession(http.HandlerFunc(s.deleteAlertChannel)))
	mux.Handle("POST /api/v1/system/alert-channels/{id}/test", s.requireSession(http.HandlerFunc(s.testAlertChannel)))
	mux.Handle("GET /api/v1/notifications", s.requireSession(http.HandlerFunc(s.listNotifications)))
	mux.Handle("POST /api/v1/notifications/read-all", s.requireSession(http.HandlerFunc(s.readAllNotifications)))
	mux.Handle("POST /api/v1/notifications/{id}/read", s.requireSession(http.HandlerFunc(s.readNotification)))
	mux.Handle("GET /api/v1/projects", s.requireSession(http.HandlerFunc(s.listProjects)))
	mux.Handle("POST /api/v1/projects", s.requireSession(http.HandlerFunc(s.createProject)))
	mux.Handle("PUT /api/v1/projects/active", s.requireSession(http.HandlerFunc(s.setActiveProject)))
	mux.Handle("PUT /api/v1/projects/{id}", s.requireSession(http.HandlerFunc(s.updateProject)))
	mux.Handle("DELETE /api/v1/projects/{id}", s.requireSession(http.HandlerFunc(s.deleteProject)))
	mux.Handle("GET /api/v1/projects/{id}/api-keys", s.requireSession(http.HandlerFunc(s.listProjectKeys)))
	mux.Handle("GET /api/v1/projects/{id}/invocations", s.requireSession(http.HandlerFunc(s.listProjectInvocations)))
	mux.Handle("POST /api/v1/projects/{id}/api-keys", s.requireSession(http.HandlerFunc(s.createProjectKey)))
	mux.Handle("DELETE /api/v1/projects/{id}/api-keys/{keyID}", s.requireSession(http.HandlerFunc(s.deleteProjectKey)))
	mux.Handle("GET /api/v1/rulesets", s.requireSession(http.HandlerFunc(s.listRulesets)))
	mux.Handle("POST /api/v1/rulesets", s.requireSession(http.HandlerFunc(s.saveRuleset)))
	mux.Handle("GET /api/v1/rulesets/{id}", s.requireSession(http.HandlerFunc(s.getRuleset)))
	mux.Handle("PUT /api/v1/rulesets/{id}", s.requireSession(http.HandlerFunc(s.saveRuleset)))
	mux.Handle("DELETE /api/v1/rulesets/{id}", s.requireSession(http.HandlerFunc(s.deleteRuleset)))
	mux.Handle("POST /api/v1/engine/evaluate", s.requireSessionOrProjectKey(http.HandlerFunc(s.evaluate)))
	return s.recover(s.accessLog(mux))
}

func (s *Server) listAlertChannels(w http.ResponseWriter, r *http.Request) {
	items, err := s.alertChannels.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) saveAlertChannel(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name    string `json:"name"`
		Type    string `json:"type"`
		URL     string `json:"url"`
		Secret  string `json:"secret"`
		Enabled bool   `json:"enabled"`
	}
	if err := decodeBody(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Type = strings.TrimSpace(input.Type)
	input.URL = strings.TrimSpace(input.URL)
	parsed, parseErr := url.ParseRequestURI(input.URL)
	if input.Name == "" || len([]rune(input.Name)) > 80 || !isAlertChannelType(input.Type) || parseErr != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" || parsed.User != nil {
		writeError(w, http.StatusBadRequest, errors.New("名称、有效的 HTTP(S) 地址和支持的渠道类型为必填项"))
		return
	}
	id := r.PathValue("id")
	if id == "internal-inbox" {
		writeError(w, http.StatusConflict, errors.New("站内通知为系统内置渠道，不能修改"))
		return
	}
	var item domain.AlertChannel
	if id != "" {
		var err error
		item, err = s.alertChannels.Get(r.Context(), id)
		if err != nil {
			s.writeServiceError(w, err)
			return
		}
	} else {
		bytes := make([]byte, 8)
		if _, err := rand.Read(bytes); err != nil {
			writeError(w, http.StatusInternalServerError, errors.New("无法生成渠道 ID"))
			return
		}
		id = "channel-" + hex.EncodeToString(bytes)
		item.ID = id
		item.CreatedAt = time.Now().UTC()
	}
	if input.Secret != "" {
		item.Secret = input.Secret
	}
	item.Name, item.Type, item.URL, item.Enabled, item.UpdatedAt = input.Name, input.Type, input.URL, input.Enabled, time.Now().UTC()
	if err := s.alertChannels.Save(r.Context(), item); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	item.HasSecret = item.Secret != ""
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) deleteAlertChannel(w http.ResponseWriter, r *http.Request) {
	if r.PathValue("id") == "internal-inbox" {
		writeError(w, http.StatusConflict, errors.New("站内通知为系统内置渠道，不能删除"))
		return
	}
	if err := s.alertChannels.Delete(r.Context(), r.PathValue("id")); err != nil {
		s.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) testAlertChannel(w http.ResponseWriter, r *http.Request) {
	item, err := s.alertChannels.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	message := "SnakeRule 告警渠道测试：这是一条来自系统设置的测试通知。"
	body, err := alertPayload(item.Type, message)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("无法生成测试通知"))
		return
	}
	targetURL := item.URL
	if item.Type == "dingtalk" && item.Secret != "" {
		timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		mac := hmac.New(sha256.New, []byte(item.Secret))
		_, _ = mac.Write([]byte(timestamp + "\n" + item.Secret))
		parsed, _ := url.Parse(targetURL)
		query := parsed.Query()
		query.Set("timestamp", timestamp)
		query.Set("sign", base64.StdEncoding.EncodeToString(mac.Sum(nil)))
		parsed.RawQuery = query.Encode()
		targetURL = parsed.String()
	}
	request, err := http.NewRequestWithContext(r.Context(), http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("Webhook 地址无效"))
		return
	}
	request.Header.Set("Content-Type", "application/json")
	if item.Type == "webhook" && item.Secret != "" {
		request.Header.Set("Authorization", "Bearer "+item.Secret)
	}
	if item.Type == "feishu" && item.Secret != "" {
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		mac := hmac.New(sha256.New, []byte(timestamp+"\n"+item.Secret))
		payload["timestamp"], payload["sign"] = timestamp, base64.StdEncoding.EncodeToString(mac.Sum(nil))
		body, _ = json.Marshal(payload)
		request.Body = io.NopCloser(bytes.NewReader(body))
		request.ContentLength = int64(len(body))
	}
	client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		writeError(w, http.StatusBadGateway, errors.New("测试通知发送失败，请检查地址和网络"))
		return
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		writeError(w, http.StatusBadGateway, errors.New("Webhook 返回非成功状态码"))
		return
	}
	responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	if readErr != nil {
		writeError(w, http.StatusBadGateway, errors.New("无法读取 Webhook 响应"))
		return
	}
	if item.Type != "webhook" && len(responseBody) > 0 {
		var result map[string]json.RawMessage
		if json.Unmarshal(responseBody, &result) == nil {
			for _, key := range []string{"errcode", "code", "StatusCode"} {
				if raw, ok := result[key]; ok {
					var code int
					if json.Unmarshal(raw, &code) == nil && code != 0 {
						writeError(w, http.StatusBadGateway, errors.New("消息平台拒绝了测试通知"))
						return
					}
				}
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func isAlertChannelType(value string) bool {
	switch value {
	case "webhook", "wecom", "dingtalk", "feishu":
		return true
	default:
		return false
	}
}

func alertPayload(channelType, message string) ([]byte, error) {
	var payload map[string]any
	switch channelType {
	case "feishu":
		payload = map[string]any{"msg_type": "text", "content": map[string]string{"text": message}}
	case "wecom":
		payload = map[string]any{"msgtype": "text", "text": map[string]string{"content": message}}
	case "dingtalk":
		payload = map[string]any{"msgtype": "text", "text": map[string]string{"content": message}}
	default:
		payload = map[string]any{"type": "snakerule.alert.test", "title": "SnakeRule 告警渠道测试", "message": message, "sent_at": time.Now().UTC().Format(time.RFC3339)}
	}
	return json.Marshal(payload)
}

func (s *Server) listRulesets(w http.ResponseWriter, r *http.Request) {
	items, err := s.rulesets.List(r.Context())
	if err != nil {
		writeError(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]any{"items": items})
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
			WorkspaceID: repo.WorkspaceIDFrom(r.Context()), ProjectID: repo.ProjectIDFrom(r.Context()),
			RulesetID: rulesetID, Method: r.Method, Path: r.URL.Path, Source: source, CredentialType: credential.Type, StatusCode: recorder.status,
			Matched:  recorder.status >= 200 && recorder.status < 300 && decision.Matched,
			HitRules: hits, DurationMS: time.Since(startedAt).Milliseconds(), CreatedAt: startedAt.UTC(),
		})
	}()
	var req struct {
		RulesetID string          `json:"rulesetId"`
		Facts     map[string]any  `json:"facts"`
		Ruleset   *domain.Ruleset `json:"ruleset,omitempty"`
	}
	if err := decodeBody(w, r, &req); err != nil {
		writeError(w, 400, err)
		return
	}
	rulesetID = strings.TrimSpace(req.RulesetID)
	if strings.TrimSpace(req.RulesetID) == "" {
		writeError(w, 400, errors.New("rulesetId is required"))
		return
	}
	if req.Ruleset != nil && isProjectKeyRequest(r.Context()) {
		writeError(w, http.StatusForbidden, errors.New("项目 API Key 不能运行未保存的规则草稿"))
		return
	}
	var err error
	if req.Ruleset != nil {
		if req.Ruleset.ID != req.RulesetID {
			writeError(w, http.StatusBadRequest, errors.New("rulesetId must match ruleset.id"))
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

func (s *Server) listProjectInvocations(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if _, err := s.projects.Get(r.Context(), projectID); err != nil {
		s.writeServiceError(w, err)
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	isPager := 1
	if strings.TrimSpace(r.URL.Query().Get("is_pager")) == "0" {
		isPager = 0
	}
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	items := []domain.InvocationLog{}
	if s.invocationLogs != nil {
		items = s.invocationLogs.List(repo.WorkspaceIDFrom(r.Context()), projectID)
	}
	total := len(items)
	if isPager == 0 {
		size = total
		page = 1
	} else {
		start := (page - 1) * size
		if start >= total {
			items = []domain.InvocationLog{}
		} else {
			end := start + size
			if end > total {
				end = total
			}
			items = items[start:end]
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"list":       items,
		"pagination": map[string]int{"is_pager": isPager, "page": page, "size": size, "total": total},
	})
}

type responseStatusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *responseStatusRecorder) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseStatusRecorder) Write(data []byte) (int, error) {
	return w.ResponseWriter.Write(data)
}
func (s *Server) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repo.ErrNotFound):
		writeError(w, 404, err)
	case errors.Is(err, service.ErrInvalid):
		writeError(w, 400, err)
	default:
		writeError(w, 500, err)
	}
}

func decodeBody(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON value")
	}
	return nil
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"message": err.Error()}})
}
func (s *Server) recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				s.logger.Error("HTTP handler panic", "panic", recovered)
				writeError(w, 500, errors.New("internal server error"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
func (s *Server) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Info("http request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}
