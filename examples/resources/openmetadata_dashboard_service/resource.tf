# A service with no external connection, e.g. to hold hand-made data models.
resource "openmetadata_dashboard_service" "saved_reports" {
  name         = "SavedReports"
  service_type = "CustomDashboard"
  description  = "Saved reports: parameterised SQL with documented output columns."
  domains      = ["Data"]

  lifecycle {
    prevent_destroy = true
  }
}

# An ingested BI service whose connection (and secrets) stay managed in the OpenMetadata UI.
# Omitting connection_json means the provider never sends or changes the connection.
resource "openmetadata_dashboard_service" "tableau" {
  name         = "Tableau Dashboards"
  service_type = "Tableau"
  description  = "Our Tableau dashboards"
  domains      = ["Data"]

  lifecycle {
    prevent_destroy = true
  }
}
