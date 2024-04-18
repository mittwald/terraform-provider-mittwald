resource "mittwald_cronjob" "demo" {
  project_id = mittwald_project.foobar.id
  app_id     = mittwald_app.wordpress.id

  interval    = "*/5 * * * *"
  description = "Demo Cronjob"

  destination = {
    command = {
      interpreter = "/usr/bin/php"
      path        = "/html"
      parameters  = ["-r", "echo 'Hello World';"]
    }
  }
}
