// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package fakecircle

import (
	"cmp"
	"net/http"
	"slices"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// notificationPreferencePageSize is deliberately small so that listing a
// project's preferences takes several pages and exercises client.V3ListAll.
const notificationPreferencePageSize = 2

// NonConfigurablePreferenceID is a preference the fake seeds with
// is_user_configurable false. The real ID of such a preference has not been
// observed; this one only exists in the fake.
const NonConfigurablePreferenceID = "0f0e0d0c-0b0a-4908-8706-050403020100"

type notificationPreferenceDef struct {
	ID               string
	Name             string
	GroupName        string
	GroupOrder       int
	SectionName      string
	SectionOrder     int
	Order            int
	UserConfigurable bool
	EnabledByDefault bool
}

// notificationPreferenceDefs is the global catalogue of preferences. The first
// five IDs and names were observed in production; their defaults were not, and
// the non-configurable entry is made up.
var notificationPreferenceDefs = []notificationPreferenceDef{
	{
		ID: "c9930e26-d1e1-46e2-a5d2-45c77843fcd2", Name: "On default branch (failed, error, unauthorized)",
		GroupName: "Workflow Completed", GroupOrder: 1, SectionName: "Build", SectionOrder: 2, Order: 1,
		UserConfigurable: true, EnabledByDefault: true,
	},
	{
		ID: "3b5d94e8-6feb-4eb6-8458-ae2571fb4192", Name: "Job awaiting approval",
		GroupName: "Job", GroupOrder: 2, SectionName: "Build", SectionOrder: 2, Order: 1,
		UserConfigurable: true,
	},
	{
		ID: "796a37f9-7b11-4e92-9322-927e76583cf3", Name: "Succeeded",
		GroupName: "Releases", GroupOrder: 1, SectionName: "Deployments", SectionOrder: 3, Order: 1,
		UserConfigurable: true,
	},
	{
		ID: "4234f1ce-9bcd-4a9c-966c-4e365c69e1ed", Name: "Failed",
		GroupName: "Releases", GroupOrder: 1, SectionName: "Deployments", SectionOrder: 3, Order: 2,
		UserConfigurable: true,
	},
	{
		ID: "74d43e43-955f-4596-bda6-c0ef96ba3119", Name: "Inconclusive",
		GroupName: "Releases", GroupOrder: 1, SectionName: "Deployments", SectionOrder: 3, Order: 3,
		UserConfigurable: true,
	},
	{
		ID: NonConfigurablePreferenceID, Name: "Plan usage",
		GroupName: "Account", GroupOrder: 1, SectionName: "Organization", SectionOrder: 1, Order: 1,
		UserConfigurable: false, EnabledByDefault: true,
	},
}

type channelConfig struct {
	ID        uuid.UUID
	OrgID     uuid.UUID
	ProjectID uuid.UUID
	Target    string
	Enabled   bool
}

type notificationState struct {
	channelConfigs map[uuid.UUID]*channelConfig
	// preferences holds each project's preference values, seeded from the
	// catalogue defaults the first time the project is touched.
	preferences map[uuid.UUID]map[string]bool
	// slackChannelNames maps a Slack channel ID to the name CircleCI reports.
	slackChannelNames map[string]string
	// welcomeMessages counts the writes that make CircleCI post its welcome
	// message into the Slack channel.
	welcomeMessages int
}

func newNotificationState() notificationState {
	return notificationState{
		channelConfigs:    make(map[uuid.UUID]*channelConfig),
		preferences:       make(map[uuid.UUID]map[string]bool),
		slackChannelNames: make(map[string]string),
	}
}

func (s *Service) setupNotificationRoutes(r chi.Router) {
	r.Get("/api/v3/notification/channel-configs", s.listChannelConfigs)
	r.Post("/api/v3/notification/channel-configs", s.postChannelConfig)
	r.Get("/api/v3/notification/channel-configs/{id}", s.getChannelConfig)
	r.Post("/api/v3/notification/channel-configs/{id}/update", s.postChannelConfigUpdate)
	r.Delete("/api/v3/notification/channel-configs/{id}", s.deleteChannelConfig)
	r.Get("/api/v3/notification/preferences", s.listNotificationPreferences)
	r.Post("/api/v3/notification/preferences", s.postNotificationPreferences)
}

// AddSlackChannel makes the fake report name as the channel_name of any
// channel config targeting the Slack channel id. Without it, channel_name is
// left out of responses, as observed for the production list endpoint.
func (s *Service) AddSlackChannel(id, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notifications.slackChannelNames[id] = name
}

// SlackWelcomeMessages reports how many times CircleCI would have posted its
// "Welcome! CircleCI is now configured to post updates" message: on every
// create and every update that leaves the config enabled, even when nothing
// changed.
func (s *Service) SlackWelcomeMessages() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.notifications.welcomeMessages
}

