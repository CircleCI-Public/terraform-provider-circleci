// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/connection"
)

var (
	_ resource.Resource                   = &forgejoConnectionResource{}
	_ resource.ResourceWithConfigure      = &forgejoConnectionResource{}
	_ resource.ResourceWithImportState    = &forgejoConnectionResource{}
	_ resource.ResourceWithModifyPlan     = &forgejoConnectionResource{}
	_ resource.ResourceWithValidateConfig = &forgejoConnectionResource{}
)

var forgejoAudiencePattern = regexp.MustCompile(`^u:[^:]+:.+$`)

type forgejoConnectionResourceModel struct {
	Id                    types.String `tfsdk:"id"`
	OrganizationId        types.String `tfsdk:"organization_id"`
	Url                   types.String `tfsdk:"url"`
	Owner                 types.String `tfsdk:"owner"`
	ConnectionAudience    types.String `tfsdk:"connection_audience"`
	CheckoutAudience      types.String `tfsdk:"checkout_audience"`
	RequestHeaders        types.Map    `tfsdk:"request_headers"`
	OwnerId               types.String `tfsdk:"owner_id"`
	OwnerType             types.String `tfsdk:"owner_type"`
	WebhookUrl            types.String `tfsdk:"webhook_url"`
	WebhookEvents         types.List   `tfsdk:"webhook_events"`
	WebhookSecret         types.String `tfsdk:"webhook_secret"`
	WebhookSecretRotation types.String `tfsdk:"webhook_secret_rotation"`
}

// NewForgejoConnectionResource returns the resource that connects a CircleCI
// organization to a Forgejo instance.
func NewForgejoConnectionResource() resource.Resource {
	return &forgejoConnectionResource{}
}

type forgejoConnectionResource struct {
	client *connection.Service
}

func (r *forgejoConnectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_forgejo_connection"
}

func (r *forgejoConnectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	audienceValidators := []validator.String{
		stringvalidator.RegexMatches(forgejoAudiencePattern,
			"must be the audience Forgejo generated for the Authorized Integration, of the form u:<user id>:<uuid>"),
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Connects a CircleCI organization to a Forgejo instance, so the organization's pipelines can check out code from and be triggered by Forgejo repositories. " +
			"An organization can have only one Forgejo connection.\n\n" +
			"CircleCI checks the instance before it stores the connection, so the following must already exist in Forgejo (version 16 or later): " +
			"a user without site administrator rights that can see `owner`, and two Authorized Integrations created by that user. " +
			"Both integrations trust the issuer `https://oidc.circleci.com/org/<organization_id>`, and each has a claim rule requiring `sub` to equal its subject: " +
			"`circleci/soc-integrations/forgejo/connection` for the connection integration (scopes `read:user`, `read:organization` and `read:repository`, since CircleCI also uses it to look up repositories and fetch configuration), and " +
			"`circleci/soc-integrations/forgejo/checkout` for the checkout integration (scope `read:repository`).\n\n" +
			"After creating the connection, create a webhook in Forgejo for `owner` that sends `webhook_events` to `webhook_url`, signed with `webhook_secret`.\n\n" +
			"Destroying the connection also deletes every pipeline and trigger CircleCI holds for its repositories. It changes nothing in Forgejo, so delete the webhook and both Authorized Integrations there yourself.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the connection.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the CircleCI organization to connect. Changing this value forces a new resource, which deletes the organization's Forgejo pipelines and triggers.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "The base URL of the Forgejo instance, which must be reachable from CircleCI. Changing this value forces a new resource, which deletes the organization's Forgejo pipelines and triggers. CircleCI reports it only in a canonical form, which can differ from the configured value, so after an import it is taken from the configuration.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{requiresReplaceIfPreviouslyKnown()},
			},
			"owner": schema.StringAttribute{
				MarkdownDescription: "The Forgejo organization, or user, whose repositories CircleCI builds. A user owner must be the user that created the Authorized Integrations. Changing this value forces a new resource, which deletes the organization's Forgejo pipelines and triggers. CircleCI does not report it, so after an import it is taken from the configuration.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{requiresReplaceIfPreviouslyKnown()},
			},
			"connection_audience": schema.StringAttribute{
				MarkdownDescription: "The audience Forgejo generated for the connection Authorized Integration, copied from the integration's settings.",
				Required:            true,
				Validators:          audienceValidators,
			},
			"checkout_audience": schema.StringAttribute{
				MarkdownDescription: "The audience Forgejo generated for the checkout Authorized Integration, copied from the integration's settings.",
				Required:            true,
				Validators:          audienceValidators,
			},
			"request_headers": schema.MapAttribute{
				MarkdownDescription: "Headers CircleCI sends with every request to the Forgejo instance, for example to authenticate to a proxy in front of it. " +
					"CircleCI never returns header values, so only added or removed headers are detected as drift.",
				Optional:    true,
				Sensitive:   true,
				ElementType: types.StringType,
			},
			"owner_id": schema.StringAttribute{
				MarkdownDescription: "The Forgejo ID of `owner`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"owner_type": schema.StringAttribute{
				MarkdownDescription: "Whether `owner` is a Forgejo `organization` or `user`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"webhook_url": schema.StringAttribute{
				MarkdownDescription: "The URL of the webhook to create in Forgejo. Not set after an import.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"webhook_events": schema.ListAttribute{
				MarkdownDescription: "The events the Forgejo webhook must send. Not set after an import.",
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"webhook_secret": schema.StringAttribute{
				MarkdownDescription: "The secret the Forgejo webhook signs deliveries with. CircleCI returns it only when the connection is created and when the secret is rotated, so it is not set after an import until `webhook_secret_rotation` is changed.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"webhook_secret_rotation": schema.StringAttribute{
				MarkdownDescription: "Any value. Setting it to a new value issues a new `webhook_secret`; update the Forgejo webhook with it. Removing it does not. The previous secret keeps verifying deliveries until one signed with the new secret arrives.",
				Optional:            true,
			},
		},
	}
}

