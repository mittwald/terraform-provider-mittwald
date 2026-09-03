/*
This example deploys a managed WordPress instance.

The admin password is defined as a sensitive input variable, and the
version is selected dynamically using a semver constraint.

Note that this example will also continuously _update_ your WordPress
instance on the 7.* branch.
*/

variable "admin_password" {
  description = "The password for the admin user of the wordpress app"
  type        = string
  sensitive   = true
}

data "mittwald_app" "wordpress_7" {
  name     = "wordpress"
  selector = "7.*"
}

resource "mittwald_app" "wordpress" {
  project_id = mittwald_project.foobar.id

  app     = data.mittwald_app.wordpress_7.name
  version = data.mittwald_app.wordpress_7.version

  description   = "Martins Test-App"
  update_policy = "patchlevel"

  user_inputs = {
    "site_title"  = "My awesome site"
    "admin_user"  = "martin"
    "admin_pass"  = var.admin_password
    "admin_email" = "martin@mittwald.example"
  }
}