package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/szplanner/snakerule/internal/domain"
	"github.com/szplanner/snakerule/internal/repo"
)

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	if !s.canManageOrganization(r, repo.OrganizationIDFrom(r.Context())) {
		writeError(w, http.StatusForbidden, errors.New("没有管理该组织用户的权限"))
		return
	}
	if s.users == nil {
		writeJSON(w, http.StatusOK, paginatedListData(r, []domain.User{}, nil))
		return
	}
	items := make([]domain.User, 0)
	organizationID := repo.OrganizationIDFrom(r.Context())
	for _, user := range s.users.List() {
		if user.Role == "superadmin" {
			continue
		}
		for _, id := range user.OrganizationIDs {
			if id == organizationID {
				user.OrganizationIDs = []string{organizationID}
				if role := user.OrganizationRoles[organizationID]; role != "" {
					user.OrganizationRoles = map[string]string{organizationID: role}
				} else {
					user.OrganizationRoles = map[string]string{}
				}
				user.ProjectRoles = nil
				items = append(items, user)
				break
			}
		}
	}
	writeJSON(w, http.StatusOK, paginatedListData(r, items, nil))
}

func (s *Server) saveUser(w http.ResponseWriter, r *http.Request) {
	if !s.canManageOrganization(r, repo.OrganizationIDFrom(r.Context())) {
		writeError(w, http.StatusForbidden, errors.New("没有管理该组织用户的权限"))
		return
	}
	if s.users == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("用户存储未初始化"))
		return
	}
	var input struct {
		ID       string `json:"id"`
		Email    string `json:"email"`
		Name     string `json:"name"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := decodeBody(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.Name = strings.TrimSpace(input.Name)
	if input.Role != "organization_admin" && input.Role != "organization_member" {
		input.Role = "organization_member"
	}
	if input.Email == "" || !strings.Contains(input.Email, "@") {
		writeError(w, http.StatusBadRequest, errors.New("请输入有效邮箱"))
		return
	}
	organizationID := repo.OrganizationIDFrom(r.Context())
	if r.Method == http.MethodPost || r.Method == http.MethodPut {
		if r.Method == http.MethodPut {
			input.ID = r.PathValue("id")
		}
		if input.ID != "" {
			existing, ok := s.users.FindByID(input.ID)
			if !ok || existing.Role == "superadmin" || !containsString(existing.OrganizationIDs, organizationID) {
				writeError(w, http.StatusNotFound, repo.ErrNotFound)
				return
			}
			existing.Email, existing.Name = input.Email, input.Name
			if existing.OrganizationRoles == nil {
				existing.OrganizationRoles = map[string]string{}
			}
			if existing.Role == "organization_admin" {
				for _, id := range existing.OrganizationIDs {
					existing.OrganizationRoles[id] = "organization_admin"
				}
				existing.Role = "organization_member"
			}
			if existing.OrganizationRoles[organizationID] == "organization_admin" && input.Role != "organization_admin" && s.organizationAdminCount(organizationID) <= 1 {
				writeError(w, http.StatusConflict, errors.New("组织必须至少保留一名管理员"))
				return
			}
			existing.OrganizationRoles[organizationID] = input.Role
			if err := s.users.Save(existing, input.Password); err != nil {
				writeError(w, http.StatusConflict, err)
				return
			}
			existing.PasswordHash, existing.PasswordSalt = "", ""
			writeJSON(w, http.StatusOK, map[string]any{"item": existing, "organization_role": input.Role})
			return
		}
		if existing, ok := s.users.FindByEmail(input.Email); ok {
			if existing.Role == "superadmin" {
				writeError(w, http.StatusConflict, errors.New("超级管理员已拥有平台访问权限"))
				return
			}
			if !containsString(existing.OrganizationIDs, organizationID) {
				existing.OrganizationIDs = append(existing.OrganizationIDs, organizationID)
			}
			if existing.OrganizationRoles == nil {
				existing.OrganizationRoles = map[string]string{}
			}
			if existing.Role == "organization_admin" {
				for _, id := range existing.OrganizationIDs {
					existing.OrganizationRoles[id] = "organization_admin"
				}
				existing.Role = "organization_member"
			}
			existing.OrganizationRoles[organizationID] = input.Role
			if err := s.users.Save(existing, ""); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			existing.PasswordHash, existing.PasswordSalt = "", ""
			writeJSON(w, http.StatusOK, map[string]any{"item": existing, "organization_role": input.Role})
			return
		}
		if input.Name == "" || len(input.Password) < 10 {
			writeError(w, http.StatusBadRequest, errors.New("新账号需填写姓名和至少 10 位初始密码"))
			return
		}
		user := domain.User{Email: input.Email, Name: input.Name, Role: "organization_member", OrganizationIDs: []string{organizationID}, OrganizationRoles: map[string]string{organizationID: input.Role}}
		if err := s.users.Save(user, input.Password); err != nil {
			writeError(w, http.StatusConflict, err)
			return
		}
		created, _ := s.users.FindByEmail(input.Email)
		created.PasswordHash, created.PasswordSalt = "", ""
		writeJSON(w, http.StatusCreated, map[string]any{"item": created, "organization_role": input.Role})
		return
	}
	writeError(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) {
	if !s.canManageOrganization(r, repo.OrganizationIDFrom(r.Context())) {
		writeError(w, http.StatusForbidden, errors.New("没有管理该组织用户的权限"))
		return
	}
	if s.users == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("用户存储未初始化"))
		return
	}
	user, ok := s.users.FindByID(r.PathValue("id"))
	if !ok || user.Role == "superadmin" {
		writeError(w, http.StatusNotFound, repo.ErrNotFound)
		return
	}
	organizationID := repo.OrganizationIDFrom(r.Context())
	if !containsString(user.OrganizationIDs, organizationID) {
		writeError(w, http.StatusNotFound, repo.ErrNotFound)
		return
	}
	if (user.OrganizationRoles[organizationID] == "organization_admin" || user.Role == "organization_admin") && s.organizationAdminCount(organizationID) <= 1 {
		writeError(w, http.StatusConflict, errors.New("组织必须至少保留一名管理员"))
		return
	}
	user.OrganizationIDs = removeString(user.OrganizationIDs, organizationID)
	delete(user.OrganizationRoles, organizationID)
	projects, _ := s.projects.List(r.Context())
	for _, project := range projects {
		delete(user.ProjectRoles, project.ID)
	}
	if len(user.OrganizationIDs) == 0 {
		if err := s.users.Delete(user.ID); err != nil {
			s.writeServiceError(w, err)
			return
		}
	} else if err := s.users.Save(user, ""); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) organizationAdminCount(organizationID string) int {
	count := 0
	if s.users == nil {
		return count
	}
	for _, user := range s.users.List() {
		if containsString(user.OrganizationIDs, organizationID) && (user.Role == "organization_admin" || user.OrganizationRoles[organizationID] == "organization_admin") {
			count++
		}
	}
	return count
}

func containsString(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func removeString(items []string, value string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item != value {
			out = append(out, item)
		}
	}
	return out
}
