variable "database_password" {
  type      = string
  sensitive = true
}

resource "mittwald_mysql_database" "martin_test" {
  project_id  = mittwald_project.foobar.id
  version     = "8.0"
  description = "Foo"

  character_settings = {
    character_set = "utf8mb4"
    collation     = "utf8mb4_general_ci"
  }

  user = {
    access_level    = "full"
    password        = var.database_password
    external_access = false
  }
}
