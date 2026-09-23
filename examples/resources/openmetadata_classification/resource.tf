resource "openmetadata_classification" "data_quality" {
  name               = "DataQuality"
  display_name       = "Data Quality"
  description        = "Tags for data quality status."
  mutually_exclusive = true
}
