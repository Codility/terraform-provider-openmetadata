// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Codility/terraform-provider-openmetadata/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &DashboardServiceResource{}
var _ resource.ResourceWithImportState = &DashboardServiceResource{}

const dashboardServiceCollection = "services/dashboardServices"

type DashboardServiceResource struct {
	client *client.Client
}

type DashboardServiceResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	DisplayName    types.String `tfsdk:"display_name"`
	Description    types.String `tfsdk:"description"`
	ServiceType    types.String `tfsdk:"service_type"`
	ConnectionJSON types.String `tfsdk:"connection_json"`
	Owners         types.List   `tfsdk:"owners"`
	Domains        types.List   `tfsdk:"domains"`
	HardDelete     types.Bool   `tfsdk:"hard_delete"`
	FQN            types.String `tfsdk:"fully_qualified_name"`
}

func NewDashboardServiceResource() resource.Resource {
	return &DashboardServiceResource{}
}

func (r *DashboardServiceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dashboard_service"
}

func (r *DashboardServiceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an OpenMetadata Dashboard Service. Updates are sent as a JSON Patch of the changed " +
			"fields, so a service whose connection is managed in the OpenMetadata UI keeps its connection and secrets.",
		Attributes: map[string]schema.Attribute{
			"id":                   IDAttribute(),
			"name":                 NameAttribute(),
			"display_name":         DisplayNameAttribute(),
			"description":          DescriptionAttribute(false),
			"fully_qualified_name": FullyQualifiedNameAttribute(),
			"service_type": schema.StringAttribute{
				Description: "Type of dashboard service (e.g., Tableau, Looker, Superset, PowerBI, Metabase, CustomDashboard).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"connection_json": schema.StringAttribute{
				Description: "Connection configuration as a JSON string; the structure depends on service_type. " +
					"Omit it to manage the connection in the OpenMetadata UI: the provider then never sends or changes it. " +
					"On create, an omitted connection defaults to {\"config\": {\"type\": <service_type>}}, which is " +
					"enough for CustomDashboard. Schemas: " +
					"https://github.com/open-metadata/OpenMetadata/tree/main/openmetadata-spec/src/main/resources/json/schema/entity/services/connections/dashboard",
				Optional:  true,
				Sensitive: true,
			},
			"owners": OwnersAttribute(),
			"domains": schema.ListAttribute{
				Description: "Fully qualified names of the domains this service belongs to. Omit to leave domains as set in OpenMetadata.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"hard_delete": schema.BoolAttribute{
				Description: "Hard-delete the service on destroy. Defaults to false: destroy soft-deletes, which can be " +
					"restored from the OpenMetadata UI. Deleting a service always removes its dashboards, charts and data models with it.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
		},
	}
}

