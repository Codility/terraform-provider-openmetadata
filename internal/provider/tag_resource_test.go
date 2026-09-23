// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccTagResource exercises the full CRUD + import lifecycle of
// openmetadata_tag. A classification is created as a dependency and destroyed
// after the test.
func TestAccTagResource(t *testing.T) {
	classificationName := testRandName("cls")
	tagName := testRandName("tag")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// ── Create and Read ──────────────────────────────────────────────
			{
				Config: testAccTagConfig(classificationName, tagName, "Initial tag description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_tag.test", "name", tagName),
					resource.TestCheckResourceAttr("openmetadata_tag.test", "description", "Initial tag description"),
					resource.TestCheckResourceAttr("openmetadata_tag.test", "classification", classificationName),
					resource.TestCheckResourceAttrSet("openmetadata_tag.test", "id"),
					resource.TestCheckResourceAttrSet("openmetadata_tag.test", "fully_qualified_name"),
				),
			},
			// ── Update description ────────────────────────────────────────────
			{
				Config: testAccTagConfig(classificationName, tagName, "Updated tag description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_tag.test", "description", "Updated tag description"),
				),
			},
			// ── Import ───────────────────────────────────────────────────────
			// Import by FQN: "<classification>.<tag>" (not UUID).
			// parent is not read back from the API in readIntoState (it stays
			// null), so it is excluded from the state-equality check.
			{
				ResourceName:            "openmetadata_tag.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"parent"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["openmetadata_tag.test"]
					return rs.Primary.Attributes["fully_qualified_name"], nil
				},
			},
		},
	})
}

func testAccTagConfig(classificationName, tagName, description string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_classification" "parent" {
  name        = %q
  description = "Parent classification for tag acceptance test"
}

resource "openmetadata_tag" "test" {
  name           = %q
  description    = %q
  classification = openmetadata_classification.parent.name
}
`, testProviderBlock(), classificationName, tagName, description)
}
