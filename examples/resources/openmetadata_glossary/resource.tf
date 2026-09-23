resource "openmetadata_glossary" "business_glossary" {
  name               = "BusinessGlossary"
  display_name       = "Business Glossary"
  description        = "Core business terms and definitions."
  mutually_exclusive = false
}
