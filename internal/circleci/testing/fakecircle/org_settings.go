// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package fakecircle

import (
	"encoding/json"
	"maps"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// defaultOrgSettings are the settings a new org starts with. They are the
// values a real organization returned from GET /api/v3/orgs/{id}/settings,
// and the keys are every setting that endpoint reports.
var defaultOrgSettings = map[string]bool{
	"is_runner_terms_of_service_accepted":   false,
	"enable_ai_error_summarization":         true,
	"enable_ai_agents":                      true,
	"enable_unversioned_config":             false,
	"enable_certified_public_orbs":          true,
	"enable_chunk_ip_ranges":                false,
	"enable_marketing_tracking":             true,
	"enable_minor_ai_features":              true,
	"enable_preferences_tracking":           true,
	"enable_private_orbs":                   false,
	"enable_statistics_tracking":            true,
	"enable_uncertified_public_orbs":        true,
	"is_user_checkout_keys_disabled":        false,
	"is_running_disabled":                   false,
	"is_privacy_optout_enforced":            false,
	"enable_image_brownouts":                true,
	"is_context_group_restriction_required": false,
	"enable_resource_class_brownouts":       true,
	"is_code_signing_restricted_to_admins":  false,
}

func (s *Service) setupOrgSettingsRoutes(r chi.Router) {
	r.Get("/api/v3/orgs/{id}/settings", s.getOrgSettings)
	r.Post("/api/v3/orgs/{id}/update-settings", s.postOrgSettings)
}

// orgSettingsLocked returns the settings of the org named by the path,
// creating them from the defaults on first use. It writes a 404 and returns
// false when there is no such org. s.mu must be held for writing.
func (s *Service) orgSettingsLocked(w http.ResponseWriter, r *http.Request) (map[string]bool, bool) {
	id, ok := pathUUID(w, r)
	if !ok {
		return nil, false
	}
	if _, found := s.orgs[id]; !found {
		v3Error(w, r, http.StatusNotFound, "org.not_found")
		return nil, false
	}
	settings, found := s.orgSettings[id]
	if !found {
		settings = maps.Clone(defaultOrgSettings)
		s.orgSettings[id] = settings
	}
	return settings, true
}

// orgSettingsEntity is the body the settings endpoints answer with. It has
// no id or references, as observed on the real API.
func orgSettingsEntity(settings map[string]bool) v3Entity {
	return v3Entity{Attributes: maps.Clone(settings)}
}

func (s *Service) getOrgSettings(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	settings, ok := s.orgSettingsLocked(w, r)
	if !ok {
		return
	}
	v3Data(w, r, http.StatusOK, orgSettingsEntity(settings))
}

// postOrgSettings applies the settings the request carries and leaves the
// rest alone. It accepts {"data":{"attributes":{<setting>: <bool>, ...}}}
// and rejects anything else, including a setting name the GET endpoint does
// not report.
func (s *Service) postOrgSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Data struct {
			Attributes map[string]json.RawMessage `json:"attributes"`
		} `json:"data"`
	}
	if err := decodeStrict(r, &req); err != nil {
		v3Error(w, r, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	changes := make(map[string]bool, len(req.Data.Attributes))
	for name, raw := range req.Data.Attributes {
		if _, known := defaultOrgSettings[name]; !known {
			v3Error(w, r, http.StatusBadRequest, "unknown setting "+name)
			return
		}
		var value bool
		if err := json.Unmarshal(raw, &value); err != nil {
			v3Error(w, r, http.StatusBadRequest, name+" must be a boolean")
			return
		}
		changes[name] = value
	}
	if len(changes) == 0 {
		v3Error(w, r, http.StatusBadRequest, "no settings to update")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	settings, ok := s.orgSettingsLocked(w, r)
	if !ok {
		return
	}
	s.orgSettingsRequests = append(s.orgSettingsRequests, maps.Clone(changes))
	maps.Copy(settings, changes)
	v3Data(w, r, http.StatusOK, orgSettingsEntity(settings))
}

// OrgSettingsRequests returns the attributes of every update-settings request
// the fake applied, oldest first. Each map holds only the settings the request
// carried, so a test can tell a setting that was left out from one sent as
// false.
func (s *Service) OrgSettingsRequests() []map[string]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]map[string]bool, 0, len(s.orgSettingsRequests))
	for _, req := range s.orgSettingsRequests {
		out = append(out, maps.Clone(req))
	}
	return out
}

// OrgSettings returns the current settings of orgID, which are the defaults
// until something changes them.
func (s *Service) OrgSettings(orgID uuid.UUID) map[string]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if settings, found := s.orgSettings[orgID]; found {
		return maps.Clone(settings)
	}
	return maps.Clone(defaultOrgSettings)
}
