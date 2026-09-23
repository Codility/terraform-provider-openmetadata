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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &TagResource{}
var _ resource.ResourceWithImportState = &TagResource{}

const tagCollection = "tags"

type TagResource struct {
	client *client.Client
}

type TagResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	DisplayName       types.String `tfsdk:"display_name"`
	Description       types.String `tfsdk:"description"`
	Classification    types.String `tfsdk:"classification"`
	Parent            types.String `tfsdk:"parent"`
	MutuallyExclusive types.Bool   `tfsdk:"mutually_exclusive"`
	Domains           types.List   `tfsdk:"domains"`
	Owners            types.List   `tfsdk:"owners"`
	FQN               types.String `tfsdk:"fully_qualified_name"`
}

func NewTagResource() resource.Resource {
	return &TagResource{}
}

func (r *TagResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag"
}

func (r *TagResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an OpenMetadata Tag within a Classification.",
		Attributes: map[string]schema.Attribute{
			"id":                   IDAttribute(),
			"name":                 NameAttribute(),
			"display_name":         DisplayNameAttribute(),
			"description":          DescriptionAttribute(true),
			"fully_qualified_name": FullyQualifiedNameAttribute(),
			"classification": schema.StringAttribute{
				Description: "Name of the classification this tag belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"parent": schema.StringAttribute{
				Description: "Fully qualified name of the parent tag (for nested tags).",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"mutually_exclusive": schema.BoolAttribute{
				Description: "When true, child tags are mutually exclusive.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"domains": DomainsAttribute(),
			"owners":  OwnersAttribute(),
		},
	}
}

func (r *TagResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TagResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.buildBody(ctx, &plan)
	raw, err := r.client.CreateOrUpdate(ctx, tagCollection, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating tag", err.Error())
		return
	}

	r.readIntoState(raw, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TagResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fqn := state.FQN.ValueString()
	if fqn == "" {
		// Build FQN from classification + name
		fqn = state.Classification.ValueString() + "." + state.Name.ValueString()
		if !state.Parent.IsNull() && state.Parent.ValueString() != "" {
			fqn = state.Parent.ValueString() + "." + state.Name.ValueString()
		}
	}

	raw, err := r.client.GetByName(ctx, tagCollection, fqn, []string{"owners", "domains"})
	if err != nil {
		resp.Diagnostics.AddError("Error reading tag", err.Error())
		return
	}
	if raw == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.readIntoState(raw, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TagResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.buildBody(ctx, &plan)
	raw, err := r.client.CreateOrUpdate(ctx, tagCollection, body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating tag", err.Error())
		return
	}

	r.readIntoState(raw, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TagResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting tag", map[string]interface{}{"fqn": state.FQN.ValueString()})
	if err := r.client.Delete(ctx, tagCollection, state.ID.ValueString(), true); err != nil {
		resp.Diagnostics.AddError("Error deleting tag", err.Error())
	}
}

func (r *TagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by FQN (e.g., "Classification.TagName")
	raw, err := r.client.GetByName(ctx, tagCollection, req.ID, []string{"owners", "domains"})
	if err != nil {
		resp.Diagnostics.AddError("Error importing tag", err.Error())
		return
	}
	if raw == nil {
		resp.Diagnostics.AddError("Tag not found", fmt.Sprintf("No tag with FQN %q", req.ID))
		return
	}
	var state TagResourceModel
	r.readIntoState(raw, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TagResource) buildBody(ctx context.Context, plan *TagResourceModel) map[string]interface{} {
	body := map[string]interface{}{
		"name":           plan.Name.ValueString(),
		"classification": plan.Classification.ValueString(),
		"description":    plan.Description.ValueString(),
	}
	if !plan.DisplayName.IsNull() && !plan.DisplayName.IsUnknown() {
		body["displayName"] = plan.DisplayName.ValueString()
	}
	if !plan.Parent.IsNull() && !plan.Parent.IsUnknown() {
		body["parent"] = plan.Parent.ValueString()
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

func (r *TagResource) readIntoState(raw []byte, state *TagResourceModel) {
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

	// Extract classification from FQN (first segment)
	if fqn := state.FQN.ValueString(); fqn != "" {
		parts := splitFQN(fqn)
		if len(parts) > 0 {
			state.Classification = types.StringValue(parts[0])
		}
	}
}
