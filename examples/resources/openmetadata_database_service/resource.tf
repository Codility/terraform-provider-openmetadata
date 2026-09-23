resource "openmetadata_database_service" "postgres" {
  name         = "ProductionPostgres"
  service_type = "Postgres"
  display_name = "Production PostgreSQL"
  description  = "Main production PostgreSQL database."

  # The connection_json structure varies per service_type.
  # See the JSON schema for each connector:
  # https://github.com/open-metadata/OpenMetadata/tree/main/openmetadata-spec/src/main/resources/json/schema/entity/services/connections/database
  connection_json = jsonencode({
    config = {
      type             = "Postgres"
      scheme           = "postgresql+psycopg2"
      username         = "readonly_user"
      hostPort         = "db.example.com:5432"
      database         = "production"
      sslMode          = "verify-full"
    }
  })
}
