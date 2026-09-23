resource "openmetadata_tag" "verified" {
  name           = "Verified"
  classification = openmetadata_classification.data_quality.name
  description    = "Data has been verified and is production-ready."
}
