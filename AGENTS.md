# Agent Instructions: terraform-provider-openmetadata

Terraform provider for managing OpenMetadata resources. Built with the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework).

## Project Structure

| Path | Purpose |
|------|---------|
| `internal/provider/` | Provider configuration, resource registration |
| `internal/client/` | HTTP client for OpenMetadata REST API |
| `internal/resources/` | Resource implementations (one file per resource) |
| `internal/resources/common.go` | Shared schema attributes, JSON helpers, entity ref utilities |
| `examples/resources/<name>/` | Per-resource HCL examples (used by tfplugindocs) |
| `docs/` | Auto-generated documentation (do not edit manually) |
| `tools/tools.go` | Tool dependencies (tfplugindocs) |

## Key Conventions

- **One file per resource** in `internal/resources/`. File name matches the entity (e.g., `team.go`, `glossary_term.go`).
- **All resources** must be registered in `internal/provider/provider.go` → `Resources()`.
- **CRUD pattern**: All resources use OpenMetadata's idempotent PUT via `client.CreateOrUpdate()`. Read uses `GetByName()` or `GetByID()`. Delete uses `Delete()`.
- **Entity refs** (owners, domains, policies, etc.) are stored as `[]string` of FQN names. Use `EntityRefNames()` from `common.go` to extract names from API responses.
- **Documentation** is auto-generated via `make docs` (runs tfplugindocs). Never edit files in `docs/` directly. Update examples in `examples/resources/<name>/resource.tf` instead.
- **Releases** are handled by GoReleaser + GitHub Actions (`.goreleaser.yml`, `.github/workflows/release.yml`).

## Adding a New Resource

Use the **codegen skill** at `.github/skills/codegen/SKILL.md`. It reads the OpenMetadata JSON schema and generates a complete Go resource file following existing patterns.

Prompt your AI agent: *"Generate a TF resource for `<entityName>`"*

## Skills

| Skill | Path | Use when... |
|-------|------|------------|
| `codegen` | `.github/skills/codegen/SKILL.md` | Adding a new resource — generates Go code from OM JSON schema |

## Build & Test

```bash
make build   # Compile
make test    # Run tests
make lint    # Linter
make docs    # Regenerate documentation
```
