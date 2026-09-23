# Testing a New Resource

Standard procedure for smoke-testing a newly generated resource against a live OpenMetadata instance.

## Prerequisites

- A running OpenMetadata instance you can safely create/destroy test resources on
- A valid JWT token for the OpenMetadata API
- Go toolchain installed
- Terraform CLI installed
- `~/.terraformrc` configured with `dev_overrides` so Terraform uses the local binary:
  ```hcl
  provider_installation {
    dev_overrides {
      "open-metadata/openmetadata" = "<directory-containing-the-binary>"
    }
    direct {}
  }
  ```

## Build

From the provider repository root:

```sh
go build -o <output-dir>/terraform-provider-openmetadata .
go vet ./...
```

The `<output-dir>` must match the path in `dev_overrides` above.

## Write Test Config

Create a `main.tf` in a temporary directory:

```hcl
terraform {
  required_providers {
    openmetadata = {
      source = "open-metadata/openmetadata"
    }
  }
}

provider "openmetadata" {}

resource "openmetadata_<entity>" "test" {
  name        = "TFSmokeTest<Entity>"
  description = "Temporary resource for TF smoke test. Safe to delete."
  # ... entity-specific fields ...
}
```

**Naming convention**: Prefix test resources with `TFSmokeTest` so they're identifiable for cleanup.

## Run the Test Cycle

All commands from the test directory. Set the required environment variables:

```sh
export OPENMETADATA_HOST="https://your-openmetadata-instance.example.com"
export OPENMETADATA_TOKEN="<your-jwt-token>"
```

### 1. Plan (verify config is valid)
```sh
terraform plan
```
Should show "N to add, 0 to change, 0 to destroy".

### 2. Apply (create resources)
```sh
terraform apply -auto-approve
```
All resources should be created successfully.

### 3. Idempotency Check (critical)
```sh
terraform plan
```
**Must show "No changes."** If it shows changes, there's a drift between `buildBody` and `readIntoState` — the resource reads back different values than what was written.

Common idempotency issues:
- Field not read back in `readIntoState` → Terraform sees "known after apply" drift
- Entity ref list: writing `["PolicyName"]` but reading back with different casing
- Bool defaults: not setting `Computed: true` + `Default` on optional bools

### 4. Destroy (cleanup)
```sh
terraform destroy -auto-approve
```
All resources destroyed. Verify in the OpenMetadata UI if needed.

### 5. Clean State
```sh
rm -f terraform.tfstate terraform.tfstate.backup
```

## After Testing

1. Remove `dev_overrides` from `~/.terraformrc` (or restore from backup) if you no longer need local development mode.
2. Clean the provider binary from the output directory.
