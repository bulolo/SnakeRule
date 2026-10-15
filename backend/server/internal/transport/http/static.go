package httpapi

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed web/*
var webFiles embed.FS

func (s *Server) index(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data, err := webFiles.ReadFile("web/index.html")
	if err != nil {
		http.Error(w, "frontend unavailable", http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(data)
}

func (s *Server) assets() http.Handler {
	assets, err := fs.Sub(webFiles, "web")
	if err != nil {
		return http.NotFoundHandler()
	}
	return http.StripPrefix("/assets/", http.FileServer(http.FS(assets)))
}