func (s *Service) channelConfigEntity(c *channelConfig) v3Entity {
	attrs := map[string]any{
		"channel_type": "slack",
		"target":       c.Target,
		"is_enabled":   c.Enabled,
	}
	if name, ok := s.notifications.slackChannelNames[c.Target]; ok {
		attrs["channel_name"] = name
	}
	return v3Entity{
		ID:         &c.ID,
		Attributes: attrs,
		References: map[string]any{
			"project": v3Ref{ID: c.ProjectID},
			"org":     v3Ref{ID: c.OrgID},
		},
	}
}

// projectInOrgLocked reports whether projectID is a project of orgID. A nil
// orgID matches any organization.
func (s *Service) projectInOrgLocked(orgID, projectID uuid.UUID) bool {
	p, ok := s.projects[projectID]
	if !ok {
		return false
	}
	return orgID == uuid.Nil || p.Org.id == orgID
}

func scopeIsProject(w http.ResponseWriter, r *http.Request, scope string) bool {
	if scope != "project" {
		v3Error(w, r, http.StatusBadRequest, "scope must be project")
		return false
	}
	return true
}

func (s *Service) listChannelConfigs(w http.ResponseWriter, r *http.Request) {
	orgID, ok := filterUUID(w, r, "org_id", true)
	if !ok {
		return
	}
	projectID, ok := filterUUID(w, r, "project_id", false)
	if !ok {
		return
	}
	if scope := r.URL.Query().Get("filter[scope]"); scope != "" && !scopeIsProject(w, r, scope) {
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]v3Entity, 0)
	for _, c := range s.notifications.channelConfigs {
		if c.OrgID != orgID || (projectID != uuid.Nil && c.ProjectID != projectID) {
			continue
		}
		items = append(items, s.channelConfigEntity(c))
	}
	v3Collection(w, r, items)
}

type channelConfigCreateBody struct {
	Data struct {
		Attributes struct {
			ChannelType string `json:"channel_type"`
			Scope       string `json:"scope"`
			Target      string `json:"target"`
			IsEnabled   *bool  `json:"is_enabled"`
		} `json:"attributes"`
		References struct {
			Org     v3Ref `json:"org"`
			Project v3Ref `json:"project"`
		} `json:"references"`
	} `json:"data"`
}

