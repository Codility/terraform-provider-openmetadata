// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccDatabaseServiceResource exercises the full CRUD + import lifecycle of
// openmetadata_database_service.
//
// A minimal MySQL connection config is used. OpenMetadata does not validate
// connectivity on create — it only stores the config — so fake credentials are
// fine for acceptance testing.
//
// connection_json is excluded from import verification by design: the provider
// intentionally does not read the connection back from the API (it may contain
// masked/redacted fields), so the original JSON is preserved in state only.
func TestAccDatabaseServiceResource(t *testing.T) {
	name := testRandName("dbsvc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// ── Create and Read ──────────────────────────────────────────────
			{
				Config: testAccDatabaseServiceConfig(name, "Initial db service description", "Mysql"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_database_service.test", "name", name),
					resource.TestCheckResourceAttr("openmetadata_database_service.test", "description", "Initial db service description"),
					resource.TestCheckResourceAttr("openmetadata_database_service.test", "service_type", "Mysql"),
					resource.TestCheckResourceAttrSet("openmetadata_database_service.test", "id"),
					resource.TestCheckResourceAttrSet("openmetadata_database_service.test", "fully_qualified_name"),
				),
			},
			// ── Update description ────────────────────────────────────────────
			// service_type has RequiresReplace — keep it the same.
			{
				Config: testAccDatabaseServiceConfig(name, "Updated db service description", "Mysql"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_database_service.test", "description", "Updated db service description"),
				),
			},
			// ── Import ───────────────────────────────────────────────────────
			// Import by name (not UUID). The provider's ImportState uses GetByName.
			{
				ResourceName:            "openmetadata_database_service.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connection_json"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["openmetadata_database_service.test"]
					return rs.Primary.Attributes["name"], nil
				},
			},
		},
	})
}

// TestAccDatabaseServiceResourceTypes verifies a selection of service_type
// values is accepted without error.
func TestAccDatabaseServiceResourceTypes(t *testing.T) {
	for _, svcType := range []string{"Postgres", "BigQuery", "Snowflake"} {
		svcType := svcType
		t.Run(svcType, func(t *testing.T) {
			name := testRandName("dbsvc")
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: testAccDatabaseServiceConfig(name, "Service type test", svcType),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("openmetadata_database_service.test", "service_type", svcType),
						),
					},
				},
			})
		})
	}
}

// dbServiceConnectionJSON returns a minimal connection JSON that satisfies the
// OpenMetadata schema for the given service type.
// Connectivity is not tested — OM stores the config without validating reachability.
func dbServiceConnectionJSON(serviceType string) string {
	switch serviceType {
	case "Postgres":
		return fmt.Sprintf(`{"config":{"type":%q,"scheme":"postgresql+psycopg2","username":"test","hostPort":"localhost:5432"}}`, serviceType)
	case "BigQuery":
		// Use GCP ADC (Application Default Credentials) — minimal valid BigQuery config.
		// gcpConfig (not gcsConfig) is the correct field name per the OM JSON schema.
		return fmt.Sprintf(`{"config":{"type":%q,"credentials":{"gcpConfig":{"type":"gcp_adc","projectId":"my-project"}}}}`, serviceType)
	case "Snowflake":
		return fmt.Sprintf(`{"config":{"type":%q,"username":"test","password":"test","account":"testaccount","database":"testdb","warehouse":"testwh"}}`, serviceType)
	default:
		// Mysql and other types
		return fmt.Sprintf(`{"config":{"type":%q,"scheme":"mysql+pymysql","username":"test","hostPort":"localhost:3306"}}`, serviceType)
	}
}

func testAccDatabaseServiceConfig(name, description, serviceType string) string {
	connectionJSON := dbServiceConnectionJSON(serviceType)
	return fmt.Sprintf(`
%s

resource "openmetadata_database_service" "test" {
  name            = %q
  description     = %q
  service_type    = %q
  connection_json = %q
}
`, testProviderBlock(), name, description, serviceType, connectionJSON)
}
