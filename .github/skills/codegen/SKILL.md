---
name: codegen
description: "Generate a new Terraform resource for the OpenMetadata provider. Use when: adding a new OM entity, generating resource Go code, scaffolding CRUD for an OM API type. Reads OM JSON schemas and existing resources as examples to produce idiomatic Go."
argument-hint: "Entity name (e.g., 'ingestionPipeline', 'dataProduct', 'persona')"
---

# OpenMetadata Terraform Resource Generator

Generate a complete, production-ready Go resource file for the `terraform-provider-openmetadata` provider by reading the OpenMetadata JSON schema and following established patterns from existing hand-written resources.

## When to Use

- User asks to add a new resource/entity to the provider
- User asks to generate code for an OM entity type
- User wants to support a new OM API collection in Terraform

## Prerequisites

Before generating, you need:
1. **Entity name** (camelCase): e.g., `ingestionPipeline`, `dataProduct`, `persona`
2. **API collection path**: e.g., `services/ingestionPipelines`, `dataProducts`, `personas`
3. If unsure about the collection, check the [OM Swagger docs](https://sandbox.open-metadata.org/swagger.html) or ask the user.

## Procedure

### Step 1: Read the OM JSON Schema

The schemas live in the main OpenMetadata repository. If not already available locally, clone it first:

```
git clone --depth 1 https://github.com/open-metadata/OpenMetadata.git
```

Find the Create schema for the entity:

```
OpenMetadata/openmetadata-spec/src/main/resources/json/schema/api/<category>/create<Entity>.json
```

Common categories: `teams/`, `classification/`, `governance/`, `services/`, `data/`, `domains/`, `policies/`.

Also read the entity schema (for response field details):
```
OpenMetadata/openmetadata-spec/src/main/resources/json/schema/entity/<category>/<entity>.json
```

If neither file exists, ask the user to confirm the entity name.

### Step 2: Read Reference Files

Read these files to understand the patterns — **always** read them before generating:

1. **[common.go](../../../internal/resources/common.go)** — shared attribute helpers and JSON extraction utilities
2. **[client.go](../../../internal/client/client.go)** — HTTP client API (`CreateOrUpdate`, `GetByName`, `GetByID`, `Delete`)
3. **One simple resource** as primary pattern: **[classification.go](../../../internal/resources/classification.go)** or **[role.go](../../../internal/resources/role.go)**
4. **One complex resource** if the entity has parent-child or FQN relationships: **[tag.go](../../../internal/resources/tag.go)** or **[glossary_term.go](../../../internal/resources/glossary_term.go)**
5. **[database_service.go](../../../internal/resources/database_service.go)** if the entity has a polymorphic/complex JSON field (like `connection`)

### Step 3: Map Schema → Resource

Follow the field mapping rules in [./references/field-mapping.md](./references/field-mapping.md).

### Step 4: Generate the Resource File

Write the Go file to `internal/resources/<entity_snake>.go` following the template structure in [./references/resource-template.md](./references/resource-template.md).

### Step 5: Register & Build

1. **Register** the resource in `internal/provider/provider.go` → `Resources()` method. Add `resources.New<Entity>Resource,` to the return slice.
2. **Build:** Run `go build -o /tmp/terraform-provider-openmetadata .` from the project root
3. **Vet:** Run `go vet ./...`
4. Fix any compile errors, then repeat until clean.

### Step 6: Smoke Test (optional, if user requests)

See [./references/testing.md](./references/testing.md) for the standard test procedure.

## Important Conventions

- **Package**: All resources are in `package resources` (`internal/resources/`)
- **File naming**: `snake_case.go` matching the entity (e.g., `ingestion_pipeline.go`)
- **No `// Code generated` comment**: Unlike the old Python codegen, these files are hand-crafted by the agent and may be edited
- **Copyright header**: Always include the Apache 2.0 header at the top
- **Imports**: Use `goimports`-style grouping (stdlib, then external, then internal)
- **Error handling**: Use `resp.Diagnostics.AddError()` — never `log.Fatal` or `panic`
