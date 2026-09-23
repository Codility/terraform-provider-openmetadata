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

var _ resource.Resource = &GlossaryTermResource{}
var _ resource.ResourceWithImportState = &GlossaryTermResource{}

const glossaryTermCollection = "glossaryTerms"

type GlossaryTermResource struct {
	client *client.Client
}

type GlossaryTermResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	DisplayName       types.String `tfsdk:"display_name"`
	Description       types.String `tfsdk:"description"`
	Glossary          types.String `tfsdk:"glossary"`
	Parent            types.String `tfsdk:"parent"`
	Synonyms          types.List   `tfsdk:"synonyms"`
	RelatedTerms      types.List   `tfsdk:"related_terms"`
	MutuallyExclusive types.Bool   `tfsdk:"mutually_exclusive"`
	Domains           types.List   `tfsdk:"domains"`
	Owners            types.List   `tfsdk:"owners"`
	FQN               types.String `tfsdk:"fully_qualified_name"`
}

func NewGlossaryTermResource() resource.Resource {
	return &GlossaryTermResource{}
}

func (r *GlossaryTermResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_glossary_term"
}

func (r *GlossaryTermResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an OpenMetadata Glossary Term.",
		Attributes: map[string]schema.Attribute{
			"id":                   IDAttribute(),
			"name":                 NameAttribute(),
			"display_name":         DisplayNameAttribute(),
			"description":          DescriptionAttribute(true),
			"fully_qualified_name": FullyQualifiedNameAttribute(),
			"glossary": schema.StringAttribute{
				Description: "Fully qualified name of the glossary this term belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"parent": schema.StringAttribute{
				Description: "Fully qualified name of the parent glossary term (for nested terms).",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"synonyms": schema.ListAttribute{
				Description: "Alternate names that are synonyms for this term.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"related_terms": schema.ListAttribute{
				Description: "Fully qualified names of related glossary terms.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"mutually_exclusive": schema.BoolAttribute{
				Description: "When true, child terms are mutually exclusive.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"domains": DomainsAttribute(),
			"owners":  OwnersAttribute(),
		},
	}
}

func (r *GlossaryTermResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GlossaryTermResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GlossaryTermResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.buildBody(ctx, &plan)
	raw, err := r.client.CreateOrUpdate(ctx, glossaryTermCollection, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating glossary term", err.Error())
		return
	}

	r.readIntoState(raw, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GlossaryTermResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GlossaryTermResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fqn := state.FQN.ValueString()
	if fqn == "" {
		// Build FQN: glossary.name or parent.name
		if !state.Parent.IsNull() && state.Parent.ValueString() != "" {
			fqn = state.Parent.ValueString() + "." + state.Name.ValueString()
		} else {
			fqn = state.Glossary.ValueString() + "." + state.Name.ValueString()
		}
	}

	raw, err := r.client.GetByName(ctx, glossaryTermCollection, fqn, []string{"owners", "domains", "relatedTerms"})
	if err != nil {
		resp.Diagnostics.AddError("Error reading glossary term", err.Error())
		return
	}
	if raw == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.readIntoState(raw, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GlossaryTermResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan GlossaryTermResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.buildBody(ctx, &plan)
	raw, err := r.client.CreateOrUpdate(ctx, glossaryTermCollection, body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating glossary term", err.Error())
		return
	}

	r.readIntoState(raw, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GlossaryTermResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GlossaryTermResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting glossary term", map[string]interface{}{"fqn": state.FQN.ValueString()})
	if err := r.client.Delete(ctx, glossaryTermCollection, state.ID.ValueString(), true); err != nil {
		resp.Diagnostics.AddError("Error deleting glossary term", err.Error())
	}
}

func (r *GlossaryTermResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by FQN (e.g., "Glossary.TermName" or "Glossary.Parent.TermName")
	raw, err := r.client.GetByName(ctx, glossaryTermCollection, req.ID, []string{"owners", "domains", "relatedTerms"})
	if err != nil {
		resp.Diagnostics.AddError("Error importing glossary term", err.Error())
		return
	}
	if raw == nil {
		resp.Diagnostics.AddError("Glossary term not found", fmt.Sprintf("No glossary term with FQN %q", req.ID))
		return
	}
	var state GlossaryTermResourceModel
	r.readIntoState(raw, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GlossaryTermResource) buildBody(ctx context.Context, plan *GlossaryTermResourceModel) map[string]interface{} {
	body := map[string]interface{}{
		"glossary":    plan.Glossary.ValueString(),
		"name":        plan.Name.ValueString(),
		"description": plan.Description.ValueString(),
	}
	if !plan.DisplayName.IsNull() && !plan.DisplayName.IsUnknown() {
		body["displayName"] = plan.DisplayName.ValueString()
	}
	if !plan.Parent.IsNull() && !plan.Parent.IsUnknown() {
		body["parent"] = plan.Parent.ValueString()
	}
	if !plan.Synonyms.IsNull() && !plan.Synonyms.IsUnknown() {
		var synonyms []string
		plan.Synonyms.ElementsAs(ctx, &synonyms, false)
		body["synonyms"] = synonyms
	}
	if !plan.RelatedTerms.IsNull() && !plan.RelatedTerms.IsUnknown() {
		var related []string
		plan.RelatedTerms.ElementsAs(ctx, &related, false)
		body["relatedTerms"] = related
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

func (r *GlossaryTermResource) readIntoState(raw []byte, state *GlossaryTermResourceModel) {
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

	// Extract glossary from the glossary field in the response
	if g, ok := data["glossary"]; ok && g != nil {
		if gMap, ok := g.(map[string]interface{}); ok {
			if fqn, ok := gMap["fullyQualifiedName"].(string); ok {
				state.Glossary = types.StringValue(fqn)
			} else if name, ok := gMap["name"].(string); ok {
				state.Glossary = types.StringValue(name)
			}
		}
	}

	state.Synonyms = StringSliceToList(RawStringList(data, "synonyms"))
	state.RelatedTerms = StringListVal(data, "relatedTerms")
	state.Domains = StringListVal(data, "domains")
	state.Owners = OwnersListNull()
}
