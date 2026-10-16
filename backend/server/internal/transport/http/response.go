package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
)

// APIResponse is the common JSON envelope for all versioned API responses.
// HTTP status communicates transport semantics; code and error_code are stable client fields.
type APIResponse struct {
	Code      int    `json:"code"`
	Msg       string `json:"msg"`
	ErrorCode string `json:"error_code,omitempty"`
	Data      any    `json:"data,omitempty"`
}

type PaginationInfo struct {
	IsPager int `json:"is_pager"`
	Page    int `json:"page"`
	Size    int `json:"size"`
	Total   int `json:"total"`
}

func paginatedList[T any](r *http.Request, items []T) ([]T, PaginationInfo) {
	if items == nil {
		items = []T{}
	}
	pagination := PaginationInfo{Total: len(items)}
	if r.URL.Query().Get("is_pager") == "0" {
		pagination.IsPager = 0
		return items, pagination
	}
	pagination.IsPager = 1
	pagination.Page = positiveQueryInt(r, "page", 1)
	pagination.Size = positiveQueryInt(r, "size", 20)
	if pagination.Size > 100 {
		pagination.Size = 100
	}
	start := (pagination.Page - 1) * pagination.Size
	if start >= len(items) {
		return []T{}, pagination
	}
	end := start + pagination.Size
	if end > len(items) {
		end = len(items)
	}
	return items[start:end], pagination
}

func positiveQueryInt(r *http.Request, key string, fallback int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func paginatedListData[T any](r *http.Request, items []T, extra map[string]any) map[string]any {
	list, pagination := paginatedList(r, items)
	data := map[string]any{"list": list, "pagination": pagination}
	for key, value := range extra {
		data[key] = value
	}
	return data
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	if response, ok := value.(APIResponse); ok {
		writeRawJSON(w, status, response)
		return
	}
	writeRawJSON(w, status, APIResponse{Code: 0, Msg: "success", Data: value})
}

func writeRawJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	message := "internal server error"
	if status < http.StatusInternalServerError && err != nil {
		message = err.Error()
	}
	writeJSON(w, status, APIResponse{Code: 1, Msg: message, ErrorCode: errorCodeForStatus(status)})
}

func errorCodeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "invalid_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusConflict:
		return "conflict"
	default:
		if status >= http.StatusInternalServerError {
			return "internal_error"
		}
		return "request_failed"
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
