/*
This example deploys a custom PHP app.

The PHP version is selected dynamically using a semver constraint.
The app installation is also linked with a MySQL database.

Note that this example will also continuously _update_ your PHP
environment instance on the 8.* branch.
*/

data "mittwald_systemsoftware" "php" {
  name     = "php"
  selector = "^8.5"
}

data "mittwald_systemsoftware" "composer" {
  name        = "composer"
  recommended = true
}

data "mittwald_systemsoftware" "mysql" {
  name        = "mysql"
  recommended = true
}

resource "mittwald_app" "custom_php" {
  project_id = mittwald_project.foobar.id

  app     = "php"
  version = "1.0.0"

  description   = "Martins Test-App"
  document_root = "/public"
  update_policy = "none"

  databases = [
    {
      kind    = "mysql"
      purpose = "primary"
      id      = mittwald_mysql_database.foobar_database.id
      user_id = mittwald_mysql_database.foobar_database.user.id
    }
  ]

  dependencies = {
    (data.mittwald_systemsoftware.php.name) = {
      version       = data.mittwald_systemsoftware.php.version
      update_policy = "patchLevel"
    }
    (data.mittwald_systemsoftware.composer.name) = {
      version       = data.mittwald_systemsoftware.composer.version
      update_policy = "patchLevel"
    },
    (data.mittwald_systemsoftware.mysql.name) = {
      version       = data.mittwald_systemsoftware.mysql.version
      update_policy = "patchLevel"
    },
  }
}
