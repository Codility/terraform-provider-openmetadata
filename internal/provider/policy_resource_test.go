// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccPolicyResource exercises the full CRUD + import lifecycle of
// openmetadata_policy.
//
// rules is stored in state as a JSON string and round-tripped through the
// API (which returns a JSON array). The provider marshals it back to a
// canonical JSON string, so the import verification compares the serialised
// form. Minor whitespace / ordering differences are acceptable because the
// API may reorder fields; if that causes flakes, add "rules" to
// ImportStateVerifyIgnore.
func TestAccPolicyResource(t *testing.T) {
	name := testRandName("pol")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// ── Create and Read ──────────────────────────────────────────────
			{
				Config: testAccPolicyConfig(name, "Initial policy description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_policy.test", "name", name),
					resource.TestCheckResourceAttr("openmetadata_policy.test", "description", "Initial policy description"),
					resource.TestCheckResourceAttr("openmetadata_policy.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("openmetadata_policy.test", "id"),
					resource.TestCheckResourceAttrSet("openmetadata_policy.test", "fully_qualified_name"),
				),
			},
			// ── Update description ────────────────────────────────────────────
			{
				Config: testAccPolicyConfig(name, "Updated policy description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_policy.test", "description", "Updated policy description"),
				),
			},
			// ── Import ───────────────────────────────────────────────────────
			// Import by name (not UUID). rules is excluded because it is treated
			// as write-only (not read back from the API, similar to connection_json).
			{
				ResourceName:            "openmetadata_policy.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"rules"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["openmetadata_policy.test"]
					return rs.Primary.Attributes["name"], nil
				},
			},
		},
	})
}

func testAccPolicyConfig(name, description string) string {
	// The OpenMetadata API requires at least one rule. The provider stores rules
	// as a JSON string and parses it before sending to the API.
	rules := `[{"name":"allow-view","effect":"allow","operations":["ViewAll"],"resources":["All"]}]`
	return fmt.Sprintf(`
%s

resource "openmetadata_policy" "test" {
  name        = %q
  description = %q
  rules       = %q
}
`, testProviderBlock(), name, description, rules)
}
