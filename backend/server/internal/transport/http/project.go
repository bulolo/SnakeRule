package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/szplanner/snakerule/internal/domain"
	"github.com/szplanner/snakerule/internal/repo"
)

var projectSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	items, err := s.projects.List(r.Context())
	if err != nil {
		writeError(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]any{"items": items, "active_project_id": s.sessions.projectID(r)})
}

func (s *Server) createProject(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
		Template    string `json:"template"`
	}
	if err := decodeBody(w, r, &input); err != nil {
		writeError(w, 400, err)
		return
	}
	if input.Template != "" && input.Template != "ecommerce" {
		writeError(w, 400, errors.New("不支持的项目模板"))
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len([]rune(input.Name)) > 80 {
		writeError(w, 400, errors.New("项目名称必填且不能超过 80 个字符"))
		return
	}
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		writeError(w, 500, err)
		return
	}
	slug := strings.TrimSpace(strings.ToLower(input.Slug))
	if slug == "" {
		slug = "project-" + hex.EncodeToString(bytes[:4])
	}
	if !projectSlugPattern.MatchString(slug) {
		writeError(w, 400, errors.New("slug 仅支持小写字母、数字、连字符和下划线，长度不超过 32"))
		return
	}
	existing, _ := s.projects.List(r.Context())
	for _, item := range existing {
		if item.Slug == slug {
			writeError(w, 409, errors.New("项目 slug 已存在"))
			return
		}
	}
	now := time.Now().UTC()
	project := domain.Project{ID: "prj-" + hex.EncodeToString(bytes), Name: input.Name, Slug: slug, Description: strings.TrimSpace(input.Description), Icon: input.Icon, CreatedAt: now, UpdatedAt: now}
	if err := s.projects.Save(r.Context(), project); err != nil {
		writeError(w, 500, err)
		return
	}
	if input.Template == "ecommerce" {
		ctx := repo.WithProject(r.Context(), project.ID)
		if err := s.rulesets.Save(ctx, ecommerceDemoRuleset()); err != nil {
			_ = s.projects.Delete(r.Context(), project.ID)
			writeError(w, 500, err)
			return
		}
	}
	writeJSON(w, http.StatusCreated, project)
}

func ecommerceDemoRuleset() domain.Ruleset {
	return domain.Ruleset{ID: "order-risk", Name: "电商订单风控", Version: "1.0.0", Description: "演示订单风险识别、VIP 标记和人工审核策略。", Enabled: true, Rules: []domain.Rule{
		{ID: "blacklisted-customer", Name: "黑名单客户拦截", Priority: 300, Enabled: true, Conditions: domain.Condition{Fact: "customer.blacklisted", Operator: "equal", Value: true}, Actions: []domain.Action{{Type: "tag", Config: map[string]any{"label": "BLOCK_ORDER"}}}},
		{ID: "large-new-order", Name: "大额新客订单审核", Priority: 200, Enabled: true, Conditions: domain.Condition{All: []domain.Condition{{Fact: "order.amount", Operator: "greaterThan", Value: 10000}, {Fact: "customer.isNew", Operator: "equal", Value: true}}}, Actions: []domain.Action{{Type: "tag", Config: map[string]any{"label": "MANUAL_REVIEW"}}}},
		{ID: "vip-order", Name: "VIP 订单标记", Priority: 100, Enabled: true, Conditions: domain.Condition{Fact: "customer.level", Operator: "equal", Value: "VIP"}, Actions: []domain.Action{{Type: "tag", Config: map[string]any{"label": "VIP"}}}},
	}}
}

func (s *Server) updateProject(w http.ResponseWriter, r *http.Request) {
	item, err := s.projects.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	var input struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
	}
	if err := decodeBody(w, r, &input); err != nil {
		writeError(w, 400, err)
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Slug = strings.TrimSpace(strings.ToLower(input.Slug))
	if input.Name == "" || len([]rune(input.Name)) > 80 || input.Slug == "" || !projectSlugPattern.MatchString(input.Slug) {
		writeError(w, 400, errors.New("项目名称必填且不超过 80 个字符，slug 格式无效"))
		return
	}
	all, _ := s.projects.List(r.Context())
	for _, other := range all {
		if other.ID != item.ID && other.Slug == input.Slug {
			writeError(w, 409, errors.New("项目 slug 已存在"))
			return
		}
	}
	item.Name, item.Slug, item.Description, item.Icon, item.UpdatedAt = input.Name, input.Slug, strings.TrimSpace(input.Description), input.Icon, time.Now().UTC()
	if err := s.projects.Save(r.Context(), item); err != nil {
		writeError(w, 500, err)
		return
	}
	writeJSON(w, 200, item)
}

