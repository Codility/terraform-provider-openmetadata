// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"fmt"

	"github.com/Codility/terraform-provider-openmetadata/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &TeamResource{}
var _ resource.ResourceWithImportState = &TeamResource{}

const teamCollection = "teams"

type TeamResource struct {
	client *client.Client
}

type TeamResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	DisplayName types.String `tfsdk:"display_name"`
	Description types.String `tfsdk:"description"`
	TeamType    types.String `tfsdk:"team_type"`
	Email       types.String `tfsdk:"email"`
	IsJoinable  types.Bool   `tfsdk:"is_joinable"`
	Parents     types.List   `tfsdk:"parents"`
	Policies    types.List   `tfsdk:"policies"`
	Domains     types.List   `tfsdk:"domains"`
	Owners      types.List   `tfsdk:"owners"`
	FQN         types.String `tfsdk:"fully_qualified_name"`
}

func NewTeamResource() resource.Resource {
	return &TeamResource{}
}

func (r *TeamResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}

func (r *TeamResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an OpenMetadata Team.",
		Attributes: map[string]schema.Attribute{
			"id":                   IDAttribute(),
			"name":                 NameAttribute(),
			"display_name":         DisplayNameAttribute(),
			"description":          DescriptionAttribute(false),
			"fully_qualified_name": FullyQualifiedNameAttribute(),
			"team_type": schema.StringAttribute{
				Description: "Type of team: Group, Department, Division, BusinessUnit, or Organization.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("Group"),
				Validators: []validator.String{
					stringvalidator.OneOf("Group", "Department", "Division", "BusinessUnit", "Organization"),
				},
			},
			"email": schema.StringAttribute{
				Description: "Email address of the team.",
				Optional:    true,
			},
			"is_joinable": schema.BoolAttribute{
				Description: "Whether any user can join this team during sign up.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"parents": schema.ListAttribute{
				Description: "Names or fully qualified names of parent teams. When omitted, OpenMetadata automatically places the team under the root Organisation team; this default is not reflected in state.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"policies": schema.ListAttribute{
				Description: "Names or fully qualified names of policies attached to this team.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"domains": DomainsAttribute(),
			"owners":  OwnersAttribute(),
		},
	}
}

func (r *TeamResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *TeamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TeamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.buildCreateBody(ctx, &plan)

	raw, err := r.client.CreateOrUpdate(ctx, teamCollection, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating team", err.Error())
		return
	}

	r.readIntoState(ctx, raw, &plan, resp)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TeamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TeamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	raw, err := r.client.GetByName(ctx, teamCollection, state.Name.ValueString(), []string{"owners", "parents", "policies", "domains"})
	if err != nil {
		resp.Diagnostics.AddError("Error reading team", err.Error())
		return
	}
	if raw == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.readIntoState(ctx, raw, &state, resp)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TeamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TeamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.buildCreateBody(ctx, &plan)

	raw, err := r.client.CreateOrUpdate(ctx, teamCollection, body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating team", err.Error())
		return
	}

	r.readIntoState(ctx, raw, &plan, resp)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TeamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TeamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting team", map[string]interface{}{"name": state.Name.ValueString()})

	if err := r.client.Delete(ctx, teamCollection, state.ID.ValueString(), true); err != nil {
		resp.Diagnostics.AddError("Error deleting team", err.Error())
	}
}

func (r *TeamResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by FQN (name)
	raw, err := r.client.GetByName(ctx, teamCollection, req.ID, []string{"owners", "parents", "policies", "domains"})
	if err != nil {
		resp.Diagnostics.AddError("Error importing team", err.Error())
		return
	}
	if raw == nil {
		resp.Diagnostics.AddError("Team not found", fmt.Sprintf("No team with name %q found", req.ID))
		return
	}

	var state TeamResourceModel
	r.readIntoState(ctx, raw, &state, resp)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// --- internal helpers ---

func (r *TeamResource) buildCreateBody(ctx context.Context, plan *TeamResourceModel) map[string]interface{} {
	body := map[string]interface{}{
		"name":     plan.Name.ValueString(),
		"teamType": plan.TeamType.ValueString(),
	}
	if !plan.DisplayName.IsNull() && !plan.DisplayName.IsUnknown() {
		body["displayName"] = plan.DisplayName.ValueString()
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		body["description"] = plan.Description.ValueString()
	}
	if !plan.Email.IsNull() && !plan.Email.IsUnknown() {
		body["email"] = plan.Email.ValueString()
	}
	if !plan.IsJoinable.IsNull() && !plan.IsJoinable.IsUnknown() {
		body["isJoinable"] = plan.IsJoinable.ValueBool()
	}
	if !plan.Parents.IsNull() && !plan.Parents.IsUnknown() {
		var parents []string
		plan.Parents.ElementsAs(ctx, &parents, false)
		body["parents"] = parents
	}
	if !plan.Policies.IsNull() && !plan.Policies.IsUnknown() {
		var policies []string
		plan.Policies.ElementsAs(ctx, &policies, false)
		body["policies"] = policies
	}
	if !plan.Domains.IsNull() && !plan.Domains.IsUnknown() {
		var domains []string
		plan.Domains.ElementsAs(ctx, &domains, false)
		body["domains"] = domains
	}
	if !plan.Owners.IsNull() && !plan.Owners.IsUnknown() {
		body["owners"] = r.extractOwners(ctx, plan)
	}
	return body
}

func (r *TeamResource) extractOwners(ctx context.Context, plan *TeamResourceModel) []EntityRef {
	type ownerModel struct {
		ID   types.String `tfsdk:"id"`
		Type types.String `tfsdk:"type"`
	}
	var owners []ownerModel
	plan.Owners.ElementsAs(ctx, &owners, false)
	refs := make([]EntityRef, len(owners))
	for i, o := range owners {
		refs[i] = EntityRef{ID: o.ID.ValueString(), Type: o.Type.ValueString()}
	}
	return refs
}

func (r *TeamResource) readIntoState(ctx context.Context, raw []byte, state *TeamResourceModel, resp interface{}) {
	data, err := Unmarshal(raw)
	if err != nil {
		addDiagError(resp, "Error parsing team response", err.Error())
		return
	}

	state.ID = StringVal(data, "id")
	state.Name = StringVal(data, "name")
	state.DisplayName = StringVal(data, "displayName")
	state.Description = StringVal(data, "description")
	state.TeamType = StringVal(data, "teamType")
	state.Email = StringVal(data, "email")
	state.IsJoinable = BoolVal(data, "isJoinable")
	state.FQN = StringVal(data, "fullyQualifiedName")
	// parents is intentionally NOT read from the API response. OM always places
	// teams under Organisation by default, which would override the null state
	// and cause "Provider produced inconsistent result after apply". Parents
	// provided by the user are sent on create/update but not reflected back.
	// For import, parents is excluded via ImportStateVerifyIgnore.
	state.Parents = types.ListNull(types.StringType)
	state.Policies = StringListVal(data, "policies")
	state.Domains = StringListVal(data, "domains")
	state.Owners = OwnersListNull()
}

// addDiagError is a helper that works with both Create/Update/Read response types.
func addDiagError(resp interface{}, summary, detail string) {
	switch r := resp.(type) {
	case *resource.CreateResponse:
		r.Diagnostics.AddError(summary, detail)
	case *resource.ReadResponse:
		r.Diagnostics.AddError(summary, detail)
	case *resource.UpdateResponse:
		r.Diagnostics.AddError(summary, detail)
	case *resource.ImportStateResponse:
		r.Diagnostics.AddError(summary, detail)
	}
}
