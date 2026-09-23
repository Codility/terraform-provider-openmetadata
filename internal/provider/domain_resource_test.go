// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccDomainResource exercises the full CRUD + import lifecycle of
// openmetadata_domain.
func TestAccDomainResource(t *testing.T) {
	name := testRandName("dom")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// ── Create and Read ──────────────────────────────────────────────
			{
				Config: testAccDomainConfig(name, "Initial domain description", "Source-aligned"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_domain.test", "name", name),
					resource.TestCheckResourceAttr("openmetadata_domain.test", "description", "Initial domain description"),
					resource.TestCheckResourceAttr("openmetadata_domain.test", "domain_type", "Source-aligned"),
					resource.TestCheckResourceAttrSet("openmetadata_domain.test", "id"),
					resource.TestCheckResourceAttrSet("openmetadata_domain.test", "fully_qualified_name"),
				),
			},
			// ── Update description ────────────────────────────────────────────
			// domain_type has RequiresReplace — keep it the same to avoid recreation.
			{
				Config: testAccDomainConfig(name, "Updated domain description", "Source-aligned"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_domain.test", "description", "Updated domain description"),
				),
			},
			// ── Import ───────────────────────────────────────────────────────
			// Import by name (not UUID). The provider's ImportState uses GetByName.
			{
				ResourceName:      "openmetadata_domain.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["openmetadata_domain.test"]
					return rs.Primary.Attributes["name"], nil
				},
			},
		},
	})
}

// TestAccDomainResourceTypes verifies all valid domain_type values are accepted.
func TestAccDomainResourceTypes(t *testing.T) {
	for _, domainType := range []string{"Source-aligned", "Consumer-aligned", "Aggregate"} {
		domainType := domainType // capture loop variable
		t.Run(domainType, func(t *testing.T) {
			name := testRandName("dom")
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: testAccDomainConfig(name, "Domain type acceptance test", domainType),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("openmetadata_domain.test", "domain_type", domainType),
						),
					},
				},
			})
		})
	}
}

func testAccDomainConfig(name, description, domainType string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_domain" "test" {
  name        = %q
  description = %q
  domain_type = %q
}
`, testProviderBlock(), name, description, domainType)
}
