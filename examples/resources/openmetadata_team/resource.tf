resource "openmetadata_team" "data_engineering" {
  name         = "DataEngineering"
  display_name = "Data Engineering"
  description  = "Owns ETL pipelines, data models, and data infrastructure."
  team_type    = "Group"
  is_joinable  = false
}