func (r *DashboardServiceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DashboardServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DashboardServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// PUT is create-or-update: refuse to silently adopt (and overwrite) an existing service.
	existing, err := r.client.GetByName(ctx, dashboardServiceCollection, plan.Name.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Error checking for an existing dashboard service", err.Error())
		return
	}
	if existing != nil {
		resp.Diagnostics.AddError("Dashboard service already exists",
			fmt.Sprintf("A dashboard service named %q already exists. Import it instead of creating it.", plan.Name.ValueString()))
		return
	}

	body, err := r.buildBody(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error building request body", err.Error())
		return
	}

	raw, err := r.client.CreateOrUpdate(ctx, dashboardServiceCollection, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating dashboard service", err.Error())
		return
	}

	r.readIntoState(raw, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DashboardServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DashboardServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	raw, err := r.client.GetByName(ctx, dashboardServiceCollection, state.Name.ValueString(), []string{"owners", "domains"})
	if err != nil {
		resp.Diagnostics.AddError("Error reading dashboard service", err.Error())
		return
	}
	if raw == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.readIntoState(raw, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DashboardServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state DashboardServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A new connection needs the full entity, so it goes through PUT.
	if known(plan.ConnectionJSON) && !plan.ConnectionJSON.Equal(state.ConnectionJSON) {
		body, err := r.buildBody(ctx, &plan)
		if err != nil {
			resp.Diagnostics.AddError("Error building request body", err.Error())
			return
		}
		if _, err := r.client.CreateOrUpdate(ctx, dashboardServiceCollection, body); err != nil {
			resp.Diagnostics.AddError("Error updating dashboard service connection", err.Error())
			return
		}
	}

	// Everything else is patched, so a UI-managed connection is never sent back.
	ops, err := r.patchOps(ctx, &plan, &state)
	if err != nil {
		resp.Diagnostics.AddError("Error building patch", err.Error())
		return
	}
	if len(ops) > 0 {
		if _, err := r.client.Patch(ctx, dashboardServiceCollection, state.ID.ValueString(), ops); err != nil {
			resp.Diagnostics.AddError("Error updating dashboard service", err.Error())
			return
		}
	}

	raw, err := r.client.GetByName(ctx, dashboardServiceCollection, plan.Name.ValueString(), []string{"owners", "domains"})
	if err != nil || raw == nil {
		resp.Diagnostics.AddError("Error reading dashboard service after update", fmt.Sprintf("%v", err))
		return
	}
	r.readIntoState(raw, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DashboardServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DashboardServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hard := state.HardDelete.ValueBool()
	tflog.Info(ctx, "Deleting dashboard service", map[string]interface{}{"name": state.Name.ValueString(), "hard_delete": hard})
	if err := r.client.Delete(ctx, dashboardServiceCollection, state.ID.ValueString(), hard); err != nil {
		resp.Diagnostics.AddError("Error deleting dashboard service", err.Error())
	}
}

func (r *DashboardServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	raw, err := r.client.GetByName(ctx, dashboardServiceCollection, req.ID, []string{"owners", "domains"})
	if err != nil {
		resp.Diagnostics.AddError("Error importing dashboard service", err.Error())
		return
	}
	if raw == nil {
		resp.Diagnostics.AddError("Dashboard service not found", fmt.Sprintf("No dashboard service with name %q", req.ID))
		return
	}
	state := DashboardServiceResourceModel{
		Owners:         OwnersListNull(),
		ConnectionJSON: types.StringNull(),
		HardDelete:     types.BoolValue(false),
	}
	r.readIntoState(raw, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DashboardServiceResource) buildBody(ctx context.Context, plan *DashboardServiceResourceModel) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"name":        plan.Name.ValueString(),
		"serviceType": plan.ServiceType.ValueString(),
		// OM rejects a service PUT without a connection (NPE), so default to the bare type.
		"connection": map[string]interface{}{"config": map[string]interface{}{"type": plan.ServiceType.ValueString()}},
	}
	if known(plan.ConnectionJSON) {
		var conn interface{}
		if err := json.Unmarshal([]byte(plan.ConnectionJSON.ValueString()), &conn); err != nil {
			return nil, fmt.Errorf("invalid connection_json: %w", err)
		}
		body["connection"] = conn
	}
	if known(plan.DisplayName) {
		body["displayName"] = plan.DisplayName.ValueString()
	}
	if known(plan.Description) {
		body["description"] = plan.Description.ValueString()
	}
	if known(plan.Owners) {
		body["owners"] = extractOwnerRefs(ctx, plan.Owners)
	}
	if known(plan.Domains) {
		var vals []string
		plan.Domains.ElementsAs(ctx, &vals, false)
		body["domains"] = vals
	}
	return body, nil
}

// patchOps returns JSON Patch operations for the fields that differ between plan and state.
func (r *DashboardServiceResource) patchOps(ctx context.Context, plan, state *DashboardServiceResourceModel) ([]map[string]interface{}, error) {
	var ops []map[string]interface{}
	add := func(path string, value interface{}) {
		ops = append(ops, map[string]interface{}{"op": "add", "path": path, "value": value})
	}
	if known(plan.Description) && !plan.Description.Equal(state.Description) {
		add("/description", plan.Description.ValueString())
	}
	if known(plan.DisplayName) && !plan.DisplayName.Equal(state.DisplayName) {
		add("/displayName", plan.DisplayName.ValueString())
	}
	if known(plan.Owners) && !plan.Owners.Equal(state.Owners) {
		add("/owners", extractOwnerRefs(ctx, plan.Owners))
	}
	if known(plan.Domains) && !plan.Domains.Equal(state.Domains) {
		var names []string
		plan.Domains.ElementsAs(ctx, &names, false)
		refs := make([]EntityRef, 0, len(names))
		for _, n := range names {
			raw, err := r.client.GetByName(ctx, "domains", n, nil)
			if err != nil {
				return nil, fmt.Errorf("looking up domain %q: %w", n, err)
			}
			if raw == nil {
				return nil, fmt.Errorf("domain %q not found", n)
			}
			data, err := Unmarshal(raw)
			if err != nil {
				return nil, err
			}
			refs = append(refs, EntityRef{ID: StringVal(data, "id").ValueString(), Type: "domain"})
		}
		add("/domains", refs)
	}
	return ops, nil
}

// readIntoState copies server fields into state. Owners, connection_json and hard_delete
// keep their configured values: OM masks the connection, and owners are managed only when set.
func (r *DashboardServiceResource) readIntoState(raw []byte, state *DashboardServiceResourceModel) {
	data, err := Unmarshal(raw)
	if err != nil {
		return
	}
	state.ID = StringVal(data, "id")
	state.Name = StringVal(data, "name")
	state.DisplayName = StringVal(data, "displayName")
	state.Description = StringVal(data, "description")
	state.ServiceType = StringVal(data, "serviceType")
	state.FQN = StringVal(data, "fullyQualifiedName")
	state.Domains = StringListVal(data, "domains")
	if state.Owners.IsUnknown() {
		state.Owners = OwnersListNull()
	}
	if state.HardDelete.IsNull() || state.HardDelete.IsUnknown() {
		state.HardDelete = types.BoolValue(false)
	}
}

type knowable interface {
	IsNull() bool
	IsUnknown() bool
}

func known(v knowable) bool {
	return !v.IsNull() && !v.IsUnknown()
}
