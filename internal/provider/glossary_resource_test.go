// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccGlossaryResource exercises the full CRUD + import lifecycle of
// openmetadata_glossary.
func TestAccGlossaryResource(t *testing.T) {
	name := testRandName("glos")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// ── Create and Read ──────────────────────────────────────────────
			{
				Config: testAccGlossaryConfig(name, "Initial glossary description", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_glossary.test", "name", name),
					resource.TestCheckResourceAttr("openmetadata_glossary.test", "description", "Initial glossary description"),
					resource.TestCheckResourceAttr("openmetadata_glossary.test", "mutually_exclusive", "false"),
					resource.TestCheckResourceAttrSet("openmetadata_glossary.test", "id"),
					resource.TestCheckResourceAttrSet("openmetadata_glossary.test", "fully_qualified_name"),
				),
			},
			// ── Update ───────────────────────────────────────────────────────
			{
				Config: testAccGlossaryConfig(name, "Updated glossary description", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_glossary.test", "description", "Updated glossary description"),
					resource.TestCheckResourceAttr("openmetadata_glossary.test", "mutually_exclusive", "true"),
				),
			},
			// ── Import ───────────────────────────────────────────────────────
			// Import by name (not UUID). The provider's ImportState uses GetByName.
			{
				ResourceName:      "openmetadata_glossary.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["openmetadata_glossary.test"]
					return rs.Primary.Attributes["name"], nil
				},
			},
		},
	})
}

func testAccGlossaryConfig(name, description string, mutuallyExclusive bool) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_glossary" "test" {
  name               = %q
  description        = %q
  mutually_exclusive = %t
}
`, testProviderBlock(), name, description, mutuallyExclusive)
}
