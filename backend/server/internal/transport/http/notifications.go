package httpapi

import (
	"errors"
	"net/http"

	"github.com/szplanner/snakerule/internal/domain"
)

func (s *Server) listNotifications(w http.ResponseWriter, r *http.Request) {
	items := []domain.Notification{}
	if s.notifications == nil {
		writeJSON(w, http.StatusOK, paginatedListData(r, items, nil))
		return
	}
	items = s.notifications.List()
	writeJSON(w, http.StatusOK, paginatedListData(r, items, nil))
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
