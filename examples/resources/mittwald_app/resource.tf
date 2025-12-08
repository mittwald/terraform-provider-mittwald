variable "admin_password" {
  description = "The password for the admin user of the wordpress app"
  type        = string
  sensitive   = true
}

data "mittwald_systemsoftware" "php" {
  name     = "php"
  selector = "^8.2"
}

data "mittwald_systemsoftware" "composer" {
  name        = "composer"
  recommended = true
}

data "mittwald_systemsoftware" "mysql" {
  name        = "mysql"
  recommended = true
}

resource "mittwald_app" "wordpress" {
  project_id = mittwald_project.foobar.id

  app     = "wordpress"
  version = "6.3.1"

  description   = "Martins Test-App"
  update_policy = "patchlevel"

  user_inputs = {
    "site_title"  = "My awesome site"
    "admin_user"  = "martin"
    "admin_pass"  = var.admin_password
    "admin_email" = "martin@mittwald.example"
  }
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
