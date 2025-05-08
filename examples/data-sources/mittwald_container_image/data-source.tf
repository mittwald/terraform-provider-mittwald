data "mittwald_container_image" "nginx" {
  image = "nginx:1.28.0"
}

resource "mittwald_container_stack" "nginx" {
  project_id    = mittwald_project.test.id
  default_stack = true

  containers = {
    nginx = {
      description = "Example web server"
      image       = data.mittwald_container_image.nginx.image
      entrypoint  = data.mittwald_container_image.nginx.entrypoint
      command     = data.mittwald_container_image.nginx.command

      // ...
    }
  }
}
