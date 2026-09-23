// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccRoleResource exercises the full CRUD + import lifecycle of
// openmetadata_role.
//
// The `policies` field references "OrganizationPolicy", which is seeded in
// every fresh OpenMetadata installation. Adjust if your instance uses a
// different default policy name.
//
// Import state verification ignores `policies` because the API returns policy
// references as fully-qualified names, which may differ from the short names
// sent during create (e.g. "OrganizationPolicy" vs "OrganizationPolicy").
// Once the provider normalises FQNs on read, this ignore can be removed.
func TestAccRoleResource(t *testing.T) {
	name := testRandName("role")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// ── Create and Read ──────────────────────────────────────────────
			{
				Config: testAccRoleConfig(name, "Initial role description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_role.test", "name", name),
					resource.TestCheckResourceAttr("openmetadata_role.test", "description", "Initial role description"),
					resource.TestCheckResourceAttrSet("openmetadata_role.test", "id"),
					resource.TestCheckResourceAttrSet("openmetadata_role.test", "fully_qualified_name"),
				),
			},
			// ── Update description ────────────────────────────────────────────
			{
				Config: testAccRoleConfig(name, "Updated role description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_role.test", "description", "Updated role description"),
				),
			},
			// ── Import ───────────────────────────────────────────────────────
			// Import by name (not UUID). The provider's ImportState uses GetByName.
			{
				ResourceName:            "openmetadata_role.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"policies"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["openmetadata_role.test"]
					return rs.Primary.Attributes["name"], nil
				},
			},
		},
	})
}

func testAccRoleConfig(name, description string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_role" "test" {
  name        = %q
  description = %q
  # OrganizationPolicy is seeded in every fresh OpenMetadata installation.
  policies = ["OrganizationPolicy"]
}
`, testProviderBlock(), name, description)
}
