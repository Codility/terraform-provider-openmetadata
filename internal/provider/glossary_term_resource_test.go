// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccGlossaryTermResource exercises the full CRUD + import lifecycle of
// openmetadata_glossary_term. A glossary is created as a dependency.
func TestAccGlossaryTermResource(t *testing.T) {
	glossaryName := testRandName("glos")
	termName := testRandName("term")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// ── Create and Read ──────────────────────────────────────────────
			{
				Config: testAccGlossaryTermConfig(glossaryName, termName, "Initial term description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_glossary_term.test", "name", termName),
					resource.TestCheckResourceAttr("openmetadata_glossary_term.test", "description", "Initial term description"),
					resource.TestCheckResourceAttrSet("openmetadata_glossary_term.test", "id"),
					resource.TestCheckResourceAttrSet("openmetadata_glossary_term.test", "fully_qualified_name"),
					// glossary is read back from the API response as the FQN of the parent glossary
					resource.TestCheckResourceAttrSet("openmetadata_glossary_term.test", "glossary"),
				),
			},
			// ── Update description and add synonyms ───────────────────────────
			{
				Config: testAccGlossaryTermWithSynonymsConfig(glossaryName, termName, "Updated term description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_glossary_term.test", "description", "Updated term description"),
					resource.TestCheckResourceAttr("openmetadata_glossary_term.test", "synonyms.#", "1"),
					resource.TestCheckResourceAttr("openmetadata_glossary_term.test", "synonyms.0", "alias_one"),
				),
			},
			// ── Import ───────────────────────────────────────────────────────
			// Import by FQN: "<glossary>.<term>" (not UUID).
			// parent is not read back from the API in readIntoState (stays null).
			{
				ResourceName:            "openmetadata_glossary_term.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"parent"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["openmetadata_glossary_term.test"]
					return rs.Primary.Attributes["fully_qualified_name"], nil
				},
			},
		},
	})
}

func testAccGlossaryTermConfig(glossaryName, termName, description string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_glossary" "parent" {
  name        = %q
  description = "Parent glossary for term acceptance test"
}

resource "openmetadata_glossary_term" "test" {
  name        = %q
  description = %q
  glossary    = openmetadata_glossary.parent.name
}
`, testProviderBlock(), glossaryName, termName, description)
}

func testAccGlossaryTermWithSynonymsConfig(glossaryName, termName, description string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_glossary" "parent" {
  name        = %q
  description = "Parent glossary for term acceptance test"
}

resource "openmetadata_glossary_term" "test" {
  name        = %q
  description = %q
  glossary    = openmetadata_glossary.parent.name
  synonyms    = ["alias_one"]
}
`, testProviderBlock(), glossaryName, termName, description)
}
