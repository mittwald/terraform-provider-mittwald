resource "mittwald_cronjob" "demo" {
  project_id = mittwald_project.foobar.id
  app_id     = mittwald_app.wordpress.id

  interval    = "*/5 * * * *"
  description = "Demo Cronjob"

  destination = {
    command = {
      interpreter = "php"
      path        = "/html"
      arguments   = ["-r", "echo 'Hello World';"]
    }
  }
}
