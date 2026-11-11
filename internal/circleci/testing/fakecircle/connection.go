// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package fakecircle

import (
	"maps"
	"net/http"
	"net/textproto"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

const (
	forgejoWebhookBaseURL = "https://internal.circleci.com/private/soc/forgejo/webhook/"
	forgejoSubjectPrefix  = "circleci/soc-integrations/forgejo/"
	forgejoIssuerBaseURL  = "https://oidc.circleci.com/org/"
)

type connection struct {
	ID                 uuid.UUID
	OrgID              uuid.UUID
	URL                string
	OwnerID            string
	OwnerLogin         string
	OwnerType          string
	ConnectionAudience string
	CheckoutAudience   string
	RequestHeaders     map[string]string
	WebhookSecret      string
	SecretAddedAt      time.Time
	RotationPending    bool
	CreatedAt          time.Time
}

// AddForgejoRepo makes a Forgejo repository visible to every Forgejo
// connection, so pipelines and triggers can be created for it.
func (s *Service) AddForgejoRepo(fullName, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.forgejoRepos[fullName] = id
}

// AddForgejoUser makes owner resolve as a Forgejo user rather than an
// organization during connection setup.
func (s *Service) AddForgejoUser(login string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.forgejoUsers[login] = true
}

func (s *Service) setupConnectionRoutes(r chi.Router) {
	r.Post("/api/v3/provider/connections/setup", s.postConnectionSetup)
	r.Get("/api/v3/provider/connections", s.listConnections)
	r.Get("/api/v3/provider/connections/{id}", s.getConnection)
	r.Delete("/api/v3/provider/connections/{id}", s.deleteConnection)
	r.Post("/api/v3/provider/connections/{id}/set-oidc-audience", s.postConnectionAudience)
	r.Post("/api/v3/provider/connections/{id}/set-request-headers", s.postConnectionHeaders)
	r.Post("/api/v3/provider/connections/{id}/rotate-webhook-secret", s.postConnectionRotateSecret)
}

type connectionSetupRequest struct {
	Type string `json:"type"`
	VCS  *struct {
		Provider           string            `json:"provider"`
		URL                string            `json:"url"`
		Owner              string            `json:"owner"`
		ConnectionAudience string            `json:"connection_audience"`
		CheckoutAudience   string            `json:"checkout_audience"`
		RequestHeaders     map[string]string `json:"request_headers"`
	} `json:"vcs"`
}

func (s *Service) postConnectionSetup(w http.ResponseWriter, r *http.Request) {
	orgID, ok := filterUUID(w, r, "org_id", true)
	if !ok {
		return
	}

	var req connectionSetupRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		v3Error(w, r, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Type != "vcs" {
		v3Error(w, r, http.StatusBadRequest, `type must be "vcs".`)
		return
	}
	if req.VCS == nil || req.VCS.Provider == "" {
		v3Error(w, r, http.StatusBadRequest, "vcs.provider is required.")
		return
	}
	if req.VCS.Provider != "forgejo" {
		v3Error(w, r, http.StatusBadRequest, "vcs.provider is not a supported provider.")
		return
	}
	for name, value := range map[string]string{
		"url":                 req.VCS.URL,
		"owner":               req.VCS.Owner,
		"connection_audience": req.VCS.ConnectionAudience,
		"checkout_audience":   req.VCS.CheckoutAudience,
	} {
		if value == "" {
			v3Error(w, r, http.StatusBadRequest, "vcs."+name+" is required.")
			return
		}
	}
	userID, ok := audiencesShareUser(req.VCS.ConnectionAudience, req.VCS.CheckoutAudience)
	if !ok {
		v3Error(w, r, http.StatusBadRequest,
			"connection_audience and checkout_audience must be audiences Forgejo generated for the same user")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, c := range s.connections {
		if c.OrgID == orgID {
			v3Error(w, r, http.StatusConflict, "this CircleCI organization already has a connection to this provider")
			return
		}
	}

	c := &connection{
		ID:                 uuid.New(),
		OrgID:              orgID,
		URL:                strings.TrimSuffix(req.VCS.URL, "/"),
		OwnerLogin:         req.VCS.Owner,
		OwnerType:          "organization",
		ConnectionAudience: req.VCS.ConnectionAudience,
		CheckoutAudience:   req.VCS.CheckoutAudience,
		RequestHeaders:     canonicalHeaders(req.VCS.RequestHeaders),
		WebhookSecret:      uuid.NewString(),
		SecretAddedAt:      time.Now().UTC(),
		CreatedAt:          time.Now().UTC(),
	}
	c.OwnerID = "42"
	if s.forgejoUsers[req.VCS.Owner] {
		c.OwnerType = "user"
		c.OwnerID = userID
	}
	s.connections[c.ID] = c

	w.Header().Set("Location", "/api/v3/provider/connections/"+c.ID.String())
	v3Data(w, r, http.StatusCreated, v3Entity{
		ID: &c.ID,
		Attributes: map[string]any{
			"next_step": "configure",
			"owner": map[string]any{
				"id":    c.OwnerID,
				"login": c.OwnerLogin,
				"type":  c.OwnerType,
			},
			"webhook": map[string]any{
				"url":    forgejoWebhookBaseURL + c.ID.String(),
				"secret": c.WebhookSecret,
				"events": []string{"push", "pull_request"},
			},
		},
	})
}

func (s *Service) listConnections(w http.ResponseWriter, r *http.Request) {
	orgID, ok := filterUUID(w, r, "org_id", true)
	if !ok {
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]v3Entity, 0)
	for _, c := range s.connections {
		if c.OrgID == orgID {
			items = append(items, c.entity())
		}
	}
	respond(w, r, http.StatusOK, map[string]any{"data": items, "page": map[string]any{"next": nil, "prev": nil}})
}

func (s *Service) getConnection(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	c, ok := s.lookupConnectionLocked(w, r)
	if !ok {
		return
	}
	v3Data(w, r, http.StatusOK, c.entity())
}

func (s *Service) deleteConnection(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.lookupConnectionLocked(w, r)
	if !ok {
		return
	}
	delete(s.connections, c.ID)
	s.deleteForgejoPipelinesLocked(c.OrgID)
	noContent(w)
}

func (s *Service) postConnectionAudience(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Purpose  string `json:"purpose"`
		Audience string `json:"audience"`
	}
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		v3Error(w, r, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Audience == "" {
		v3Error(w, r, http.StatusBadRequest, "audience is required")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.lookupConnectionLocked(w, r)
	if !ok {
		return
	}
	switch req.Purpose {
	case "connection":
		c.ConnectionAudience = req.Audience
	case "checkout":
		c.CheckoutAudience = req.Audience
	default:
		v3Error(w, r, http.StatusBadRequest, "purpose must be one of connection, checkout")
		return
	}
	v3Data(w, r, http.StatusOK, v3Entity{
		ID:         &c.ID,
		Attributes: map[string]any{"purpose": req.Purpose, "audience": req.Audience},
	})
}

func (s *Service) postConnectionHeaders(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RequestHeaders map[string]string `json:"request_headers"`
	}
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		v3Error(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.lookupConnectionLocked(w, r)
	if !ok {
		return
	}
	c.RequestHeaders = canonicalHeaders(req.RequestHeaders)
	v3Data(w, r, http.StatusOK, v3Entity{
		ID:         &c.ID,
		Attributes: map[string]any{"request_header_names": c.headerNames()},
	})
}

func (s *Service) postConnectionRotateSecret(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.lookupConnectionLocked(w, r)
	if !ok {
		return
	}
	c.WebhookSecret = uuid.NewString()
	c.RotationPending = true
	v3Data(w, r, http.StatusOK, v3Entity{
		ID:         &c.ID,
		Attributes: map[string]any{"webhook": map[string]any{"secret": c.WebhookSecret}},
	})
}

func (s *Service) lookupConnectionLocked(w http.ResponseWriter, r *http.Request) (*connection, bool) {
	id, ok := pathUUID(w, r)
	if !ok {
		return nil, false
	}
	c, ok := s.connections[id]
	if !ok {
		v3Error(w, r, http.StatusNotFound, "connection not found")
		return nil, false
	}
	return c, true
}

func (c *connection) entity() v3Entity {
	access := func(purpose, audience string) map[string]any {
		return map[string]any{
			"type":     "authorized_integration",
			"audience": audience,
			"subject":  forgejoSubjectPrefix + purpose,
			"issuer":   forgejoIssuerBaseURL + c.OrgID.String(),
		}
	}
	credentials := map[string]any{
		"access": map[string]any{
			"connection": access("connection", c.ConnectionAudience),
			"checkout":   access("checkout", c.CheckoutAudience),
		},
		"webhook_secret": map[string]any{
			"current_added_at":    c.SecretAddedAt,
			"is_rotation_pending": c.RotationPending,
		},
	}
	if names := c.headerNames(); len(names) > 0 {
		credentials["request_header_names"] = names
	}
	return v3Entity{
		ID: &c.ID,
		Attributes: map[string]any{
			"provider":     "forgejo",
			"provider_url": c.URL,
			"external_id":  c.OwnerID,
			"target_type":  c.OwnerType,
			"created_at":   c.CreatedAt,
			"updated_at":   c.CreatedAt,
			"credentials":  credentials,
		},
	}
}

func (c *connection) headerNames() []string {
	return slices.Sorted(maps.Keys(c.RequestHeaders))
}

func canonicalHeaders(headers map[string]string) map[string]string {
	out := make(map[string]string, len(headers))
	for name, value := range headers {
		out[textproto.CanonicalMIMEHeaderKey(name)] = value
	}
	return out
}

// audiencesShareUser reports the Forgejo user ID both audiences were generated
// for, mirroring the real API's check that they come from one user's
// Authorized Integrations.
func audiencesShareUser(connectionAudience, checkoutAudience string) (string, bool) {
	userID := func(audience string) string {
		parts := strings.SplitN(audience, ":", 3)
		if len(parts) != 3 || parts[0] != "u" || parts[1] == "" || parts[2] == "" {
			return ""
		}
		return parts[1]
	}
	a, b := userID(connectionAudience), userID(checkoutAudience)
	return a, a != "" && a == b
}

// ConnectionIDForOrg returns the ID of orgID's connection, or "" when it has
// none.
func (s *Service) ConnectionIDForOrg(orgID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.connections {
		if c.OrgID.String() == orgID {
			return c.ID.String()
		}
	}
	return ""
}
