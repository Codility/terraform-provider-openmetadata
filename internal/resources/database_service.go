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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &DatabaseServiceResource{}
var _ resource.ResourceWithImportState = &DatabaseServiceResource{}

const databaseServiceCollection = "services/databaseServices"

type DatabaseServiceResource struct {
	client *client.Client
}

type DatabaseServiceResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	DisplayName    types.String `tfsdk:"display_name"`
	Description    types.String `tfsdk:"description"`
	ServiceType    types.String `tfsdk:"service_type"`
	ConnectionJSON types.String `tfsdk:"connection_json"`
	Owners         types.List   `tfsdk:"owners"`
	Domains        types.List   `tfsdk:"domains"`
	FQN            types.String `tfsdk:"fully_qualified_name"`
}

func NewDatabaseServiceResource() resource.Resource {
	return &DatabaseServiceResource{}
}

func (r *DatabaseServiceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_service"
}

func (r *DatabaseServiceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an OpenMetadata Database Service.",
		Attributes: map[string]schema.Attribute{
			"id":                   IDAttribute(),
			"name":                 NameAttribute(),
			"display_name":         DisplayNameAttribute(),
			"description":          DescriptionAttribute(false),
			"fully_qualified_name": FullyQualifiedNameAttribute(),
			"service_type": schema.StringAttribute{
				Description: "Type of database service (e.g., Mysql, Postgres, BigQuery, Snowflake, Redshift, Athena, Trino, etc.).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"connection_json": schema.StringAttribute{
				Description: "Database connection configuration as a JSON string. The structure depends on the service_type. " +
					"See the JSON schema for each connector at: " +
					"https://github.com/open-metadata/OpenMetadata/tree/main/openmetadata-spec/src/main/resources/json/schema/entity/services/connections/database " +
					"You can also inspect your instance's Swagger UI at {host}/swagger.html under PUT /v1/services/databaseServices.",
				Optional:  true,
				Sensitive: true,
			},
			"owners":  OwnersAttribute(),
			"domains": DomainsAttribute(),
		},
	}
}

func (r *DatabaseServiceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DatabaseServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DatabaseServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := r.buildBody(ctx, &plan)
	if diags != nil {
		resp.Diagnostics.AddError("Error building request body", diags.Error())
		return
	}

	raw, err := r.client.CreateOrUpdate(ctx, databaseServiceCollection, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating database service", err.Error())
		return
	}

	r.readIntoState(raw, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DatabaseServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DatabaseServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	raw, err := r.client.GetByName(ctx, databaseServiceCollection, state.Name.ValueString(), []string{"owners", "domains"})
	if err != nil {
		resp.Diagnostics.AddError("Error reading database service", err.Error())
		return
	}
	if raw == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.readIntoState(raw, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DatabaseServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DatabaseServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := r.buildBody(ctx, &plan)
	if diags != nil {
		resp.Diagnostics.AddError("Error building request body", diags.Error())
		return
	}

	raw, err := r.client.CreateOrUpdate(ctx, databaseServiceCollection, body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating database service", err.Error())
		return
	}

	r.readIntoState(raw, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DatabaseServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DatabaseServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting database service", map[string]interface{}{"name": state.Name.ValueString()})
	if err := r.client.Delete(ctx, databaseServiceCollection, state.ID.ValueString(), true); err != nil {
		resp.Diagnostics.AddError("Error deleting database service", err.Error())
	}
}

func (r *DatabaseServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	raw, err := r.client.GetByName(ctx, databaseServiceCollection, req.ID, []string{"owners", "domains"})
	if err != nil {
		resp.Diagnostics.AddError("Error importing database service", err.Error())
		return
	}
	if raw == nil {
		resp.Diagnostics.AddError("Database service not found", fmt.Sprintf("No database service with name %q", req.ID))
		return
	}
	var state DatabaseServiceResourceModel
	r.readIntoState(raw, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DatabaseServiceResource) buildBody(ctx context.Context, plan *DatabaseServiceResourceModel) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"name":        plan.Name.ValueString(),
		"serviceType": plan.ServiceType.ValueString(),
	}

	if !plan.DisplayName.IsNull() && !plan.DisplayName.IsUnknown() {
		body["displayName"] = plan.DisplayName.ValueString()
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		body["description"] = plan.Description.ValueString()
	}
	if !plan.ConnectionJSON.IsNull() && !plan.ConnectionJSON.IsUnknown() {
		var conn interface{}
		if err := json.Unmarshal([]byte(plan.ConnectionJSON.ValueString()), &conn); err != nil {
			return nil, fmt.Errorf("invalid connection_json: %w", err)
		}
		body["connection"] = conn
	}
	if !plan.Owners.IsNull() && !plan.Owners.IsUnknown() {
		body["owners"] = extractOwnerRefs(ctx, plan.Owners)
	}
	if !plan.Domains.IsNull() && !plan.Domains.IsUnknown() {
		var vals []string
		plan.Domains.ElementsAs(ctx, &vals, false)
		body["domains"] = vals
	}

	return body, nil
}

func (r *DatabaseServiceResource) readIntoState(raw []byte, state *DatabaseServiceResourceModel) {
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
	state.Owners = OwnersListNull()
	// Note: connection is not read back — it's sensitive and OM may mask fields.
	// The user-provided connection_json is preserved in state as-is.
}
