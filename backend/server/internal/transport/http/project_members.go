package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/szplanner/snakerule/internal/domain"
	"github.com/szplanner/snakerule/internal/repo"
)

func (s *Server) listProjectMembers(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	items := make([]any, 0)
	if s.users != nil {
		for _, user := range s.users.List() {
			role := user.ProjectRoles[projectID]
			if role == "project_admin" || role == "project_member" {
				items = append(items, map[string]any{"user": map[string]string{"id": user.ID, "name": user.Name, "email": user.Email}, "role": role})
			}
		}
	}
	writeJSON(w, http.StatusOK, paginatedListData(r, items, map[string]any{"can_manage": s.canManageProject(r, projectID)}))
}

func (s *Server) listProjectMemberCandidates(w http.ResponseWriter, r *http.Request) {
	items := make([]domain.User, 0)
	if s.users != nil {
		for _, user := range s.users.List() {
			if user.Role == "superadmin" || !containsString(user.OrganizationIDs, repo.OrganizationIDFrom(r.Context())) {
				continue
			}
			if user.Role == "organization_admin" || user.OrganizationRoles[repo.OrganizationIDFrom(r.Context())] == "organization_admin" {
				continue
			}
			user.OrganizationIDs = []string{repo.OrganizationIDFrom(r.Context())}
			user.OrganizationRoles = nil
			user.ProjectRoles = nil
			items = append(items, user)
		}
	}
	writeJSON(w, http.StatusOK, paginatedListData(r, items, nil))
}

func (s *Server) addProjectMember(w http.ResponseWriter, r *http.Request) {
	if s.users == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("用户存储未初始化"))
		return
	}
	var input struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := decodeBody(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if input.Role != "project_admin" && input.Role != "project_member" {
		writeError(w, http.StatusBadRequest, errors.New("项目角色必须是 project_admin 或 project_member"))
		return
	}
	user, ok := s.users.FindByID(input.UserID)
	if !ok || user.Role == "superadmin" {
		writeError(w, http.StatusNotFound, repo.ErrNotFound)
		return
	}
	if !containsString(user.OrganizationIDs, repo.OrganizationIDFrom(r.Context())) {
		writeError(w, http.StatusConflict, errors.New("用户必须先加入当前组织"))
		return
	}
	if user.Role == "organization_admin" || user.OrganizationRoles[repo.OrganizationIDFrom(r.Context())] == "organization_admin" {
		writeError(w, http.StatusConflict, errors.New("组织管理员默认管理此空间内的全部项目"))
		return
	}
	if user.ProjectRoles == nil {
		user.ProjectRoles = map[string]string{}
	}
	user.ProjectRoles[r.PathValue("id")] = input.Role
	if err := s.users.Save(user, ""); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user": map[string]string{"id": user.ID, "name": user.Name, "email": user.Email}, "role": input.Role})
}

func (s *Server) updateProjectMember(w http.ResponseWriter, r *http.Request) {
	if s.users == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("用户存储未初始化"))
		return
	}
	var input struct {
		Role string `json:"role"`
	}
	if err := decodeBody(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if input.Role != "project_admin" && input.Role != "project_member" {
		writeError(w, http.StatusBadRequest, errors.New("项目角色必须是 project_admin 或 project_member"))
		return
	}
	projectID, userID := r.PathValue("id"), r.PathValue("userID")
	user, ok := s.users.FindByID(userID)
	if !ok || user.ProjectRoles[projectID] == "" {
		writeError(w, http.StatusNotFound, repo.ErrNotFound)
		return
	}
	if strings.EqualFold(user.Email, s.sessionFromRequest(r).Email) {
		writeError(w, http.StatusConflict, errors.New("不能修改自己的项目角色"))
		return
	}
	if user.ProjectRoles[projectID] == "project_admin" && input.Role != "project_admin" && s.projectAdminCount(projectID) <= 1 {
		writeError(w, http.StatusConflict, errors.New("项目必须至少保留一名项目管理员"))
		return
	}
	user.ProjectRoles[projectID] = input.Role
	if err := s.users.Save(user, ""); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user_id": userID, "role": input.Role})
}

func (s *Server) removeProjectMember(w http.ResponseWriter, r *http.Request) {
	if s.users == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("用户存储未初始化"))
		return
	}
	projectID, userID := r.PathValue("id"), r.PathValue("userID")
	user, ok := s.users.FindByID(userID)
	if !ok || user.ProjectRoles[projectID] == "" {
		writeError(w, http.StatusNotFound, repo.ErrNotFound)
		return
	}
	if strings.EqualFold(user.Email, s.sessionFromRequest(r).Email) {
		writeError(w, http.StatusConflict, errors.New("不能将自己移出项目"))
		return
	}
	if user.ProjectRoles[projectID] == "project_admin" && s.projectAdminCount(projectID) <= 1 {
		writeError(w, http.StatusConflict, errors.New("项目必须至少保留一名项目管理员"))
		return
	}
	delete(user.ProjectRoles, projectID)
	if err := s.users.Save(user, ""); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) projectAdminCount(projectID string) int {
	count := 0
	if s.users != nil {
		for _, user := range s.users.List() {
			if user.ProjectRoles[projectID] == "project_admin" {
				count++
			}
		}
	}
	return count
}
