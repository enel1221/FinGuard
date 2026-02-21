package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/inelson/finguard/internal/models"
	"github.com/inelson/finguard/internal/store"
)

// @Summary      Create a project
// @Description  Create a new project for organizing cost sources
// @Tags         Projects
// @Accept       json
// @Produce      json
// @Param        body  body      object{name=string,description=string}  true  "Project fields"
// @Success      201   {object}  models.Project
// @Failure      400   {object}  object{error=string}
// @Failure      500   {object}  object{error=string}
// @Security     SessionAuth
// @Router       /projects [post]
func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	project := &models.Project{
		Name:        req.Name,
		Description: req.Description,
	}
	if err := s.store.CreateProject(r.Context(), project); err != nil {
		s.logger.Error("failed to create project", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create project"})
		return
	}

	writeJSON(w, http.StatusCreated, project)
}

// @Summary      Get a project
// @Description  Returns a single project by ID
// @Tags         Projects
// @Produce      json
// @Param        projectID  path      string  true  "Project ID"
// @Success      200        {object}  models.Project
// @Failure      404        {object}  object{error=string}
// @Failure      500        {object}  object{error=string}
// @Security     SessionAuth
// @Router       /projects/{projectID} [get]
func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "projectID")
	project, err := s.store.GetProject(r.Context(), id)
	if err != nil {
		s.logger.Error("failed to get project", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get project"})
		return
	}
	if project == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found"})
		return
	}
	writeJSON(w, http.StatusOK, project)
}

