# Security Policy

## Reporting a vulnerability

**Do not open a public issue.**

Report privately through GitHub: **Security → Advisories → Report a vulnerability** on this repository. That opens a draft advisory visible only to you and the maintainers.

Please include what you can — affected version, reproduction steps, and what an attacker gains. We aim to acknowledge within five working days.

## Scope

This repository contains only the Terraform provider. Vulnerabilities in OpenMetadata itself belong with the [OpenMetadata project](https://github.com/open-metadata/OpenMetadata).

In scope for this repo: credential handling (the `token` attribute and `OPENMETADATA_TOKEN`), the HTTP client's transport and error paths, and anything that leaks secrets into Terraform state, plan output, or logs.

## Supported versions

The latest released version only. Fixes ship as a new release rather than a patch to an older tag.

## Releases

Release checksums are signed with the Codility GPG key published under the `codility` Terraform Registry namespace. Verify a release before use:

```bash
gpg --verify terraform-provider-openmetadata_<version>_SHA256SUMS.sig \
             terraform-provider-openmetadata_<version>_SHA256SUMS
```

Terraform performs this check automatically when installing from the registry.
