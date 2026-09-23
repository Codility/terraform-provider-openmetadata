# Resource Template

Every generated resource file follows this exact structure. Adapt the placeholders to the target entity.

## File Structure

```go
// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package resources

import (
    "context"
    "fmt"

    // Add only imports the resource actually needs:
    // "encoding/json"  ← if handling raw JSON fields
    // "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"  ← if enum validation
    // "github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"  ← if bool defaults
    // "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"  ← if RequiresReplace on custom fields
    // "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"  ← if RequiresReplace
    // "github.com/hashicorp/terraform-plugin-framework/schema/validator"  ← if validators

    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema"
    "github.com/hashicorp/terraform-plugin-framework/types"
    "github.com/hashicorp/terraform-plugin-log/tflog"
    "github.com/Codility/terraform-provider-openmetadata/internal/client"
)

// Interface compliance
var _ resource.Resource = &<Entity>Resource{}
var _ resource.ResourceWithImportState = &<Entity>Resource{}

const <entity>Collection = "<api_collection>"

type <Entity>Resource struct {
    client *client.Client
}

type <Entity>ResourceModel struct {
    ID          types.String `tfsdk:"id"`
    Name        types.String `tfsdk:"name"`
    DisplayName types.String `tfsdk:"display_name"`
    Description types.String `tfsdk:"description"`
    // ... entity-specific fields ...
    Owners      types.List   `tfsdk:"owners"`    // if applicable
    Domains     types.List   `tfsdk:"domains"`   // if applicable
    FQN         types.String `tfsdk:"fully_qualified_name"`
}

func New<Entity>Resource() resource.Resource {
    return &<Entity>Resource{}
}

func (r *<Entity>Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_<snake_entity>"
}

func (r *<Entity>Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Description: "Manages an OpenMetadata <Display Name>.",
        Attributes: map[string]schema.Attribute{
            "id":                   IDAttribute(),
            "name":                 NameAttribute(),
            "display_name":         DisplayNameAttribute(),
            "description":          DescriptionAttribute(<true_or_false>),
            // ... entity-specific attributes ...
            "owners":               OwnersAttribute(),     // if applicable
            "domains":              DomainsAttribute(),    // if applicable
            "fully_qualified_name": FullyQualifiedNameAttribute(),
        },
    }
}

func (r *<Entity>Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    c, ok := req.ProviderData.(*client.Client)
    if !ok {
        resp.Diagnostics.AddError("Unexpected provider data type",
            fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
        return
    }
    r.client = c
}

func (r *<Entity>Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan <Entity>ResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    if resp.Diagnostics.HasError() {
        return
    }

    body := r.buildBody(ctx, &plan)
    raw, err := r.client.CreateOrUpdate(ctx, <entity>Collection, body)
    if err != nil {
        resp.Diagnostics.AddError("Error creating <entity>", err.Error())
        return
    }

    r.readIntoState(raw, &plan)
    resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *<Entity>Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state <Entity>ResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    if resp.Diagnostics.HasError() {
        return
    }

    raw, err := r.client.GetByName(ctx, <entity>Collection, state.Name.ValueString(),
        []string{<read_fields>})
    if err != nil {
        resp.Diagnostics.AddError("Error reading <entity>", err.Error())
        return
    }
    if raw == nil {
        resp.State.RemoveResource(ctx)
        return
    }

    r.readIntoState(raw, &state)
    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *<Entity>Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan <Entity>ResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    if resp.Diagnostics.HasError() {
        return
    }

    body := r.buildBody(ctx, &plan)
    raw, err := r.client.CreateOrUpdate(ctx, <entity>Collection, body)
    if err != nil {
        resp.Diagnostics.AddError("Error updating <entity>", err.Error())
        return
    }

    r.readIntoState(raw, &plan)
    resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *<Entity>Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state <Entity>ResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    if resp.Diagnostics.HasError() {
        return
    }

    tflog.Info(ctx, "Deleting <entity>", map[string]interface{}{"name": state.Name.ValueString()})
    if err := r.client.Delete(ctx, <entity>Collection, state.ID.ValueString(), true); err != nil {
        resp.Diagnostics.AddError("Error deleting <entity>", err.Error())
    }
}

func (r *<Entity>Resource) ImportState(ctx context.Context, req resource.ImportStateRequest,
    resp *resource.ImportStateResponse) {

    raw, err := r.client.GetByName(ctx, <entity>Collection, req.ID,
        []string{<read_fields>})
    if err != nil {
        resp.Diagnostics.AddError("Error importing <entity>", err.Error())
        return
    }
    if raw == nil {
        resp.Diagnostics.AddError("<Entity> not found",
            fmt.Sprintf("No <entity> with name %q", req.ID))
        return
    }
    var state <Entity>ResourceModel
    r.readIntoState(raw, &state)
    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// --- internal helpers ---

func (r *<Entity>Resource) buildBody(ctx context.Context, plan *<Entity>ResourceModel) map[string]interface{} {
    body := map[string]interface{}{
        "name": plan.Name.ValueString(),
        // Include other required fields here
    }

    // Optional string fields
    if !plan.DisplayName.IsNull() && !plan.DisplayName.IsUnknown() {
        body["displayName"] = plan.DisplayName.ValueString()
    }
    if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
        body["description"] = plan.Description.ValueString()
    }

    // Optional bool fields
    // if !plan.SomeBool.IsNull() && !plan.SomeBool.IsUnknown() {
    //     body["someBool"] = plan.SomeBool.ValueBool()
    // }

    // Optional string list fields (entity refs sent as plain names)
    // if !plan.Policies.IsNull() && !plan.Policies.IsUnknown() {
    //     var vals []string
    //     plan.Policies.ElementsAs(ctx, &vals, false)
    //     body["policies"] = vals
    // }

    // Owners (nested entity refs)
    if !plan.Owners.IsNull() && !plan.Owners.IsUnknown() {
        body["owners"] = extractOwnerRefs(ctx, plan.Owners)
    }

    // Domains (string list)
    if !plan.Domains.IsNull() && !plan.Domains.IsUnknown() {
        var vals []string
        plan.Domains.ElementsAs(ctx, &vals, false)
        body["domains"] = vals
    }

    return body
}

func (r *<Entity>Resource) readIntoState(raw []byte, state *<Entity>ResourceModel) {
    data, err := Unmarshal(raw)
    if err != nil {
        return
    }

    state.ID = StringVal(data, "id")
    state.Name = StringVal(data, "name")
    state.DisplayName = StringVal(data, "displayName")
    state.Description = StringVal(data, "description")

    // String fields
    // state.SomeField = StringVal(data, "someField")

    // Bool fields
    // state.SomeBool = BoolVal(data, "someBool")

    // Entity ref list fields (OM returns objects, extract names)
    // if names := EntityRefNames(data, "policies"); names != nil {
    //     state.Policies, _ = types.ListValueFrom(context.Background(), types.StringType, names)
    // }

    // Plain string list fields
    // if vals := StringListVal(data, "tags"); vals != nil {
    //     state.Tags, _ = types.ListValueFrom(context.Background(), types.StringType, vals)
    // }

    state.FQN = StringVal(data, "fullyQualifiedName")
}
```

## Key Rules

1. **`buildBody`** always returns `map[string]interface{}` — OM's PUT API is idempotent create-or-update
2. **`readIntoState`** parses the JSON response and sets typed values on the model
3. **Create and Update** use the exact same `buildBody` + `CreateOrUpdate` pattern (OM PUT is idempotent)
4. **Read** uses `GetByName` with `?fields=` for expandable fields
5. **Delete** uses `GetByID` with `hardDelete=true`
6. **Import** uses `GetByName` with the FQN as the import ID
7. The `extractOwnerRefs` helper in `common.go` handles the owners nested list pattern
8. Never use `log.Fatal`, `panic`, or `os.Exit` — always `resp.Diagnostics.AddError`
