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
	"github.com/szplanner/snakerule/internal/domain"
	"github.com/szplanner/snakerule/internal/repo"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func (s *Server) listAlertChannels(w http.ResponseWriter, r *http.Request) {
	if !s.canAccessOrganization(r, repo.OrganizationIDFrom(r.Context())) {
		writeError(w, http.StatusForbidden, errors.New("没有访问该组织的权限"))
		return
	}
	items, err := s.alertChannels.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, paginatedListData(r, items, nil))
}

func (s *Server) saveAlertChannel(w http.ResponseWriter, r *http.Request) {
	if !s.canManageOrganization(r, repo.OrganizationIDFrom(r.Context())) {
		writeError(w, http.StatusForbidden, errors.New("没有管理该组织的权限"))
		return
	}
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
		item.OrganizationID = repo.OrganizationIDFrom(r.Context())
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
	if !s.canManageOrganization(r, repo.OrganizationIDFrom(r.Context())) {
		writeError(w, http.StatusForbidden, errors.New("没有管理该组织的权限"))
		return
	}
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
	if !s.canManageOrganization(r, repo.OrganizationIDFrom(r.Context())) {
		writeError(w, http.StatusForbidden, errors.New("没有管理该组织的权限"))
		return
	}
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
