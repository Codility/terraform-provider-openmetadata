resource "openmetadata_role" "data_steward" {
  name         = "DataSteward"
  display_name = "Data Steward"
  description  = "Responsible for data quality and governance."
  policies     = [openmetadata_policy.data_access.name]
}
