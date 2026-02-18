terraform {
  required_providers {
    mittwald = {
      source  = "mittwald/mittwald"
      version = ">= 1.0.0, < 2.0.0"
    }
  }
}

provider "mittwald" {
}

variable "server_id" {
  type = string
}

resource "mittwald_project" "example" {
  description = "terraform-provider ssh_user"
  server_id   = var.server_id
}
