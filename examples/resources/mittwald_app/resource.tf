variable "admin_password" {
  description = "The password for the admin user of the wordpress app"
  type        = string
  sensitive   = true
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
  project_id  = mittwald_project.foobar.id
  database_id = mittwald_mysql_database.foobar_database.id

  app     = "php"
  version = "1.0.0"

  description   = "Martins Test-App"
  document_root = "/public"
  update_policy = "none"

  dependencies = {
    "php" = {
      version = "8.2.8"
      update_policy = "patchLevel"
    }
    "composer" = {
      update_policy = "patchLevel"
      version       = "2.3.10"
    },
    "mysql" = {
      update_policy = "patchLevel"
      version       = "8.0.28"
    },
  }
}
