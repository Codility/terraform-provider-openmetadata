# Contributing

Thanks for your interest. This provider is maintained by Codility and is open to outside contributions.

## Before you start

Open an issue first for anything beyond a bug fix or a doc correction. New resources in particular are worth agreeing on before code is written.

## Development

```bash
make build   # Compile
make test    # Unit tests — no OpenMetadata instance needed
make lint    # golangci-lint
make docs    # Regenerate docs/ via tfplugindocs
```

Acceptance tests run against a real OpenMetadata instance. `make testacc` starts one with docker compose, runs the suite, and tears it down — allow ~20 minutes and 8 GB of free memory. To reuse an instance you already have:

```bash
export OPENMETADATA_HOST=http://localhost:8585
export OPENMETADATA_TOKEN=<jwt>
make testacc-external
```

## Adding a resource

1. Generate it with the codegen skill in [.github/skills/codegen/SKILL.md](.github/skills/codegen/SKILL.md) — it reads the OpenMetadata JSON schema and follows the existing patterns.
2. One file per resource in `internal/resources/`, named after the entity.
3. Register it in `internal/provider/provider.go` → `Resources()`.
4. Add `internal/provider/<name>_resource_test.go`. CI fails if a resource has no acceptance test.
5. Add `examples/resources/openmetadata_<name>/resource.tf`, then run `make docs`.

Never edit files under `docs/` — they are generated. Change the example or the schema description instead.

## Pull requests

- Branch from `main`; fork if you are outside the org.
- [Conventional Commits](https://www.conventionalcommits.org/) for the title: `feat:`, `fix:`, `chore:`, `docs:`.
- Squash to one commit before review, and again after each round.
- All CI checks must pass. Acceptance tests from a fork need a maintainer to approve the run.
- A maintainer merges. Outside contributors do not have write access.

## Releases

Maintainers only. Tagging `vX.Y.Z` on `main` triggers GoReleaser, which signs the checksums with the Codility GPG key and publishes to the Terraform Registry.