func (r *forgejoConnectionResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config forgejoConnectionResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.RequestHeaders.IsNull() || config.RequestHeaders.IsUnknown() {
		return
	}

	seen := map[string]string{}
	for name := range config.RequestHeaders.Elements() {
		canonical := strings.ToLower(name)
		if other, ok := seen[canonical]; ok {
			resp.Diagnostics.AddAttributeError(path.Root("request_headers"), "Duplicate request header",
				fmt.Sprintf("Header names are case-insensitive, so %q and %q name the same header.", other, name))
			return
		}
		seen[canonical] = name
	}
}

// ModifyPlan marks webhook_secret as unknown when webhook_secret_rotation
// changes, because the apply will replace it.
func (r *forgejoConnectionResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var plan, state forgejoConnectionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.WebhookSecretRotation.Equal(state.WebhookSecretRotation) && !plan.WebhookSecretRotation.IsNull() {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("webhook_secret"), types.StringUnknown())...)
	}
}

func (r *forgejoConnectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan forgejoConnectionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	headers, diags := stringMap(ctx, plan.RequestHeaders)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.SetupForgejo(ctx, plan.OrganizationId.ValueString(), connection.ForgejoSetup{
		URL:                plan.Url.ValueString(),
		Owner:              plan.Owner.ValueString(),
		ConnectionAudience: plan.ConnectionAudience.ValueString(),
		CheckoutAudience:   plan.CheckoutAudience.ValueString(),
		RequestHeaders:     headers,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating CircleCI Forgejo connection", err.Error())
		return
	}

	plan.Id = types.StringValue(result.ID)
	plan.OwnerId = types.StringValue(result.Owner.ID)
	plan.OwnerType = types.StringValue(result.Owner.Type)
	plan.WebhookUrl = types.StringValue(result.Webhook.URL)
	plan.WebhookSecret = types.StringValue(result.Webhook.Secret)
	events, diags := types.ListValueFrom(ctx, types.StringType, result.Webhook.Events)
	resp.Diagnostics.Append(diags...)
	plan.WebhookEvents = events

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *forgejoConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state forgejoConnectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := r.client.Get(ctx, state.Id.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading CircleCI Forgejo connection "+state.Id.ValueString(), err.Error())
		return
	}
	if conn.Provider != connection.ProviderForgejo {
		resp.Diagnostics.AddError("Unexpected CircleCI connection provider",
			fmt.Sprintf("Connection %s is a %s connection, not a forgejo one.", conn.ID, conn.Provider))
		return
	}

	state.OwnerId = types.StringValue(conn.ExternalID)
	state.OwnerType = types.StringValue(conn.TargetType)
	if state.WebhookEvents.IsNull() {
		state.WebhookEvents = types.ListNull(types.StringType)
	}

	if conn.Credentials != nil {
		if a, ok := conn.Credentials.Access[connection.PurposeConnection]; ok {
			state.ConnectionAudience = types.StringValue(a.Audience)
		}
		if a, ok := conn.Credentials.Access[connection.PurposeCheckout]; ok {
			state.CheckoutAudience = types.StringValue(a.Audience)
		}
		headers, diags := reconcileHeaderNames(ctx, state.RequestHeaders, conn.Credentials.RequestHeaderNames)
		resp.Diagnostics.Append(diags...)
		state.RequestHeaders = headers
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *forgejoConnectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state forgejoConnectionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.Id.ValueString()

	audiences := []struct {
		purpose     string
		plan, state types.String
	}{
		{connection.PurposeConnection, plan.ConnectionAudience, state.ConnectionAudience},
		{connection.PurposeCheckout, plan.CheckoutAudience, state.CheckoutAudience},
	}
	for _, a := range audiences {
		if a.plan.Equal(a.state) {
			continue
		}
		if err := r.client.SetOIDCAudience(ctx, id, a.purpose, a.plan.ValueString()); err != nil {
			resp.Diagnostics.AddError("Error updating the "+a.purpose+" audience of CircleCI Forgejo connection "+id, err.Error())
			return
		}
	}

	if !plan.RequestHeaders.Equal(state.RequestHeaders) {
		headers, diags := stringMap(ctx, plan.RequestHeaders)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if _, err := r.client.SetRequestHeaders(ctx, id, headers); err != nil {
			resp.Diagnostics.AddError("Error updating the request headers of CircleCI Forgejo connection "+id, err.Error())
			return
		}
	}

	plan.WebhookSecret = state.WebhookSecret
	if !plan.WebhookSecretRotation.Equal(state.WebhookSecretRotation) && !plan.WebhookSecretRotation.IsNull() {
		secret, err := r.client.RotateWebhookSecret(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("Error rotating the webhook secret of CircleCI Forgejo connection "+id, err.Error())
			return
		}
		plan.WebhookSecret = types.StringValue(secret)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *forgejoConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state forgejoConnectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, state.Id.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting CircleCI Forgejo connection "+state.Id.ValueString(), err.Error())
	}
}

func (r *forgejoConnectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.client = wrapper.ConnectionService
}

// ImportState accepts "<organization_id>/<connection_id>", because the API
// does not report which organization a connection belongs to.
func (r *forgejoConnectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	orgID, connID, ok := strings.Cut(req.ID, "/")
	if !ok || orgID == "" || connID == "" {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected an import ID of the form organization_id/connection_id, got %q.", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), connID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), orgID)...)
}

func stringMap(ctx context.Context, m types.Map) (map[string]string, diag.Diagnostics) {
	if m.IsNull() || m.IsUnknown() {
		return nil, nil
	}
	out := map[string]string{}
	diags := m.ElementsAs(ctx, &out, false)
	return out, diags
}

// reconcileHeaderNames rebuilds request_headers from the header names CircleCI
// reports. CircleCI canonicalises names and never returns values, so a known
// header keeps the name and value from state, and a header only CircleCI knows
// about is recorded with an empty value so it shows up as drift.
func reconcileHeaderNames(ctx context.Context, state types.Map, names []string) (types.Map, diag.Diagnostics) {
	known, diags := stringMap(ctx, state)
	if diags.HasError() {
		return state, diags
	}
	if len(names) == 0 {
		if state.IsNull() {
			return state, diags
		}
		return types.MapValueMust(types.StringType, map[string]attr.Value{}), diags
	}

	out := make(map[string]attr.Value, len(names))
	for _, name := range names {
		key, value := name, ""
		for k, v := range known {
			if strings.EqualFold(k, name) {
				key, value = k, v
				break
			}
		}
		out[key] = types.StringValue(value)
	}
	return types.MapValueMust(types.StringType, out), diags
}
