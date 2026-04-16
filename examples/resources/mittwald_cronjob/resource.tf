resource "mittwald_cronjob" "demo" {
  project_id = mittwald_project.foobar.id

  container = {
    stack_id   = mittwald_container_stack.example.id
    service_id = "nginx"
  }

  interval    = "*/5 * * * *"
  description = "Demo Cronjob"
  timezone    = "Europe/Berlin"

  destination = {
    container_command = ["echo", "Hello World"]
  }
}
