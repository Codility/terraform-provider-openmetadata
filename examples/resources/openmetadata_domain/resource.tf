resource "openmetadata_domain" "analytics" {
  name         = "Analytics"
  display_name = "Analytics"
  description  = "Owns reporting and BI data products."
  domain_type  = "Aggregate"
}
