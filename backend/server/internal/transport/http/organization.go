package httpapi

import (
	"errors"
	"net/http"

	"github.com/szplanner/snakerule/internal/domain"
	"github.com/szplanner/snakerule/internal/repo"
)

func (s *Server) listOrganizations(w http.ResponseWriter, r *http.Request) {
	items, err := s.organizations.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !s.isSuperAdmin(r) {
		allowed := map[string]bool{}
		for _, id := range s.sessionFromRequest(r).OrganizationIDs {
			allowed[id] = true
		}
		filtered := make([]domain.Organization, 0)
		for _, item := range items {
			if allowed[item.ID] {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	writeJSON(w, http.StatusOK, paginatedListData(r, items, map[string]any{"active_organization_id": s.sessions.organizationID(r)}))
}

func (s *Server) createOrganization(w http.ResponseWriter, r *http.Request) {
	if !s.isSuperAdmin(r) {
		writeError(w, http.StatusForbidden, errors.New("仅超级管理员可创建组织"))
		return
	}
	var input struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
		AdminUserID string `json:"admin_user_id"`
	}
	if err := decodeBody(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var admin domain.User
	if input.AdminUserID != "" && s.users != nil {
		var ok bool
		admin, ok = s.users.FindByID(input.AdminUserID)
		if !ok || admin.Role == "superadmin" {
			writeError(w, http.StatusBadRequest, errors.New("请选择有效的用户账号"))
			return
		}
	}
	item, err := s.organizations.Create(r.Context(), input.Name, input.Slug, input.Description, input.AdminUserID)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	if input.AdminUserID != "" && s.users != nil {
		admin.OrganizationIDs = append(admin.OrganizationIDs, item.ID)
		if admin.OrganizationRoles == nil {
			admin.OrganizationRoles = map[string]string{}
		}
		admin.OrganizationRoles[item.ID] = "organization_admin"
		if admin.Role == "organization_admin" {
			admin.Role = "organization_member"
		}
		if err := s.users.Save(admin, ""); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) updateOrganization(w http.ResponseWriter, r *http.Request) {
	if !s.isSuperAdmin(r) {
		writeError(w, http.StatusForbidden, errors.New("仅超级管理员可管理平台组织"))
		return
	}
	var input struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
	}
	if err := decodeBody(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := s.organizations.Update(r.Context(), r.PathValue("id"), input.Name, input.Slug, input.Description)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) deleteOrganization(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.isSuperAdmin(r) {
		writeError(w, http.StatusForbidden, errors.New("仅超级管理员可管理平台组织"))
		return
	}
	if id == repo.DefaultOrganizationID {
		writeError(w, http.StatusConflict, errors.New("默认组织不能删除"))
		return
	}
	if id == s.sessions.organizationID(r) {
		writeError(w, http.StatusConflict, errors.New("请先切换到其他组织，再删除当前组织"))
		return
	}
	if _, err := s.organizations.Get(r.Context(), id); err != nil {
		s.writeServiceError(w, err)
		return
	}
	organizationContext := repo.WithOrganization(r.Context(), id)
	projects, err := s.projects.List(organizationContext)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	for _, project := range projects {
		projectContext := repo.WithProject(organizationContext, project.ID)
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
		keys, _ := s.projectKeys.List(organizationContext, project.ID)
		for _, key := range keys {
			_ = s.projectKeys.Delete(organizationContext, project.ID, key.ID)
		}
		if s.invocationLogs != nil {
			if err := s.invocationLogs.DeleteProject(id, project.ID); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
		}
		if err := s.projects.Delete(organizationContext, project.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}
	if s.alertChannels != nil {
		if err := s.alertChannels.DeleteOrganization(organizationContext, id); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}
	if s.users != nil {
		for _, user := range s.users.List() {
			if !containsString(user.OrganizationIDs, id) {
				continue
			}
			user.OrganizationIDs = removeString(user.OrganizationIDs, id)
			delete(user.OrganizationRoles, id)
			if err := s.users.Save(user, ""); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
		}
	}
	if err := s.organizations.Delete(r.Context(), id); err != nil {
		if errors.Is(err, repo.ErrProtected) {
			writeError(w, http.StatusConflict, errors.New("默认组织不能删除"))
			return
		}
		s.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) setActiveOrganization(w http.ResponseWriter, r *http.Request) {
	var input struct {
		OrganizationID string `json:"organization_id"`
	}
	if err := decodeBody(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := s.organizations.Get(r.Context(), input.OrganizationID)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	if !s.canAccessOrganization(r, item.ID) {
		writeError(w, http.StatusForbidden, errors.New("没有访问该组织的权限"))
		return
	}
	if !s.sessions.setOrganization(r, item.ID) {
		writeError(w, http.StatusUnauthorized, errors.New("authentication required"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"active_organization_id": item.ID})
}
