// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

// Package fakecircle is an in-memory fake of the CircleCI API. It lets the
// API client packages be tested over real HTTP without network access.
package fakecircle

import (
	"errors"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

var (
	errDuplicate = errors.New("duplicate")
	errNotFound  = errors.New("not found")
)

// Service is a fake CircleCI API. It implements http.Handler, so it can be
// served with httptest.NewServer.
type Service struct {
	http.Handler
	tok string

	hit429 atomic.Bool
	hit500 atomic.Bool

	mu       sync.RWMutex
	orgs     map[uuid.UUID]*org
	projects map[uuid.UUID]*project
	contexts map[uuid.UUID]*context

	// Runner (v3) state.
	resourceClasses map[string]*resourceClass
	tokens          map[string]*token
	runners         []*runner
}

// New returns a fake API that accepts tok as its only valid Circle-Token.
func New(tok string) *Service {
	r := chi.NewRouter()
	s := &Service{
		tok:      tok,
		Handler:  r,
		orgs:     make(map[uuid.UUID]*org),
		projects: make(map[uuid.UUID]*project),
		contexts: make(map[uuid.UUID]*context),

		resourceClasses: make(map[string]*resourceClass),
		tokens:          make(map[string]*token),
		runners:         make([]*runner, 0),
	}

	r.Use(s.auth)

	r.Get("/api/test/hello", s.getHello)
	r.Post("/api/test/echo", s.postEcho)
	r.Get("/api/test/429", s.get429)
	r.Get("/api/test/500", s.get500)

	r.Post("/api/v2/organization", s.postOrganization)
	r.Get("/api/v2/organization/{org-id}", s.getOrganizationByID)
	r.Delete("/api/v2/organization/{org-id}", s.deleteOrganization)
	r.Post("/api/v2/organization/{org-id}/project", s.postProject)

	r.Get("/api/v2/project/{org-type}/{org-name}/{project-name}", s.getProject)
	r.Delete("/api/v2/project/{org-type}/{org-name}/{project-name}", s.deleteProject)
	// TODO: GET ONE ENV
	r.Get("/api/v2/project/{org-type}/{org-name}/{project-name}/envvar", s.getProjectEnv)
	r.Post("/api/v2/project/{org-type}/{org-name}/{project-name}/envvar", s.postProjectEnv)
	r.Delete("/api/v2/project/{org-type}/{org-name}/{project-name}/envvar/{env-var}", s.deleteProjectEnv)

	r.Get("/api/v2/context", s.getContextBySlug)
	r.Post("/api/v2/context", s.postContext)
	r.Get("/api/v2/context/{context-id}", s.getContextByID)
	r.Delete("/api/v2/context/{context-id}", s.deleteContext)
	r.Get("/api/v2/context/{context-id}/environment-variable", s.getContextEnv)
	r.Put("/api/v2/context/{context-id}/environment-variable/{env-var}", s.putContextEnv)
	r.Delete("/api/v2/context/{context-id}/environment-variable/{env-var}", s.deleteContextEnv)

	s.setupRunnerRoutes(r)

	return s
}

func (s *Service) getHello(w http.ResponseWriter, r *http.Request) {
	msg(w, r, http.StatusOK, "Hello World!")
}

func (s *Service) postEcho(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := render.DecodeJSON(r.Body, &body); err != nil {
		msg(w, r, http.StatusBadRequest, err.Error())
		return
	}

	respond(w, r, http.StatusOK, body)
}

// get429 fails the first request with a 429 and succeeds afterwards, so client
// retry behaviour can be exercised.
func (s *Service) get429(w http.ResponseWriter, r *http.Request) {
	if !s.hit429.Swap(true) {
		w.Header().Set("Retry-After", "1")
		msg(w, r, http.StatusTooManyRequests, "Too many requests.")
		return
	}

	msg(w, r, http.StatusOK, "Successfully retried.")
}

// get500 fails the first request with a 500 and succeeds afterwards, so client
// retry behaviour can be exercised.
func (s *Service) get500(w http.ResponseWriter, r *http.Request) {
	if !s.hit500.Swap(true) {
		msg(w, r, http.StatusInternalServerError, "Internal server error.")
		return
	}

	msg(w, r, http.StatusOK, "Successfully retried.")
}
