// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package fakecircle

import (
	"net/http"
	"slices"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var projectRoles = []string{"project-admin", "project-contributor", "project-viewer"}

type group struct {
	ID           uuid.UUID
	OrgID        uuid.UUID
	Name         string
	Description  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Members      map[uuid.UUID]bool
	ProjectRoles map[uuid.UUID]string
}

func (s *Service) setupGroupRoutes(r chi.Router) {
	r.Post("/api/v3/groups", s.postGroup)
	r.Get("/api/v3/groups", s.listGroups)
	r.Get("/api/v3/groups/{id}", s.getGroup)
	r.Put("/api/v3/groups/{id}", s.putGroup)
	r.Delete("/api/v3/groups/{id}", s.deleteGroup)
	r.Get("/api/v3/groups/{id}/members", s.listGroupMembers)
	r.Post("/api/v3/groups/{id}/add-members", s.postGroupMembers(true))
	r.Post("/api/v3/groups/{id}/remove-members", s.postGroupMembers(false))
	r.Post("/api/v3/groups/{id}/add-projects", s.postGroupProjects(true))
	r.Post("/api/v3/groups/{id}/remove-projects", s.postGroupProjects(false))
}

func (g *group) entity() v3Entity {
	return v3Entity{
		ID: &g.ID,
		Attributes: map[string]any{
			"name":        g.Name,
			"description": g.Description,
			"created_at":  g.CreatedAt,
			"updated_at":  g.UpdatedAt,
		},
		References: map[string]any{"org": v3Ref{ID: g.OrgID}},
	}
}

type groupBody struct {
	Data struct {
		Attributes struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"attributes"`
		References struct {
			Org v3Ref `json:"org"`
		} `json:"references"`
	} `json:"data"`
}

func (s *Service) postGroup(w http.ResponseWriter, r *http.Request) {
	var req groupBody
	if err := decodeStrict(r, &req); err != nil {
		v3Error(w, r, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.Data.Attributes.Name == "" {
		v3Error(w, r, http.StatusBadRequest, "name is required.")
		return
	}
	if req.Data.References.Org.ID == uuid.Nil {
		v3Error(w, r, http.StatusBadRequest, "org.id is required.")
		return
	}

	now := time.Now().UTC()
	g := &group{
		ID:           uuid.New(),
		OrgID:        req.Data.References.Org.ID,
		Name:         req.Data.Attributes.Name,
		Description:  req.Data.Attributes.Description,
		CreatedAt:    now,
		UpdatedAt:    now,
		Members:      make(map[uuid.UUID]bool),
		ProjectRoles: make(map[uuid.UUID]string),
	}

	s.mu.Lock()
	s.groups[g.ID] = g
	s.mu.Unlock()

	v3Data(w, r, http.StatusCreated, g.entity())
}

func (s *Service) listGroups(w http.ResponseWriter, r *http.Request) {
	orgID, ok := filterUUID(w, r, "org_id", true)
	if !ok {
		return
	}
	projectID, ok := filterUUID(w, r, "project_id", false)
	if !ok {
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]v3Entity, 0)
	for _, g := range s.groups {
		if g.OrgID != orgID {
			continue
		}
		attrs := map[string]any{"name": g.Name}
		if projectID != uuid.Nil {
			role, granted := g.ProjectRoles[projectID]
			if !granted {
				continue
			}
			attrs["role"] = role
		}
		items = append(items, v3Entity{ID: &g.ID, Attributes: attrs})
	}
	v3Collection(w, r, items)
}

func (s *Service) getGroup(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	g, ok := s.lookupGroupLocked(w, r)
	if !ok {
		return
	}
	v3Data(w, r, http.StatusOK, g.entity())
}

func (s *Service) putGroup(w http.ResponseWriter, r *http.Request) {
	var req groupBody
	if err := decodeStrict(r, &req); err != nil {
		v3Error(w, r, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.Data.Attributes.Name == "" {
		v3Error(w, r, http.StatusBadRequest, "name is required.")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	g, ok := s.lookupGroupLocked(w, r)
	if !ok {
		return
	}
	g.Name = req.Data.Attributes.Name
	g.Description = req.Data.Attributes.Description
	g.UpdatedAt = time.Now().UTC()
	v3Data(w, r, http.StatusOK, g.entity())
}

func (s *Service) deleteGroup(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	g, ok := s.lookupGroupLocked(w, r)
	if !ok {
		return
	}
	delete(s.groups, g.ID)
	noContent(w)
}

func (s *Service) listGroupMembers(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	g, ok := s.lookupGroupLocked(w, r)
	if !ok {
		return
	}
	items := make([]v3Entity, 0, len(g.Members))
	for userID := range g.Members {
		items = append(items, v3Entity{
			ID:         &userID,
			Attributes: map[string]any{},
			References: map[string]any{"group": v3Ref{ID: g.ID}},
		})
	}
	v3Collection(w, r, items)
}

func (s *Service) postGroupMembers(add bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Users []v3Ref `json:"users"`
		}
		if err := decodeStrict(r, &req); err != nil {
			v3Error(w, r, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}
		if len(req.Users) == 0 {
			v3Error(w, r, http.StatusBadRequest, "users is required.")
			return
		}

		s.mu.Lock()
		defer s.mu.Unlock()

		g, ok := s.lookupGroupLocked(w, r)
		if !ok {
			return
		}
		for _, u := range req.Users {
			if add {
				g.Members[u.ID] = true
			} else {
				delete(g.Members, u.ID)
			}
		}
		noContent(w)
	}
}

func (s *Service) postGroupProjects(add bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Role     string  `json:"role"`
			Projects []v3Ref `json:"projects"`
		}
		if err := decodeStrict(r, &req); err != nil {
			v3Error(w, r, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}
		if add && !slices.Contains(projectRoles, req.Role) {
			v3Error(w, r, http.StatusBadRequest, "role is not a project role.")
			return
		}
		if len(req.Projects) == 0 {
			v3Error(w, r, http.StatusBadRequest, "projects is required.")
			return
		}

		s.mu.Lock()
		defer s.mu.Unlock()

		g, ok := s.lookupGroupLocked(w, r)
		if !ok {
			return
		}
		for _, p := range req.Projects {
			if add {
				g.ProjectRoles[p.ID] = req.Role
			} else {
				delete(g.ProjectRoles, p.ID)
			}
		}
		noContent(w)
	}
}

func (s *Service) lookupGroupLocked(w http.ResponseWriter, r *http.Request) (*group, bool) {
	id, ok := pathUUID(w, r)
	if !ok {
		return nil, false
	}
	orgID, ok := filterUUID(w, r, "org_id", true)
	if !ok {
		return nil, false
	}
	g, found := s.groups[id]
	if !found || g.OrgID != orgID {
		v3Error(w, r, http.StatusNotFound, "group not found")
		return nil, false
	}
	return g, true
}
