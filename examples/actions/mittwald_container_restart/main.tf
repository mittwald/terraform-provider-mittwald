terraform {
  required_providers {
    mittwald = {
      source  = "mittwald/mittwald"
      version = "~> 1.5.0"
    }
  }
}

variable "project_id" {
  type = string
}

resource "mittwald_container_stack" "nginx" {
  project_id    = var.project_id
  default_stack = true

  containers = {
    nginx = {
      description = "Example web server"
      image       = "nginx:1.27.4"

      entrypoint = ["/docker-entrypoint.sh"]
      command    = ["nginx", "-g", "daemon off;"]

      ports = [
        {
          container_port = 80
          public_port    = 80
          protocol       = "tcp"
        }
      ]

      volumes = [
        {
          project_path = "/files/nginx/conf.d"
          mount_path   = "/etc/nginx/conf.d"
        }
      ]
    }
  }
}