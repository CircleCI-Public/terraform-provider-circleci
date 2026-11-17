// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/orgsettings"
)

var (
	_ resource.Resource                = &organizationSettingsResource{}
	_ resource.ResourceWithConfigure   = &organizationSettingsResource{}
	_ resource.ResourceWithImportState = &organizationSettingsResource{}
)

type organizationSettingsResourceModel struct {
	Id                                types.String `tfsdk:"id"`
	OrganizationId                    types.String `tfsdk:"organization_id"`
	EnableAIErrorSummarization        types.Bool   `tfsdk:"enable_ai_error_summarization"`
	EnableAIAgents                    types.Bool   `tfsdk:"enable_ai_agents"`
	EnableMinorAIFeatures             types.Bool   `tfsdk:"enable_minor_ai_features"`
	EnableUnversionedConfig           types.Bool   `tfsdk:"enable_unversioned_config"`
	EnableCertifiedPublicOrbs         types.Bool   `tfsdk:"enable_certified_public_orbs"`
	EnableUncertifiedPublicOrbs       types.Bool   `tfsdk:"enable_uncertified_public_orbs"`
	EnablePrivateOrbs                 types.Bool   `tfsdk:"enable_private_orbs"`
	EnableChunkIPRanges               types.Bool   `tfsdk:"enable_chunk_ip_ranges"`
	EnableMarketingTracking           types.Bool   `tfsdk:"enable_marketing_tracking"`
	EnablePreferencesTracking         types.Bool   `tfsdk:"enable_preferences_tracking"`
	EnableStatisticsTracking          types.Bool   `tfsdk:"enable_statistics_tracking"`
	IsPrivacyOptoutEnforced           types.Bool   `tfsdk:"is_privacy_optout_enforced"`
	IsUserCheckoutKeysDisabled        types.Bool   `tfsdk:"is_user_checkout_keys_disabled"`
	EnableImageBrownouts              types.Bool   `tfsdk:"enable_image_brownouts"`
	EnableResourceClassBrownouts      types.Bool   `tfsdk:"enable_resource_class_brownouts"`
	IsContextGroupRestrictionRequired types.Bool   `tfsdk:"is_context_group_restriction_required"`
}

// orgSetting pairs a setting's attribute in the model with its field in the
// API settings, so every conversion walks one list instead of sixteen lines.
type orgSetting struct {
	tf  *types.Bool
	api **bool
}

func (m *organizationSettingsResourceModel) settings(s *orgsettings.Settings) []orgSetting {
	return []orgSetting{
		{&m.EnableAIErrorSummarization, &s.EnableAIErrorSummarization},
		{&m.EnableAIAgents, &s.EnableAIAgents},
		{&m.EnableMinorAIFeatures, &s.EnableMinorAIFeatures},
		{&m.EnableUnversionedConfig, &s.EnableUnversionedConfig},
		{&m.EnableCertifiedPublicOrbs, &s.EnableCertifiedPublicOrbs},
		{&m.EnableUncertifiedPublicOrbs, &s.EnableUncertifiedPublicOrbs},
		{&m.EnablePrivateOrbs, &s.EnablePrivateOrbs},
		{&m.EnableChunkIPRanges, &s.EnableChunkIPRanges},
		{&m.EnableMarketingTracking, &s.EnableMarketingTracking},
		{&m.EnablePreferencesTracking, &s.EnablePreferencesTracking},
		{&m.EnableStatisticsTracking, &s.EnableStatisticsTracking},
		{&m.IsPrivacyOptoutEnforced, &s.IsPrivacyOptoutEnforced},
		{&m.IsUserCheckoutKeysDisabled, &s.IsUserCheckoutKeysDisabled},
		{&m.EnableImageBrownouts, &s.EnableImageBrownouts},
		{&m.EnableResourceClassBrownouts, &s.EnableResourceClassBrownouts},
		{&m.IsContextGroupRestrictionRequired, &s.IsContextGroupRestrictionRequired},
	}
}

// changes returns the settings the plan sets. Null and unknown are omitted:
// an unknown value is a setting the configuration leaves out, and writing it
// would change a setting the practitioner did not choose.
func (m *organizationSettingsResourceModel) changes() orgsettings.Settings {
	var s orgsettings.Settings
	for _, f := range m.settings(&s) {
		*f.api = optionalBool(*f.tf)
	}
	return s
}

// setSettings copies every setting CircleCI reported into the model.
func (m *organizationSettingsResourceModel) setSettings(s orgsettings.Settings) {
	for _, f := range m.settings(&s) {
		*f.tf = types.BoolPointerValue(*f.api)
	}
}

