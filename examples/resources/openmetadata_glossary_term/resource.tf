resource "openmetadata_glossary_term" "customer" {
  glossary    = openmetadata_glossary.business_glossary.name
  name        = "Customer"
  description = "An organization or individual that has purchased or is evaluating our product."
  synonyms    = ["Client", "Account"]
}