func (s *Service) postChannelConfig(w http.ResponseWriter, r *http.Request) {
	var req channelConfigCreateBody
	if err := decodeStrict(r, &req); err != nil {
		v3Error(w, r, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	attrs, refs := req.Data.Attributes, req.Data.References
	if attrs.ChannelType != "slack" {
		v3Error(w, r, http.StatusBadRequest, "channel_type must be slack")
		return
	}
	if !scopeIsProject(w, r, attrs.Scope) {
		return
	}
	if attrs.Target == "" {
		v3Error(w, r, http.StatusBadRequest, "target is required")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.projectInOrgLocked(refs.Org.ID, refs.Project.ID) || refs.Org.ID == uuid.Nil {
		v3Error(w, r, http.StatusNotFound, "project not found")
		return
	}
	// Assumed: the UI only allows one Slack channel per project, so a second
	// config is taken to be a conflict.
	for _, c := range s.notifications.channelConfigs {
		if c.ProjectID == refs.Project.ID {
			v3Error(w, r, http.StatusConflict, "a channel config already exists for this project")
			return
		}
	}

	c := &channelConfig{
		ID:        uuid.New(),
		OrgID:     refs.Org.ID,
		ProjectID: refs.Project.ID,
		Target:    attrs.Target,
		Enabled:   attrs.IsEnabled == nil || *attrs.IsEnabled,
	}
	s.notifications.channelConfigs[c.ID] = c
	if c.Enabled {
		s.notifications.welcomeMessages++
	}
	v3Data(w, r, http.StatusCreated, s.channelConfigEntity(c))
}

func (s *Service) lookupChannelConfigLocked(w http.ResponseWriter, r *http.Request) (*channelConfig, bool) {
	id, ok := pathUUID(w, r)
	if !ok {
		return nil, false
	}
	c, found := s.notifications.channelConfigs[id]
	if !found {
		v3Error(w, r, http.StatusNotFound, "channel config not found")
		return nil, false
	}
	return c, true
}

func (s *Service) getChannelConfig(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	c, ok := s.lookupChannelConfigLocked(w, r)
	if !ok {
		return
	}
	v3Data(w, r, http.StatusOK, s.channelConfigEntity(c))
}

// postChannelConfigUpdate takes a body that is not wrapped in data, and only
// target and is_enabled, which is how the working restapi configuration sends
// it. Any other field is rejected.
func (s *Service) postChannelConfigUpdate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Target    *string `json:"target"`
		IsEnabled *bool   `json:"is_enabled"`
	}
	if err := decodeStrict(r, &req); err != nil {
		v3Error(w, r, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.Target != nil && *req.Target == "" {
		v3Error(w, r, http.StatusBadRequest, "target must not be empty")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.lookupChannelConfigLocked(w, r)
	if !ok {
		return
	}
	if req.Target != nil {
		c.Target = *req.Target
	}
	if req.IsEnabled != nil {
		c.Enabled = *req.IsEnabled
	}
	if c.Enabled {
		s.notifications.welcomeMessages++
	}
	v3Data(w, r, http.StatusOK, s.channelConfigEntity(c))
}

func (s *Service) deleteChannelConfig(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.lookupChannelConfigLocked(w, r)
	if !ok {
		return
	}
	delete(s.notifications.channelConfigs, c.ID)
	noContent(w)
}

// projectPreferencesLocked returns the preference values of projectID,
// seeding them from the catalogue defaults. s.mu must be held for writing.
func (s *Service) projectPreferencesLocked(projectID uuid.UUID) map[string]bool {
	prefs, ok := s.notifications.preferences[projectID]
	if !ok {
		prefs = make(map[string]bool, len(notificationPreferenceDefs))
		for _, d := range notificationPreferenceDefs {
			prefs[d.ID] = d.EnabledByDefault
		}
		s.notifications.preferences[projectID] = prefs
	}
	return prefs
}

func preferenceEntity(d notificationPreferenceDef, orgID, projectID uuid.UUID, enabled bool) v3Entity {
	id := uuid.MustParse(d.ID)
	return v3Entity{
		ID: &id,
		Attributes: map[string]any{
			"preference_name":          d.Name,
			"channel":                  "slack",
			"scope":                    "project",
			"preference_display_order": d.Order,
			"preference_availability":  "general",
			"group_id":                 uuid.NewSHA1(uuid.NameSpaceOID, []byte(d.GroupName)),
			"group_name":               d.GroupName,
			"group_display_order":      d.GroupOrder,
			"section_id":               uuid.NewSHA1(uuid.NameSpaceOID, []byte(d.SectionName)),
			"section_name":             d.SectionName,
			"section_display_order":    d.SectionOrder,
			"section_description":      nil,
			"section_docs_ref":         nil,
			"is_user_configurable":     d.UserConfigurable,
			"is_enabled":               enabled,
		},
		References: map[string]any{
			"project": v3Ref{ID: projectID},
			"org":     v3Ref{ID: orgID},
		},
	}
}

// listNotificationPreferences pages through the catalogue with an offset
// cursor. filter[org_id] is optional, as the restapi configuration leaves it
// out.
func (s *Service) listNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	orgID, ok := filterUUID(w, r, "org_id", false)
	if !ok {
		return
	}
	projectID, ok := filterUUID(w, r, "project_id", true)
	if !ok {
		return
	}
	if !scopeIsProject(w, r, r.URL.Query().Get("filter[scope]")) {
		return
	}
	offset := 0
	if raw := r.URL.Query().Get("page[cursor]"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 || n > len(notificationPreferenceDefs) {
			v3Error(w, r, http.StatusBadRequest, "invalid page[cursor]")
			return
		}
		offset = n
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.projectInOrgLocked(orgID, projectID) {
		v3Error(w, r, http.StatusNotFound, "project not found")
		return
	}
	prefs := s.projectPreferencesLocked(projectID)
	projectOrg := s.projects[projectID].Org.id

	defs := slices.Clone(notificationPreferenceDefs)
	slices.SortStableFunc(defs, func(a, b notificationPreferenceDef) int {
		return cmp.Or(
			cmp.Compare(a.SectionOrder, b.SectionOrder),
			cmp.Compare(a.GroupOrder, b.GroupOrder),
			cmp.Compare(a.Order, b.Order),
		)
	})
	end := min(offset+notificationPreferencePageSize, len(defs))
	resp := v3CollectionResponse{Data: []v3Entity{}}
	for _, d := range defs[offset:end] {
		resp.Data = append(resp.Data, preferenceEntity(d, projectOrg, projectID, prefs[d.ID]))
	}
	if end < len(defs) {
		next := strconv.Itoa(end)
		resp.Page.Next = &next
	}
	respond(w, r, http.StatusOK, resp)
}

type preferencesUpdateBody struct {
	Data struct {
		Attributes struct {
			Scope   string `json:"scope"`
			Updates []struct {
				PreferenceID string `json:"preference_id"`
				IsEnabled    *bool  `json:"is_enabled"`
			} `json:"updates"`
		} `json:"attributes"`
		References struct {
			Org     v3Ref `json:"org"`
			Project v3Ref `json:"project"`
		} `json:"references"`
	} `json:"data"`
}

// postNotificationPreferences applies a bulk update. It is all or nothing:
// an unknown preference, or one that is not user configurable, rejects the
// whole request. That the real API rejects them is assumed, not observed.
func (s *Service) postNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	var req preferencesUpdateBody
	if err := decodeStrict(r, &req); err != nil {
		v3Error(w, r, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	attrs, refs := req.Data.Attributes, req.Data.References
	if !scopeIsProject(w, r, attrs.Scope) {
		return
	}
	if len(attrs.Updates) == 0 {
		v3Error(w, r, http.StatusBadRequest, "updates is required")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if refs.Org.ID == uuid.Nil || !s.projectInOrgLocked(refs.Org.ID, refs.Project.ID) {
		v3Error(w, r, http.StatusNotFound, "project not found")
		return
	}
	for _, u := range attrs.Updates {
		i := slices.IndexFunc(notificationPreferenceDefs, func(d notificationPreferenceDef) bool {
			return d.ID == u.PreferenceID
		})
		switch {
		case i < 0:
			v3Error(w, r, http.StatusBadRequest, "unknown preference "+u.PreferenceID)
			return
		case !notificationPreferenceDefs[i].UserConfigurable:
			v3Error(w, r, http.StatusBadRequest, "preference "+u.PreferenceID+" is not user configurable")
			return
		case u.IsEnabled == nil:
			v3Error(w, r, http.StatusBadRequest, "is_enabled is required")
			return
		}
	}
	prefs := s.projectPreferencesLocked(refs.Project.ID)
	for _, u := range attrs.Updates {
		prefs[u.PreferenceID] = *u.IsEnabled
	}
	noContent(w)
}
