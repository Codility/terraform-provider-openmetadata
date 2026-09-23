// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccTeamResource exercises the full CRUD + import lifecycle of
// openmetadata_team.
func TestAccTeamResource(t *testing.T) {
	name := testRandName("team")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// ── Create and Read ──────────────────────────────────────────────
			// Use Department (not Group): the OM API does not allow updating Group
			// teams, but Department teams can be updated normally.
			{
				Config: testAccTeamConfig(name, "Initial team description", "Department"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_team.test", "name", name),
					resource.TestCheckResourceAttr("openmetadata_team.test", "description", "Initial team description"),
					resource.TestCheckResourceAttr("openmetadata_team.test", "team_type", "Department"),
					resource.TestCheckResourceAttr("openmetadata_team.test", "is_joinable", "true"),
					resource.TestCheckResourceAttrSet("openmetadata_team.test", "id"),
					resource.TestCheckResourceAttrSet("openmetadata_team.test", "fully_qualified_name"),
				),
			},
			// ── Update description ────────────────────────────────────────────
			{
				Config: testAccTeamConfig(name, "Updated team description", "Department"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_team.test", "description", "Updated team description"),
					resource.TestCheckResourceAttr("openmetadata_team.test", "team_type", "Department"),
				),
			},
			// ── Import ───────────────────────────────────────────────────────
			// Import by name (not UUID). parents is ignored: OM always places
			// teams under "Organisation" and the FQN may differ from short name.
			{
				ResourceName:            "openmetadata_team.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"parents"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["openmetadata_team.test"]
					return rs.Primary.Attributes["name"], nil
				},
			},
		},
	})
}

// TestAccTeamResourceWithEmail verifies the optional email field.
func TestAccTeamResourceWithEmail(t *testing.T) {
	name := testRandName("team")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTeamConfigWithEmail(name, "team@example.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_team.test", "name", name),
					resource.TestCheckResourceAttr("openmetadata_team.test", "email", "team@example.com"),
				),
			},
		},
	})
}

func testAccTeamConfig(name, description, teamType string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_team" "test" {
  name        = %q
  description = %q
  team_type   = %q
}
`, testProviderBlock(), name, description, teamType)
}

func testAccTeamConfigWithEmail(name, email string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_team" "test" {
  name  = %q
  email = %q
}
`, testProviderBlock(), name, email)
}
