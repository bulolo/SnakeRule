package httpapi

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/szplanner/snakerule/internal/repo"
)

func (s *Server) getPlatformSettings(w http.ResponseWriter, r *http.Request) {
	if s.platformSettings == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("平台设置存储未初始化"))
		return
	}
	settings, err := s.platformSettings.Get(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (s *Server) updatePlatformSettings(w http.ResponseWriter, r *http.Request) {
	if s.platformSettings == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("平台设置存储未初始化"))
		return
	}
	var settings repo.PlatformSettings
	if err := decodeBody(w, r, &settings); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	settings.Name = strings.TrimSpace(settings.Name)
	settings.Tagline = strings.TrimSpace(settings.Tagline)
	settings.LogoURL = strings.TrimSpace(settings.LogoURL)
	if settings.Name == "" || len([]rune(settings.Name)) > 60 || len([]rune(settings.Tagline)) > 120 {
		writeError(w, http.StatusBadRequest, errors.New("产品名称必填且不超过 60 个字符，标语不能超过 120 个字符"))
		return
	}
	if settings.LogoURL != "" {
		parsed, err := url.ParseRequestURI(settings.LogoURL)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
			writeError(w, http.StatusBadRequest, errors.New("Logo 地址必须是有效的 HTTP(S) URL"))
			return
		}
	}
	if err := s.platformSettings.Save(r.Context(), settings); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}
