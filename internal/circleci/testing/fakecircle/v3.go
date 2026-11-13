// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package fakecircle

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (s *Service) setupV3Routes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(s.bearerOnly)
		s.setupConnectionRoutes(r)
		s.setupPipelineV3Routes(r)
	})
}

type v3Ref struct {
	ID uuid.UUID `json:"id"`
}

type v3Entity struct {
	ID         *uuid.UUID `json:"id,omitempty"`
	Attributes any        `json:"attributes"`
	References any        `json:"references,omitempty"`
}

type v3DataResponse struct {
	Data v3Entity `json:"data"`
}

func v3Data(w http.ResponseWriter, r *http.Request, statusCode int, e v3Entity) {
	respond(w, r, statusCode, v3DataResponse{Data: e})
}

// v3Error writes the error envelope the V3 API uses.
func v3Error(w http.ResponseWriter, r *http.Request, statusCode int, title string) {
	respond(w, r, statusCode, map[string]any{
		"error": map[string]any{"title": title},
	})
}

func noContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func pathUUID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		v3Error(w, r, http.StatusBadRequest, "id must be a UUID")
		return uuid.Nil, false
	}
	return id, true
}

// filterUUID parses filter[name]. It writes a 400 and returns false when the
// value is not a UUID, and returns uuid.Nil, true when it is absent and not
// required.
func filterUUID(w http.ResponseWriter, r *http.Request, name string, required bool) (uuid.UUID, bool) {
	raw := r.URL.Query().Get("filter[" + name + "]")
	if raw == "" {
		if required {
			v3Error(w, r, http.StatusBadRequest, "filter["+name+"] is required")
			return uuid.Nil, false
		}
		return uuid.Nil, true
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		v3Error(w, r, http.StatusBadRequest, "filter["+name+"] must be a UUID")
		return uuid.Nil, false
	}
	return id, true
}
