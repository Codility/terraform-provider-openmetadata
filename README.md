# terraform-provider-openmetadata

Terraform provider for managing [OpenMetadata](https://open-metadata.org/) resources as code.

## Supported Resources

| Resource | Description |
|---|---|
| `openmetadata_team` | Teams (Group, Department, Division, BusinessUnit) |
| `openmetadata_classification` | Tag classifications (categories) |
| `openmetadata_tag` | Tags within a classification |
| `openmetadata_glossary` | Business glossaries |
| `openmetadata_glossary_term` | Terms within a glossary |
| `openmetadata_policy` | Access control policies |
| `openmetadata_role` | Roles referencing policies |
| `openmetadata_database_service` | Database service connections |
| `openmetadata_domain` | Domains (Source-aligned, Consumer-aligned, Aggregate) |

## Quick Start

```hcl
terraform {
  required_providers {
    openmetadata = {
      source  = "codility/openmetadata"
      version = "~> 0.1"
    }
  }
}

provider "openmetadata" {
  host  = "https://openmetadata.example.com"
  token = var.openmetadata_token
}

resource "openmetadata_team" "data_engineering" {
  name         = "DataEngineering"
  display_name = "Data Engineering"
  description  = "Owns ETL pipelines and data infrastructure."
  team_type    = "Group"
}
```

## Authentication

| Method | Details |
|---|---|
| Provider block | `host` and `token` attributes |
| Environment variables | `OPENMETADATA_HOST` and `OPENMETADATA_TOKEN` |

Environment variables are used as fallback when provider attributes are not set.

## Building

```bash
make build     # Requires Go 1.22+
make install   # Build + install to ~/.terraform.d/plugins
```

## Development

```bash
make build   # Compile the provider
make fmt     # Format Go code
make lint    # Run linter (requires golangci-lint)
make test    # Run tests
make docs    # Generate documentation via tfplugindocs
make deps    # Tidy Go modules
```

## Documentation

Provider documentation is generated using [tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs). After making schema changes:

```bash
make docs
```

Generated docs are placed in `docs/`. Per-resource examples live in `examples/resources/<resource_name>/resource.tf`.

## Import

All resources support `terraform import`:

```bash
terraform import openmetadata_team.example "TeamName"
terraform import openmetadata_classification.example "ClassificationName"
terraform import openmetadata_tag.example "Classification.TagName"
terraform import openmetadata_glossary.example "GlossaryName"
terraform import openmetadata_glossary_term.example "Glossary.TermName"
terraform import openmetadata_policy.example "PolicyName"
terraform import openmetadata_role.example "RoleName"
terraform import openmetadata_database_service.example "ServiceName"
terraform import openmetadata_domain.example "DomainName"
```

## Release

Releases are automated via [GoReleaser](https://goreleaser.com/) and GitHub Actions. To create a release:

1. Tag a commit: `git tag v0.1.0`
2. Push the tag: `git push origin v0.1.0`

The GitHub Actions workflow builds multi-platform binaries, signs checksums with GPG, and publishes a GitHub Release.

## Architecture

```
├── main.go                          # Provider entry point
├── internal/
│   ├── client/client.go             # HTTP client (auth, CRUD, error handling)
│   ├── provider/provider.go         # Provider config (host, token, resource registry)
│   │                                # + acceptance tests (one per resource)
│   └── resources/
│       ├── common.go                # Shared schema attributes + JSON helpers
│       ├── team.go                  # openmetadata_team
│       ├── classification.go        # openmetadata_classification
│       ├── tag.go                   # openmetadata_tag
│       ├── glossary.go              # openmetadata_glossary
│       ├── glossary_term.go         # openmetadata_glossary_term
│       ├── policy.go                # openmetadata_policy
│       ├── role.go                  # openmetadata_role
│       ├── database_service.go      # openmetadata_database_service
│       └── domain.go                # openmetadata_domain
├── .github/skills/codegen/          # AI codegen skill for new resources
├── examples/                        # Example Terraform configs
├── docker/test/                     # OpenMetadata stack for acceptance tests
└── Makefile                         # Build, test, install targets
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). To report a vulnerability, see [SECURITY.md](SECURITY.md) — do not open a public issue.

## License

[Apache License 2.0](LICENSE)