func (s *Server) deleteProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.projects.Get(r.Context(), id); err != nil {
		s.writeServiceError(w, err)
		return
	}
	if id == s.sessions.projectID(r) {
		if !s.sessions.setProject(r, "") {
			writeError(w, 401, errors.New("authentication required"))
			return
		}
	}
	ctx := repo.WithProject(r.Context(), id)
	sets, err := s.rulesets.List(ctx)
	if err != nil {
		writeError(w, 500, err)
		return
	}
	for _, set := range sets {
		if err := s.rulesets.Delete(ctx, set.ID); err != nil {
			writeError(w, 500, err)
			return
		}
	}
	keys, err := s.projectKeys.List(r.Context(), id)
	if err != nil {
		writeError(w, 500, err)
		return
	}
	for _, key := range keys {
		if err := s.projectKeys.Delete(r.Context(), id, key.ID); err != nil {
			writeError(w, 500, err)
			return
		}
	}
	if s.invocationLogs != nil {
		if err := s.invocationLogs.DeleteProject(repo.WorkspaceIDFrom(r.Context()), id); err != nil {
			writeError(w, 500, err)
			return
		}
	}
	if err := s.projects.Delete(r.Context(), id); err != nil {
		s.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"active_project_id": s.sessions.projectID(r)})
}

func (s *Server) setActiveProject(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ProjectID string `json:"project_id"`
	}
	if err := decodeBody(w, r, &input); err != nil {
		writeError(w, 400, err)
		return
	}
	if _, err := s.projects.Get(r.Context(), input.ProjectID); err != nil {
		s.writeServiceError(w, err)
		return
	}
	if !s.sessions.setProject(r, input.ProjectID) {
		writeError(w, 401, errors.New("authentication required"))
		return
	}
	writeJSON(w, 200, map[string]string{"active_project_id": input.ProjectID})
}

func (s *Server) listProjectKeys(w http.ResponseWriter, r *http.Request) {
	if _, err := s.projects.Get(r.Context(), r.PathValue("id")); err != nil {
		s.writeServiceError(w, err)
		return
	}
	items, err := s.projectKeys.List(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]any{"items": items})
}

