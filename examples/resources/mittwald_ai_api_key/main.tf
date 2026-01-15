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

variable "customer_id" {
  type = string
}

resource "mittwald_project" "example" {
  server_id   = var.server_id
  description = "Example mittwald AI Project"
}
