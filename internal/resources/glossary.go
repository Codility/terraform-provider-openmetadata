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

var _ resource.Resource = &GlossaryResource{}
var _ resource.ResourceWithImportState = &GlossaryResource{}

const glossaryCollection = "glossaries"

type GlossaryResource struct {
	client *client.Client
}

type GlossaryResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	DisplayName       types.String `tfsdk:"display_name"`
	Description       types.String `tfsdk:"description"`
	MutuallyExclusive types.Bool   `tfsdk:"mutually_exclusive"`
	Domains           types.List   `tfsdk:"domains"`
	Owners            types.List   `tfsdk:"owners"`
	FQN               types.String `tfsdk:"fully_qualified_name"`
}

func NewGlossaryResource() resource.Resource {
	return &GlossaryResource{}
}

func (r *GlossaryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_glossary"
}

func (r *GlossaryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an OpenMetadata Glossary.",
		Attributes: map[string]schema.Attribute{
			"id":                   IDAttribute(),
			"name":                 NameAttribute(),
			"display_name":         DisplayNameAttribute(),
			"description":          DescriptionAttribute(true),
			"fully_qualified_name": FullyQualifiedNameAttribute(),
			"mutually_exclusive": schema.BoolAttribute{
				Description: "When true, glossary terms that are direct children are mutually exclusive. Changing this value forces a new resource.",
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

func (r *GlossaryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GlossaryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GlossaryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.buildBody(ctx, &plan)
	raw, err := r.client.CreateOrUpdate(ctx, glossaryCollection, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating glossary", err.Error())
		return
	}

	r.readIntoState(raw, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GlossaryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GlossaryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	raw, err := r.client.GetByName(ctx, glossaryCollection, state.Name.ValueString(), []string{"owners", "domains"})
	if err != nil {
		resp.Diagnostics.AddError("Error reading glossary", err.Error())
		return
	}
	if raw == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.readIntoState(raw, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GlossaryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan GlossaryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.buildBody(ctx, &plan)
	raw, err := r.client.CreateOrUpdate(ctx, glossaryCollection, body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating glossary", err.Error())
		return
	}

	r.readIntoState(raw, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GlossaryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GlossaryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting glossary", map[string]interface{}{"name": state.Name.ValueString()})
	if err := r.client.Delete(ctx, glossaryCollection, state.ID.ValueString(), true); err != nil {
		resp.Diagnostics.AddError("Error deleting glossary", err.Error())
	}
}

func (r *GlossaryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	raw, err := r.client.GetByName(ctx, glossaryCollection, req.ID, []string{"owners", "domains"})
	if err != nil {
		resp.Diagnostics.AddError("Error importing glossary", err.Error())
		return
	}
	if raw == nil {
		resp.Diagnostics.AddError("Glossary not found", fmt.Sprintf("No glossary with name %q", req.ID))
		return
	}
	var state GlossaryResourceModel
	r.readIntoState(raw, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GlossaryResource) buildBody(ctx context.Context, plan *GlossaryResourceModel) map[string]interface{} {
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

func (r *GlossaryResource) readIntoState(raw []byte, state *GlossaryResourceModel) {
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