// @Summary      List projects
// @Description  Returns all projects
// @Tags         Projects
// @Produce      json
// @Success      200  {object}  object{projects=[]models.Project}
// @Failure      500  {object}  object{error=string}
// @Security     SessionAuth
// @Router       /projects [get]
func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.store.ListProjects(r.Context())
	if err != nil {
		s.logger.Error("failed to list projects", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list projects"})
		return
	}
	if projects == nil {
		projects = []*models.Project{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": projects})
}

// @Summary      Update a project
// @Description  Update an existing project's name and/or description
// @Tags         Projects
// @Accept       json
// @Produce      json
// @Param        projectID  path      string                                   true  "Project ID"
// @Param        body       body      object{name=string,description=string}   true  "Fields to update"
// @Success      200        {object}  models.Project
// @Failure      400        {object}  object{error=string}
// @Failure      404        {object}  object{error=string}
// @Failure      500        {object}  object{error=string}
// @Security     SessionAuth
// @Router       /projects/{projectID} [put]
func (s *Server) handleUpdateProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "projectID")

	existing, err := s.store.GetProject(r.Context(), id)
	if err != nil {
		s.logger.Error("failed to get project", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get project"})
		return
	}
	if existing == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found"})
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}

	if err := s.store.UpdateProject(r.Context(), existing); err != nil {
		s.logger.Error("failed to update project", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update project"})
		return
	}

	writeJSON(w, http.StatusOK, existing)
}

// @Summary      Delete a project
// @Description  Delete a project and all associated cost sources, members, and costs
// @Tags         Projects
// @Produce      json
// @Param        projectID  path      string  true  "Project ID"
// @Success      200        {object}  object{status=string}
// @Failure      500        {object}  object{error=string}
// @Security     SessionAuth
// @Router       /projects/{projectID} [delete]
func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "projectID")
	if err := s.store.DeleteProject(r.Context(), id); err != nil {
		s.logger.Error("failed to delete project", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete project"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- Cost Sources ---

// @Summary      Create a cost source
// @Description  Add a new cost source (AWS, Azure, GCP, Kubernetes, or plugin) to a project
// @Tags         CostSources
// @Accept       json
// @Produce      json
// @Param        projectID  path      string                                                   true  "Project ID"
// @Param        body       body      object{type=string,name=string,config=object,enabled=bool}  true  "Cost source fields"
// @Success      201        {object}  models.CostSource
// @Failure      400        {object}  object{error=string}
// @Failure      500        {object}  object{error=string}
// @Security     SessionAuth
// @Router       /projects/{projectID}/sources [post]
func (s *Server) handleCreateCostSource(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")

	var req struct {
		Type    models.CostSourceType `json:"type"`
		Name    string                `json:"name"`
		Config  json.RawMessage       `json:"config"`
		Enabled *bool                 `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" || req.Type == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and type are required"})
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	cs := &models.CostSource{
		ProjectID: projectID,
		Type:      req.Type,
		Name:      req.Name,
		Config:    req.Config,
		Enabled:   enabled,
	}

	if err := s.store.CreateCostSource(r.Context(), cs); err != nil {
		s.logger.Error("failed to create cost source", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create cost source"})
		return
	}

	writeJSON(w, http.StatusCreated, cs)
}

// @Summary      List cost sources
// @Description  Returns all cost sources for a project
// @Tags         CostSources
// @Produce      json
// @Param        projectID  path      string  true  "Project ID"
// @Success      200        {object}  object{sources=[]models.CostSource}
// @Failure      500        {object}  object{error=string}
// @Security     SessionAuth
// @Router       /projects/{projectID}/sources [get]
func (s *Server) handleListCostSources(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	sources, err := s.store.ListCostSources(r.Context(), projectID)
	if err != nil {
		s.logger.Error("failed to list cost sources", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list cost sources"})
		return
	}
	if sources == nil {
		sources = []*models.CostSource{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"sources": sources})
}

// @Summary      Get a cost source
// @Description  Returns a single cost source by ID
// @Tags         CostSources
// @Produce      json
// @Param        projectID  path      string  true  "Project ID"
// @Param        sourceID   path      string  true  "Cost source ID"
// @Success      200        {object}  models.CostSource
// @Failure      404        {object}  object{error=string}
// @Failure      500        {object}  object{error=string}
// @Security     SessionAuth
// @Router       /projects/{projectID}/sources/{sourceID} [get]
func (s *Server) handleGetCostSource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "sourceID")
	cs, err := s.store.GetCostSource(r.Context(), id)
	if err != nil {
		s.logger.Error("failed to get cost source", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get cost source"})
		return
	}
	if cs == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "cost source not found"})
		return
	}
	writeJSON(w, http.StatusOK, cs)
}

// @Summary      Delete a cost source
// @Description  Remove a cost source from a project
// @Tags         CostSources
// @Produce      json
// @Param        projectID  path      string  true  "Project ID"
// @Param        sourceID   path      string  true  "Cost source ID"
// @Success      200        {object}  object{status=string}
// @Failure      500        {object}  object{error=string}
// @Security     SessionAuth
// @Router       /projects/{projectID}/sources/{sourceID} [delete]
func (s *Server) handleDeleteCostSource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "sourceID")
	if err := s.store.DeleteCostSource(r.Context(), id); err != nil {
		s.logger.Error("failed to delete cost source", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete cost source"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- Project Members ---

// @Summary      List project members
// @Description  Returns all role assignments for a project
// @Tags         Members
// @Produce      json
// @Param        projectID  path      string  true  "Project ID"
// @Success      200        {object}  object{members=[]models.ProjectRole}
// @Failure      500        {object}  object{error=string}
// @Security     SessionAuth
// @Router       /projects/{projectID}/members [get]
func (s *Server) handleListProjectMembers(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	roles, err := s.store.ListProjectRoles(r.Context(), projectID)
	if err != nil {
		s.logger.Error("failed to list project members", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list members"})
		return
	}
	if roles == nil {
		roles = []*models.ProjectRole{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": roles})
}

// @Summary      Add a project member
// @Description  Assign a role to a user or group for a project
// @Tags         Members
// @Accept       json
// @Produce      json
// @Param        projectID  path      string                                                  true  "Project ID"
// @Param        body       body      object{subjectType=string,subjectId=string,role=string}  true  "Member role assignment"
// @Success      201        {object}  models.ProjectRole
// @Failure      400        {object}  object{error=string}
// @Failure      500        {object}  object{error=string}
// @Security     SessionAuth
// @Router       /projects/{projectID}/members [post]
func (s *Server) handleAddProjectMember(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")

	var req struct {
		SubjectType models.SubjectType `json:"subjectType"`
		SubjectID   string             `json:"subjectId"`
		Role        models.Role        `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	pr := &models.ProjectRole{
		ProjectID:   projectID,
		SubjectType: req.SubjectType,
		SubjectID:   req.SubjectID,
		Role:        req.Role,
	}
	if err := s.store.SetProjectRole(r.Context(), pr); err != nil {
		s.logger.Error("failed to set project role", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to add member"})
		return
	}

	writeJSON(w, http.StatusCreated, pr)
}

// @Summary      Remove a project member
// @Description  Remove a user or group role assignment from a project
// @Tags         Members
// @Produce      json
// @Param        projectID    path      string  true   "Project ID"
// @Param        subjectID    path      string  true   "Subject ID (user or group)"
// @Param        subjectType  query     string  false  "Subject type: user or group"  default(user)
// @Success      200          {object}  object{status=string}
// @Failure      500          {object}  object{error=string}
// @Security     SessionAuth
// @Router       /projects/{projectID}/members/{subjectID} [delete]
func (s *Server) handleRemoveProjectMember(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	subjectID := chi.URLParam(r, "subjectID")
	subjectType := models.SubjectType(r.URL.Query().Get("subjectType"))
	if subjectType == "" {
		subjectType = models.SubjectUser
	}

	if err := s.store.RemoveProjectRole(r.Context(), projectID, subjectType, subjectID); err != nil {
		s.logger.Error("failed to remove project member", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to remove member"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

// --- Project Costs ---

// @Summary      Get project costs
// @Description  Returns aggregated cost summary for a project
// @Tags         Costs
// @Produce      json
// @Param        projectID  path      string  true  "Project ID"
// @Success      200        {object}  store.CostSummary
// @Failure      500        {object}  object{error=string}
// @Security     SessionAuth
// @Router       /projects/{projectID}/costs [get]
func (s *Server) handleGetProjectCosts(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")

	summary, err := s.store.AggregateCosts(r.Context(), store.CostQuery{
		ProjectID: projectID,
	})
	if err != nil {
		s.logger.Error("failed to aggregate costs", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get costs"})
		return
	}

	writeJSON(w, http.StatusOK, summary)
}
