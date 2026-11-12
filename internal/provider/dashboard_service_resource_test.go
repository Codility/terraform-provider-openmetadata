// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const dashSvc = "openmetadata_dashboard_service.test"

// TestAccDashboardServiceResource covers create, patch-based update (description,
// display name, domains) and import by name.
func TestAccDashboardServiceResource(t *testing.T) {
	name := testRandName("dashsv")
	domain := testRandName("dsdom")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDashboardServiceGone(name),
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardServiceConfig(name, domain, "Initial description", "", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dashSvc, "name", name),
					resource.TestCheckResourceAttr(dashSvc, "service_type", "CustomDashboard"),
					resource.TestCheckResourceAttr(dashSvc, "description", "Initial description"),
					resource.TestCheckResourceAttr(dashSvc, "hard_delete", "false"),
					resource.TestCheckResourceAttr(dashSvc, "domains.#", "0"),
					resource.TestCheckResourceAttrSet(dashSvc, "id"),
					resource.TestCheckResourceAttrSet(dashSvc, "fully_qualified_name"),
				),
			},
			{
				Config: testAccDashboardServiceConfig(name, domain, "Updated description", "Saved Reports", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dashSvc, "description", "Updated description"),
					resource.TestCheckResourceAttr(dashSvc, "display_name", "Saved Reports"),
					resource.TestCheckResourceAttr(dashSvc, "domains.#", "1"),
					resource.TestCheckResourceAttr(dashSvc, "domains.0", domain),
				),
			},
			{
				ResourceName:            dashSvc,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connection_json", "owners", "hard_delete"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return s.RootModule().Resources[dashSvc].Primary.Attributes["name"], nil
				},
			},
		},
	})
}

// TestAccDashboardServiceKeepsConnection: once connection_json is dropped from config,
// updates must not touch the stored connection (a PUT without one fails in OM).
func TestAccDashboardServiceKeepsConnection(t *testing.T) {
	name := testRandName("dashcn")
	conn := `{"config":{"type":"CustomDashboard","sourcePythonClass":"tfacc.probe.Source"}}`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDashboardServiceGone(name),
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardServiceWithConnection(name, "With connection", conn),
				Check:  testAccCheckDashboardServiceSourceClass(name, "tfacc.probe.Source"),
			},
			{
				Config: testAccDashboardServiceWithConnection(name, "Connection now UI-managed", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dashSvc, "description", "Connection now UI-managed"),
					testAccCheckDashboardServiceSourceClass(name, "tfacc.probe.Source"),
				),
			},
		},
	})
}

// TestAccDashboardServiceRefusesExisting: create must not adopt a service that already exists.
func TestAccDashboardServiceRefusesExisting(t *testing.T) {
	name := testRandName("dashex")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardServiceWithConnection(name, "First", ""),
			},
			{
				Config: testAccDashboardServiceWithConnection(name, "First", "") + fmt.Sprintf(`
resource "openmetadata_dashboard_service" "duplicate" {
  name         = %q
  service_type = "CustomDashboard"
}
`, name),
				ExpectError: regexp.MustCompile(`already exists`),
			},
		},
	})
}

func testAccDashboardServiceConfig(name, domain, description, displayName string, withDomain bool) string {
	extra := ""
	if displayName != "" {
		extra += fmt.Sprintf("  display_name = %q\n", displayName)
	}
	if withDomain {
		extra += "  domains      = [openmetadata_domain.test.name]\n"
	}
	return fmt.Sprintf(`
%s

resource "openmetadata_domain" "test" {
  name        = %q
  description = "Domain for dashboard service tests"
  domain_type = "Aggregate"
}

resource "openmetadata_dashboard_service" "test" {
  name         = %q
  service_type = "CustomDashboard"
  description  = %q
%s}
`, testProviderBlock(), domain, name, description, extra)
}

func testAccDashboardServiceWithConnection(name, description, conn string) string {
	connLine := ""
	if conn != "" {
		connLine = fmt.Sprintf("  connection_json = %q\n", conn)
	}
	return fmt.Sprintf(`
%s

resource "openmetadata_dashboard_service" "test" {
  name         = %q
  service_type = "CustomDashboard"
  description  = %q
%s}
`, testProviderBlock(), name, description, connLine)
}

// omGet reads an entity straight from the API, bypassing the provider.
func omGet(path string) (int, map[string]interface{}, error) {
	req, err := http.NewRequest(http.MethodGet, os.Getenv("OPENMETADATA_HOST")+"/api/v1/"+path, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENMETADATA_TOKEN"))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	var data map[string]interface{}
	_ = json.Unmarshal(body, &data)
	return resp.StatusCode, data, nil
}

func testAccCheckDashboardServiceSourceClass(name, want string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		_, data, err := omGet("services/dashboardServices/name/" + url.PathEscape(name))
		if err != nil {
			return err
		}
		conn, _ := data["connection"].(map[string]interface{})
		cfg, _ := conn["config"].(map[string]interface{})
		if got, _ := cfg["sourcePythonClass"].(string); got != want {
			return fmt.Errorf("stored connection lost: sourcePythonClass = %q, want %q", got, want)
		}
		return nil
	}
}

// Destroy soft-deletes by default: the service is gone from normal reads but still restorable.
func testAccCheckDashboardServiceGone(name string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		status, _, err := omGet("services/dashboardServices/name/" + url.PathEscape(name))
		if err != nil {
			return err
		}
		if status != http.StatusNotFound {
			return fmt.Errorf("dashboard service %q still readable after destroy (HTTP %d)", name, status)
		}
		status, data, err := omGet("services/dashboardServices/name/" + url.PathEscape(name) + "?include=deleted")
		if err != nil {
			return err
		}
		if status != http.StatusOK || data["deleted"] != true {
			return fmt.Errorf("dashboard service %q was not soft-deleted (HTTP %d)", name, status)
		}
		return nil
	}
}
