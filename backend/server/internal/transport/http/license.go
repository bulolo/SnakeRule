package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

const maxLicenseFileSize = 1 << 20

func (s *Server) getLicense(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.license.GetStatus())
}

func (s *Server) uploadLicense(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxLicenseFileSize+(1<<10))
	if err := r.ParseMultipartForm(maxLicenseFileSize); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("授权文件不能超过 1 MB"))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil || header.Size <= 0 || header.Size > maxLicenseFileSize {
		writeError(w, http.StatusBadRequest, errors.New("请选择不超过 1 MB 的授权文件"))
		return
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, maxLicenseFileSize+1))
	if err != nil || len(content) > maxLicenseFileSize {
		writeError(w, http.StatusBadRequest, errors.New("无法读取授权文件"))
		return
	}
	key, err := extractLicense(content)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.license.Upload(r.Context(), key); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, s.license.GetStatus())
}

func (s *Server) clearLicense(w http.ResponseWriter, _ *http.Request) {
	if err := s.license.Clear(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, s.license.GetStatus())
}

func extractLicense(content []byte) (string, error) {
	value := strings.TrimSpace(string(content))
	if value == "" {
		return "", errors.New("授权文件为空")
	}
	if !strings.HasPrefix(value, "{") {
		return value, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(content, &payload); err != nil {
		return "", errors.New("授权 JSON 格式无效")
	}
	for _, name := range []string{"signed_token", "license_key", "licenseKey"} {
		if token, ok := payload[name].(string); ok && strings.TrimSpace(token) != "" {
			return strings.TrimSpace(token), nil
		}
	}
	return "", errors.New("授权 JSON 中缺少 signed_token")
}
