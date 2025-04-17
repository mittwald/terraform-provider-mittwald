locals {
  nginx_port = 80
}

resource "mittwald_container_stack" "nginx" {
  project_id    = mittwald_project.example.id
  default_stack = true

  containers = {
    nginx = {
      description = "Example web server"
      image       = "nginx:1.27.4"
      entrypoint  = ["/docker-entrypoint.sh"]

      // command = ["php -S 0.0.0.0:$PORT"]

      // environment = {
      //   FOO = "bar"
      // }

      ports = [
        {
          container_port = 80
          public_port    = local.nginx_port
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

resource "mittwald_virtualhost" "nginx" {
  hostname   = "${mittwald_project.test.short_id}.project.space"
  project_id = mittwald_project.test.id

  paths = {
    "/" = {
      container = {
        container_id = mittwald_container_stack.nginx.containers.nginx.id
        port         = "${local.nginx_port}/tcp"
      }
    }
  }
}
