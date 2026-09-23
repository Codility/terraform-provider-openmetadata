# Field Mapping Rules

How to map OpenMetadata JSON Schema properties to Terraform resource attributes.

## Standard Fields (Always Present)

These fields exist on virtually every OM entity and are handled via `common.go` helpers:

| OM JSON field | TF attribute | Model type | Schema helper | Notes |
|---|---|---|---|---|
| `id` (UUID) | `id` | `types.String` | `IDAttribute()` | Computed, plan-modifier: UseStateForUnknown |
| `name` | `name` | `types.String` | `NameAttribute()` | Required, RequiresReplace |
| `displayName` | `display_name` | `types.String` | `DisplayNameAttribute()` | Optional+Computed |
| `description` | `description` | `types.String` | `DescriptionAttribute(required)` | Check if in `required[]` |
| `fullyQualifiedName` | `fully_qualified_name` | `types.String` | `FullyQualifiedNameAttribute()` | Computed |
| `owners` (entityReferenceList) | `owners` | `types.List` | `OwnersAttribute()` | Nested: {id, type} |
| `domains` | `domains` | `types.List` | `DomainsAttribute()` | List of FQN strings |

**Do NOT manually define these** — always use the `common.go` helpers.

## Type Mapping

| JSON Schema type | `$ref` hint | Go model type | TF schema type | Read helper |
|---|---|---|---|---|
| `"type": "string"` | — | `types.String` | `schema.StringAttribute{}` | `StringVal(data, "key")` |
| `"type": "string"` + `"enum"` | — | `types.String` | `schema.StringAttribute{}` + Validator | `StringVal(data, "key")` |
| `"type": "boolean"` | — | `types.Bool` | `schema.BoolAttribute{}` | `BoolVal(data, "key")` |
| `"type": "array"` + items=string | — | `types.List` | `schema.ListAttribute{ElementType: types.StringType}` | `StringListVal(data, "key")` |
| `"type": "array"` + items=$ref entityRef | — | `types.List` | `schema.ListAttribute{ElementType: types.StringType}` | `EntityRefNames(data, "key")` |
| `$ref` → entityReferenceList | `entityReferenceList` | `types.List` | `OwnersAttribute()` | `ParseEntityRefs(data, "key")` |
| `$ref` → markdown/entityName/email | string-like refs | `types.String` | `schema.StringAttribute{}` | `StringVal(data, "key")` |
| `"type": "object"` | — | `types.String` (JSON) | `schema.StringAttribute{Sensitive: true}` | **Not read back** |
| Complex `oneOf`/`anyOf` | — | `types.String` (raw JSON) | `schema.StringAttribute{Sensitive: true}` | **Not read back** |

## Entity Reference Lists (Critical Pattern)

When OM schema has an array of entity names/FQNs (e.g., `policies`, `teams`, `parents`), the **TF user writes plain strings** (FQNs or names), but **OM returns entity reference objects** `[{"id": "...", "name": "...", "fullyQualifiedName": "..."}]`.

**In buildBody**: Send as plain string array (OM resolves names).
```go
if !plan.Policies.IsNull() && !plan.Policies.IsUnknown() {
    var vals []string
    plan.Policies.ElementsAs(ctx, &vals, false)
    body["policies"] = vals
}
```

**In readIntoState**: Extract names from entity ref objects.
```go
if names := EntityRefNames(data, "policies"); names != nil {
    state.Policies, _ = types.ListValueFrom(context.Background(), types.StringType, names)
}
```

## Complex/Polymorphic Fields

For fields like `connection` (database services) or `rules` (policies) where the schema is `oneOf` with many variants:

1. Model as `types.String` containing raw JSON
2. Mark as `Sensitive: true` and `Optional: true`
3. In `buildBody`: `json.Unmarshal` the string into `interface{}` and set on body
4. In `readIntoState`: **Do NOT read back** — OM may mask sensitive fields. Preserve the user's input via Terraform state.

```go
// Schema
"connection_json": schema.StringAttribute{
    Description: "Connection config as JSON string.",
    Optional:    true,
    Sensitive:   true,
},

// buildBody
if !plan.ConnectionJSON.IsNull() && !plan.ConnectionJSON.IsUnknown() {
    var conn interface{}
    json.Unmarshal([]byte(plan.ConnectionJSON.ValueString()), &conn)
    body["connection"] = conn
}

// readIntoState — skip the field (preserved in TF state)
```

## Parent-Child Entities (FQN-Based Reads)

For entities that belong to a parent (e.g., Tag → Classification, GlossaryTerm → Glossary):

1. Add a **parent reference attribute** that is `Required` + `RequiresReplace`
2. In `Read`: Build the FQN from parent + name for `GetByName`
3. In `readIntoState`: Extract the parent from the FQN

Example from tag.go:
```go
// Read: build FQN
fqn := state.Classification.ValueString() + "." + state.Name.ValueString()
raw, err := r.client.GetByName(ctx, tagCollection, fqn, fields)

// readIntoState: extract parent from FQN
if fqn := state.FQN.ValueString(); fqn != "" {
    parts := splitFQN(fqn)
    if len(parts) > 0 {
        state.Classification = types.StringValue(parts[0])
    }
}
```

## Fields to Skip

Skip these fields in the resource (they're internal OM fields):
- `extension` — custom properties, too dynamic
- `votes` — read-only aggregation
- `followers` — managed separately
- `children` — computed relationships
- `changeDescription` — audit trail
- `version` — OM internal versioning
- `updatedAt`, `updatedBy` — audit metadata
- `href` — API links
- `deleted` — soft-delete flag
- `provider` — OM system field
- `tags` — managed via separate `openmetadata_tag` + tag assignment resources

## Read Fields (GET Query Parameter)

The `?fields=` query parameter on GET requests must list expandable fields the resource uses. Only include fields that:
1. Exist in the entity's schema as expandable (typically: `owners`, `domains`, `policies`, `parents`, `tags`)
2. Are actually modeled in your resource

```go
raw, err := r.client.GetByName(ctx, collection, name, []string{"owners", "domains", "policies"})
```

If you include a field that doesn't exist on the entity, OM returns HTTP 400. **Only list fields you've verified in the schema.**

## Enum Fields

For `"type": "string"` with `"enum"` values, add a validator:

```go
import "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
import "github.com/hashicorp/terraform-plugin-framework/schema/validator"

"service_type": schema.StringAttribute{
    Description: "Type of service.",
    Required:    true,
    Validators: []validator.String{
        stringvalidator.OneOf("Mysql", "Postgres", "BigQuery", ...),
    },
},
```

## Boolean Fields with Defaults

```go
import "github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"

"mutually_exclusive": schema.BoolAttribute{
    Description: "When true, child items are mutually exclusive.",
    Optional:    true,
    Computed:    true,
    Default:     booldefault.StaticBool(false),
},
```