// orgSettingDescriptions describes each managed setting. The name in
// parentheses is the one `circleci org setting` uses.
var orgSettingDescriptions = map[string]string{
	"enable_ai_error_summarization":         "Whether CircleCI summarizes job failures with AI (`ai-error-summarization`).",
	"enable_ai_agents":                      "Whether AI agents are enabled (`ai-agents`).",
	"enable_minor_ai_features":              "Whether the smaller AI-assisted features are enabled (`minor-ai-features`).",
	"enable_unversioned_config":             "Whether unversioned configuration is allowed (`unversioned-config`).",
	"enable_certified_public_orbs":          "Whether pipelines may use certified public orbs (`certified-public-orbs`).",
	"enable_uncertified_public_orbs":        "Whether pipelines may use uncertified public orbs, the ones not published by CircleCI or its partners (`uncertified-public-orbs`).",
	"enable_private_orbs":                   "Whether private orbs are enabled (`private-orbs`).",
	"enable_chunk_ip_ranges":                "Whether IP ranges are enabled for Chunk (`chunk-ip-ranges`).",
	"enable_marketing_tracking":             "Whether marketing tracking is enabled (`marketing-tracking`).",
	"enable_preferences_tracking":           "Whether preferences tracking is enabled (`preferences-tracking`).",
	"enable_statistics_tracking":            "Whether statistics tracking is enabled (`statistics-tracking`).",
	"is_privacy_optout_enforced":            "Whether the privacy opt-out is enforced for the organization (`enforce-privacy-optout`).",
	"is_user_checkout_keys_disabled":        "Whether user checkout keys are disabled for the organization's projects (`disable-user-checkout-keys`).",
	"enable_image_brownouts":                "Whether the organization takes part in brownouts of deprecated images, the scheduled periods in which they are unavailable ahead of their removal (`image-brownouts`).",
	"enable_resource_class_brownouts":       "Whether the organization takes part in brownouts of deprecated resource classes (`resource-class-brownouts`).",
	"is_context_group_restriction_required": "Whether contexts must be restricted to a security group (`context-group-restriction`).",
}

// NewOrganizationSettingsResource returns the resource for an organization's
// settings.
func NewOrganizationSettingsResource() resource.Resource {
	return &organizationSettingsResource{}
}

type organizationSettingsResource struct {
	client *orgsettings.Service
}

func (r *organizationSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_settings"
}

func (r *organizationSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			MarkdownDescription: "The ID of the organization, the same as `organization_id`.",
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"organization_id": schema.StringAttribute{
			MarkdownDescription: "The ID of the organization whose settings to manage. Changing this value forces a new resource.",
			Required:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
	}
	for name, description := range orgSettingDescriptions {
		attrs[name] = schema.BoolAttribute{
			MarkdownDescription: description + " Left unchanged when not set.",
			Optional:            true,
			Computed:            true,
		}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the settings of a CircleCI organization. Every organization has its settings, so creating " +
			"this resource takes over the settings that already exist, and destroying it only removes them from the " +
			"Terraform state: CircleCI keeps the values last applied. The provider only writes the settings the " +
			"configuration sets; any other setting keeps its value on CircleCI, is reported in state, and is never " +
			"reported as a change. Changing settings requires an organization admin.",
		Attributes: attrs,
	}
}

// notFoundDetail explains a 404, which the V3 API returns both for an
// organization that does not exist and for a token that may not access it.
const notFoundDetail = "CircleCI answers 404 both when the organization does not exist and when the token may not " +
	"access its settings. Check organization_id, and that the token belongs to an organization admin."

func orgSettingsError(diags *diag.Diagnostics, summary string, err error) {
	detail := err.Error()
	if client.IsNotFound(err) {
		detail += "\n\n" + notFoundDetail
	}
	diags.AddError(summary, detail)
}

func (r *organizationSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan organizationSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	orgID := plan.OrganizationId.ValueString()

	// The settings already exist. Reading them first reports a missing
	// organization or a token without access as such, before any write.
	if _, err := r.client.Get(ctx, orgID); err != nil {
		orgSettingsError(&resp.Diagnostics, "Error reading CircleCI organization settings for "+orgID, err)
		return
	}

	settings, err := r.client.Update(ctx, orgID, plan.changes())
	if err != nil {
		orgSettingsError(&resp.Diagnostics, "Error updating CircleCI organization settings for "+orgID, err)
		return
	}
	plan.Id = types.StringValue(orgID)
	plan.setSettings(*settings)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read fails rather than removing the resource from state when CircleCI
// answers 404. The settings exist for as long as the organization does, so a
// 404 means a wrong organization_id or a token without access, and planning to
// create the settings again would neither fix nor explain that.
func (r *organizationSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state organizationSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	orgID := state.OrganizationId.ValueString()

	settings, err := r.client.Get(ctx, orgID)
	if err != nil {
		orgSettingsError(&resp.Diagnostics, "Error reading CircleCI organization settings for "+orgID, err)
		return
	}
	state.Id = types.StringValue(orgID)
	state.setSettings(*settings)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *organizationSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan organizationSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	orgID := plan.OrganizationId.ValueString()

	settings, err := r.client.Update(ctx, orgID, plan.changes())
	if err != nil {
		orgSettingsError(&resp.Diagnostics, "Error updating CircleCI organization settings for "+orgID, err)
		return
	}
	plan.Id = types.StringValue(orgID)
	plan.setSettings(*settings)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete only forgets the settings. They cannot be deleted, and resetting them
// to some default would change an organization the practitioner merely
// stopped managing.
func (r *organizationSettingsResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *organizationSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	wrapper, ok := req.ProviderData.(*CircleCiClientWrapper)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *CircleCiClientWrapper, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = wrapper.OrgSettingsService
}

// ImportState accepts the organization ID.
func (r *organizationSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
