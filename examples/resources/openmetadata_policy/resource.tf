resource "openmetadata_policy" "data_access" {
  name         = "DataAccessPolicy"
  display_name = "Data Access Policy"
  description  = "Policy governing access to data assets."
}
