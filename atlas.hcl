data "external_schema" "gorm" {
  program = [
    "go", "run", "-mod=mod", "ariga.io/atlas-provider-gorm", "load",
    "--path", "./internal/models",
    "--dialect", "postgres",
  ]
}

env "gorm" {
  src = data.external_schema.gorm.url
  dev = getenv("ATLAS_DEV_URL")

  migration {
    dir = "file://migrations"
  }
}
