// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Codility/terraform-provider-openmetadata/internal/client"
)

var _ resource.Resource = &DomainResource{}
var _ resource.ResourceWithImportState = &DomainResource{}

const domainCollection = "domains"

type DomainResource struct {
	client *client.Client
}

type DomainResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	DisplayName types.String `tfsdk:"display_name"`
	Description types.String `tfsdk:"description"`
	DomainType  types.String `tfsdk:"domain_type"`
	Parent      types.String `tfsdk:"parent"`
	Owners      types.List   `tfsdk:"owners"`
	Experts     types.List   `tfsdk:"experts"`
	FQN         types.String `tfsdk:"fully_qualified_name"`
}

func NewDomainResource() resource.Resource {
	return &DomainResource{}
}

func (r *DomainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *DomainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an OpenMetadata Domain.",
		Attributes: map[string]schema.Attribute{
			"id":                   IDAttribute(),
			"name":                 NameAttribute(),
			"display_name":         DisplayNameAttribute(),
			"description":          DescriptionAttribute(true),
			"fully_qualified_name": FullyQualifiedNameAttribute(),
			"domain_type": schema.StringAttribute{
				Description: "Type of the domain. Must be one of: Source-aligned, Consumer-aligned, Aggregate.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"parent": schema.StringAttribute{
				Description: "Fully qualified name of the parent domain. When null, indicates a top-level domain.",
				Optional:    true,
			},
			"owners":  OwnersAttribute(),
			"experts": ExpertsAttribute(),
		},
	}
}

func (r *DomainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.buildBody(ctx, &plan)
	raw, err := r.client.CreateOrUpdate(ctx, domainCollection, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating domain", err.Error())
		return
	}

	r.readIntoState(raw, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	raw, err := r.client.GetByName(ctx, domainCollection, state.Name.ValueString(), []string{"owners", "experts", "parent"})
	if err != nil {
		resp.Diagnostics.AddError("Error reading domain", err.Error())
		return
	}
	if raw == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.readIntoState(raw, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.buildBody(ctx, &plan)
	raw, err := r.client.CreateOrUpdate(ctx, domainCollection, body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating domain", err.Error())
		return
	}

	r.readIntoState(raw, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting domain", map[string]interface{}{"name": state.Name.ValueString()})
	if err := r.client.Delete(ctx, domainCollection, state.ID.ValueString(), true); err != nil {
		resp.Diagnostics.AddError("Error deleting domain", err.Error())
	}
}

func (r *DomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	raw, err := r.client.GetByName(ctx, domainCollection, req.ID, []string{"owners", "experts", "parent"})
	if err != nil {
		resp.Diagnostics.AddError("Error importing domain", err.Error())
		return
	}
	if raw == nil {
		resp.Diagnostics.AddError("Domain not found", fmt.Sprintf("No domain with name %q", req.ID))
		return
	}
	var state DomainResourceModel
	r.readIntoState(raw, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DomainResource) buildBody(ctx context.Context, plan *DomainResourceModel) map[string]interface{} {
	body := map[string]interface{}{
		"name":        plan.Name.ValueString(),
		"description": plan.Description.ValueString(),
		"domainType":  plan.DomainType.ValueString(),
	}
	if !plan.DisplayName.IsNull() && !plan.DisplayName.IsUnknown() {
		body["displayName"] = plan.DisplayName.ValueString()
	}
	if !plan.Parent.IsNull() && !plan.Parent.IsUnknown() {
		body["parent"] = plan.Parent.ValueString()
	}
	if !plan.Owners.IsNull() && !plan.Owners.IsUnknown() {
		body["owners"] = extractOwnerRefs(ctx, plan.Owners)
	}
	if !plan.Experts.IsNull() && !plan.Experts.IsUnknown() {
		body["experts"] = extractOwnerRefs(ctx, plan.Experts)
	}
	return body
}

func (r *DomainResource) readIntoState(raw []byte, state *DomainResourceModel) {
	data, err := Unmarshal(raw)
	if err != nil {
		return
	}
	state.ID = StringVal(data, "id")
	state.Name = StringVal(data, "name")
	state.DisplayName = StringVal(data, "displayName")
	state.Description = StringVal(data, "description")
	state.DomainType = StringVal(data, "domainType")
	state.FQN = StringVal(data, "fullyQualifiedName")

	// parent is a single entity ref → extract FQN
	if p, ok := data["parent"].(map[string]interface{}); ok {
		if fqn, ok := p["fullyQualifiedName"].(string); ok {
			state.Parent = types.StringValue(fqn)
		} else if name, ok := p["name"].(string); ok {
			state.Parent = types.StringValue(name)
		} else {
			state.Parent = types.StringNull()
		}
	} else {
		state.Parent = types.StringNull()
	}

	state.Owners = OwnersListNull()
	state.Experts = OwnersListNull()
}
