package httpapi

import (
	"errors"
	"net/http"

	"github.com/szplanner/snakerule/internal/repo"
)

func (s *Server) listWorkspaces(w http.ResponseWriter, r *http.Request) {
	items, err := s.workspaces.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "active_workspace_id": s.sessions.workspaceID(r)})
}

func (s *Server) createWorkspace(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
	}
	if err := decodeBody(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := s.workspaces.Create(r.Context(), input.Name, input.Slug, input.Description)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) updateWorkspace(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
	}
	if err := decodeBody(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := s.workspaces.Update(r.Context(), r.PathValue("id"), input.Name, input.Slug, input.Description)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) deleteWorkspace(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == repo.DefaultWorkspaceID {
		writeError(w, http.StatusConflict, errors.New("默认工作空间不能删除"))
		return
	}
	if id == s.sessions.workspaceID(r) {
		writeError(w, http.StatusConflict, errors.New("请先切换到其他工作空间，再删除当前工作空间"))
		return
	}
	if _, err := s.workspaces.Get(r.Context(), id); err != nil {
		s.writeServiceError(w, err)
		return
	}
	workspaceContext := repo.WithWorkspace(r.Context(), id)
	projects, err := s.projects.List(workspaceContext)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	for _, project := range projects {
		projectContext := repo.WithProject(workspaceContext, project.ID)
		items, err := s.rulesets.List(projectContext)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		for _, item := range items {
			if err := s.rulesets.Delete(projectContext, item.ID); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
		}
		keys, _ := s.projectKeys.List(workspaceContext, project.ID)
		for _, key := range keys {
			_ = s.projectKeys.Delete(workspaceContext, project.ID, key.ID)
		}
		if project.ID != repo.DefaultProjectID {
			if err := s.projects.Delete(workspaceContext, project.ID); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
		}
	}
	if err := s.workspaces.Delete(r.Context(), id); err != nil {
		if errors.Is(err, repo.ErrProtected) {
			writeError(w, http.StatusConflict, errors.New("默认工作空间不能删除"))
			return
		}
		s.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) setActiveWorkspace(w http.ResponseWriter, r *http.Request) {
	var input struct {
		WorkspaceID string `json:"workspace_id"`
	}
	if err := decodeBody(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := s.workspaces.Get(r.Context(), input.WorkspaceID)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	if !s.sessions.setWorkspace(r, item.ID) {
		writeError(w, http.StatusUnauthorized, errors.New("authentication required"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"active_workspace_id": item.ID})
}