func (s *Server) createProjectKey(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if _, err := s.projects.Get(r.Context(), projectID); err != nil {
		s.writeServiceError(w, err)
		return
	}
	var input struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := decodeBody(w, r, &input); err != nil {
		writeError(w, 400, err)
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Type = strings.TrimSpace(strings.ToLower(input.Type))
	if input.Type == "" {
		input.Type = "apikey"
	}
	if input.Name == "" || len([]rune(input.Name)) > 80 || (input.Type != "apikey" && input.Type != "jwt") {
		writeError(w, 400, errors.New("名称必填，且凭据类型必须为 apikey 或 jwt"))
		return
	}
	idBytes := make([]byte, 8)
	if _, err := rand.Read(idBytes); err != nil {
		writeError(w, 500, err)
		return
	}
	now := time.Now().UTC()
	keyID := "key-" + hex.EncodeToString(idBytes)
	secret := ""
	if input.Type == "apikey" {
		secretBytes := make([]byte, 32)
		if _, err := rand.Read(secretBytes); err != nil {
			writeError(w, 500, err)
			return
		}
		secret = "sr_live_" + base64.RawURLEncoding.EncodeToString(secretBytes)
	} else {
		claims, err := json.Marshal(projectJWTClaims{Issuer: "snakerule", Subject: keyID, Workspace: repo.WorkspaceIDFrom(r.Context()), Project: projectID, IssuedAt: now.Unix(), ExpiresAt: now.Add(30 * 24 * time.Hour).Unix()})
		if err != nil {
			writeError(w, 500, err)
			return
		}
		header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
		payload := base64.RawURLEncoding.EncodeToString(claims)
		unsigned := header + "." + payload
		mac := hmac.New(sha256.New, s.jwtSigningKey)
		_, _ = mac.Write([]byte(unsigned))
		secret = unsigned + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	}
	digest := sha256.Sum256([]byte(secret))
	key := domain.APIKeyCredential{ID: keyID, Name: input.Name, Type: input.Type, Prefix: secret[:16], Hash: base64.RawURLEncoding.EncodeToString(digest[:]), Workspace: repo.WorkspaceIDFrom(r.Context()), Project: projectID, CreatedAt: now}
	if err := s.projectKeys.Save(r.Context(), key); err != nil {
		writeError(w, 500, err)
		return
	}
	key.Hash = ""
	if input.Type == "jwt" {
		writeJSON(w, 201, map[string]any{"item": key, "token": secret})
		return
	}
	writeJSON(w, 201, map[string]any{"item": key, "secret": secret})
}

func (s *Server) deleteProjectKey(w http.ResponseWriter, r *http.Request) {
	if err := s.projectKeys.Delete(r.Context(), r.PathValue("id"), r.PathValue("keyID")); err != nil {
		s.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) dispatchProjectAlert(ctx context.Context, decision domain.Decision, rulesetOverride *domain.Ruleset, simulated bool) {
	if !decision.Matched {
		return
	}
	var set domain.Ruleset
	if rulesetOverride != nil {
		set = *rulesetOverride
	} else {
		var err error
		set, err = s.rulesets.Get(ctx, decision.RulesetID)
		if err != nil {
			s.logger.Warn("failed to load ruleset for alert dispatch", "ruleset", decision.RulesetID, "error", err)
			return
		}
	}
	hitIDs := make(map[string]bool, len(decision.HitRules))
	for _, hit := range decision.HitRules {
		hitIDs[hit.RuleID] = true
	}
	matchedByChannel := map[string][]string{}
	for _, rule := range set.Rules {
		if !hitIDs[rule.ID] {
			continue
		}
		alert := set.Alert
		if rule.Alert != nil {
			alert = rule.Alert
		}
		if alert == nil || !alert.Enabled {
			continue
		}
		for _, channelID := range alert.ChannelIDs {
			matchedByChannel[channelID] = append(matchedByChannel[channelID], rule.Name)
		}
	}
	if len(matchedByChannel) == 0 {
		return
	}
	project, _ := s.projects.Get(ctx, repo.ProjectIDFrom(ctx))
	channels := make([]domain.AlertChannel, 0, len(matchedByChannel))
	for id := range matchedByChannel {
		channel, err := s.alertChannels.Get(ctx, id)
		if err != nil || !channel.Enabled {
			continue
		}
		channels = append(channels, channel)
	}
	for _, channel := range channels {
		message := fmt.Sprintf("SnakeRule project %s: ruleset %s matched: %s", project.Name, decision.RulesetID, strings.Join(matchedByChannel[channel.ID], "、"))
		if channel.Type == "internal" {
			if s.notifications != nil {
				title := "规则命中告警"
				if simulated {
					title = "设计器模拟命中"
					message = "【模拟】" + message
				}
				_ = s.notifications.Add(domain.Notification{Title: title, Message: message, Project: project.Name, Ruleset: decision.RulesetID, CreatedAt: time.Now().UTC()})
			}
			continue
		}
		if simulated {
			continue
		}
		go func(channel domain.AlertChannel, message string) {
			if err := sendDecisionNotification(channel, message); err != nil {
				s.logger.Warn("rule alert delivery failed", "project", project.ID, "ruleset", decision.RulesetID, "channel", channel.ID, "error", err)
			}
		}(channel, message)
	}
}

func sendDecisionNotification(channel domain.AlertChannel, message string) error {
	body, err := alertPayload(channel.Type, message)
	if err != nil {
		return err
	}
	target := channel.URL
	if channel.Type == "dingtalk" && channel.Secret != "" {
		timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		mac := hmac.New(sha256.New, []byte(channel.Secret))
		_, _ = mac.Write([]byte(timestamp + "\n" + channel.Secret))
		parsed, _ := url.Parse(target)
		query := parsed.Query()
		query.Set("timestamp", timestamp)
		query.Set("sign", base64.StdEncoding.EncodeToString(mac.Sum(nil)))
		parsed.RawQuery = query.Encode()
		target = parsed.String()
	}
	if channel.Type == "feishu" && channel.Secret != "" {
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		mac := hmac.New(sha256.New, []byte(timestamp+"\n"+channel.Secret))
		payload["timestamp"], payload["sign"] = timestamp, base64.StdEncoding.EncodeToString(mac.Sum(nil))
		body, _ = json.Marshal(payload)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if channel.Type == "webhook" && channel.Secret != "" {
		req.Header.Set("Authorization", "Bearer "+channel.Secret)
	}
	client := &http.Client{Timeout: 4 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("destination returned HTTP %d", res.StatusCode)
	}
	return nil
}
