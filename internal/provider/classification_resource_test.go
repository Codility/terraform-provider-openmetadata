// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccClassificationResource exercises the full CRUD + import lifecycle of
// openmetadata_classification.
func TestAccClassificationResource(t *testing.T) {
	name := testRandName("cls")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// ── Create and Read ──────────────────────────────────────────────
			{
				Config: testAccClassificationConfig(name, "Initial description", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_classification.test", "name", name),
					resource.TestCheckResourceAttr("openmetadata_classification.test", "description", "Initial description"),
					resource.TestCheckResourceAttr("openmetadata_classification.test", "mutually_exclusive", "false"),
					resource.TestCheckResourceAttrSet("openmetadata_classification.test", "id"),
					resource.TestCheckResourceAttrSet("openmetadata_classification.test", "fully_qualified_name"),
				),
			},
			// ── Update ───────────────────────────────────────────────────────
			{
				Config: testAccClassificationConfig(name, "Updated description", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_classification.test", "description", "Updated description"),
					resource.TestCheckResourceAttr("openmetadata_classification.test", "mutually_exclusive", "true"),
				),
			},
			// ── Import ───────────────────────────────────────────────────────
			// Import by name (not UUID). The provider's ImportState uses GetByName.
			{
				ResourceName:      "openmetadata_classification.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["openmetadata_classification.test"]
					return rs.Primary.Attributes["name"], nil
				},
			},
		},
	})
}

func testAccClassificationConfig(name, description string, mutuallyExclusive bool) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_classification" "test" {
  name               = %q
  description        = %q
  mutually_exclusive = %t
}
`, testProviderBlock(), name, description, mutuallyExclusive)
}
