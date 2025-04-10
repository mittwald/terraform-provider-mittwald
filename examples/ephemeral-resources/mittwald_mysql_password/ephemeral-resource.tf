ephemeral "mittwald_mysql_password" "password" {
  length = 24
}

resource "mittwald_mysql_database" "test" {
  project_id  = mittwald_project.test.id
  description = "Test"
  version     = "8.4"
  user = {
    password_wo     = ephemeral.mittwald_mysql_password.password.password
    access_level    = "full"
    external_access = false
  }
}
