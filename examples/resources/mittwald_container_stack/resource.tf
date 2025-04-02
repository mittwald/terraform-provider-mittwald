resource "mittwald_container_stack" "nginx" {
  project_id    = mittwald_project.example.id
  default_stack = true

  containers = {
    foo = {
      description = "Example web server"
      image       = "nginx:1.27.4"
      entrypoint  = ["/docker-entrypoing.sh"]

      // command = ["php -S 0.0.0.0:$PORT"]

      // environment = {
      //   FOO = "bar"
      // }

      ports = [
        {
          container_port = 80
          public_port    = 80
          protocol       = "tcp"
        }
      ]

      volumes = [
        {
          project_path = "/html"
          mount_path   = "/usr/share/nginx/html"
        }
      ]
    }
  }

  volumes = {
    example = {

    }
  }
}
