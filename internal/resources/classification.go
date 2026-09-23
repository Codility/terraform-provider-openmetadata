// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"fmt"

	"github.com/Codility/terraform-provider-openmetadata/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &ClassificationResource{}
var _ resource.ResourceWithImportState = &ClassificationResource{}

const classificationCollection = "classifications"

type ClassificationResource struct {
	client *client.Client
}

type ClassificationResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	DisplayName       types.String `tfsdk:"display_name"`
	Description       types.String `tfsdk:"description"`
	MutuallyExclusive types.Bool   `tfsdk:"mutually_exclusive"`
	Domains           types.List   `tfsdk:"domains"`
	Owners            types.List   `tfsdk:"owners"`
	FQN               types.String `tfsdk:"fully_qualified_name"`
}

func NewClassificationResource() resource.Resource {
	return &ClassificationResource{}
}

func (r *ClassificationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_classification"
}

func (r *ClassificationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an OpenMetadata Classification (tag category).",
		Attributes: map[string]schema.Attribute{
			"id":                   IDAttribute(),
			"name":                 NameAttribute(),
			"display_name":         DisplayNameAttribute(),
			"description":          DescriptionAttribute(true),
			"fully_qualified_name": FullyQualifiedNameAttribute(),
			"mutually_exclusive": schema.BoolAttribute{
				Description: "When true, tags in this classification are mutually exclusive (entity can have only one). When false, multiple tags can coexist. Changing this value forces a new resource.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"domains": DomainsAttribute(),
			"owners":  OwnersAttribute(),
		},
	}
}

func (r *ClassificationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ClassificationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ClassificationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.buildBody(ctx, &plan)
	raw, err := r.client.CreateOrUpdate(ctx, classificationCollection, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating classification", err.Error())
		return
	}

	r.readIntoState(raw, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ClassificationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ClassificationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	raw, err := r.client.GetByName(ctx, classificationCollection, state.Name.ValueString(), []string{"owners", "domains"})
	if err != nil {
		resp.Diagnostics.AddError("Error reading classification", err.Error())
		return
	}
	if raw == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.readIntoState(raw, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ClassificationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ClassificationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.buildBody(ctx, &plan)
	raw, err := r.client.CreateOrUpdate(ctx, classificationCollection, body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating classification", err.Error())
		return
	}

	r.readIntoState(raw, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ClassificationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ClassificationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting classification", map[string]interface{}{"name": state.Name.ValueString()})
	if err := r.client.Delete(ctx, classificationCollection, state.ID.ValueString(), true); err != nil {
		resp.Diagnostics.AddError("Error deleting classification", err.Error())
	}
}

func (r *ClassificationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	raw, err := r.client.GetByName(ctx, classificationCollection, req.ID, []string{"owners", "domains"})
	if err != nil {
		resp.Diagnostics.AddError("Error importing classification", err.Error())
		return
	}
	if raw == nil {
		resp.Diagnostics.AddError("Classification not found", fmt.Sprintf("No classification with name %q", req.ID))
		return
	}
	var state ClassificationResourceModel
	r.readIntoState(raw, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ClassificationResource) buildBody(ctx context.Context, plan *ClassificationResourceModel) map[string]interface{} {
	body := map[string]interface{}{
		"name":        plan.Name.ValueString(),
		"description": plan.Description.ValueString(),
	}
	if !plan.DisplayName.IsNull() && !plan.DisplayName.IsUnknown() {
		body["displayName"] = plan.DisplayName.ValueString()
	}
	if !plan.MutuallyExclusive.IsNull() && !plan.MutuallyExclusive.IsUnknown() {
		body["mutuallyExclusive"] = plan.MutuallyExclusive.ValueBool()
	}
	if !plan.Domains.IsNull() && !plan.Domains.IsUnknown() {
		var domains []string
		plan.Domains.ElementsAs(ctx, &domains, false)
		body["domains"] = domains
	}
	if !plan.Owners.IsNull() && !plan.Owners.IsUnknown() {
		body["owners"] = extractOwnerRefs(ctx, plan.Owners)
	}
	return body
}

func (r *ClassificationResource) readIntoState(raw []byte, state *ClassificationResourceModel) {
	data, err := Unmarshal(raw)
	if err != nil {
		return
	}
	state.ID = StringVal(data, "id")
	state.Name = StringVal(data, "name")
	state.DisplayName = StringVal(data, "displayName")
	state.Description = StringVal(data, "description")
	state.MutuallyExclusive = BoolVal(data, "mutuallyExclusive")
	state.FQN = StringVal(data, "fullyQualifiedName")
	state.Domains = StringListVal(data, "domains")
	state.Owners = OwnersListNull()
}

// extractOwnerRefs is a shared helper for converting owners list to EntityRef slice.
func extractOwnerRefs(ctx context.Context, ownersList types.List) []EntityRef {
	type ownerModel struct {
		ID   types.String `tfsdk:"id"`
		Type types.String `tfsdk:"type"`
	}
	var owners []ownerModel
	ownersList.ElementsAs(ctx, &owners, false)
	refs := make([]EntityRef, len(owners))
	for i, o := range owners {
		refs[i] = EntityRef{ID: o.ID.ValueString(), Type: o.Type.ValueString()}
	}
	return refs
}
