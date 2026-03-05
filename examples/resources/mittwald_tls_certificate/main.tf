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

variable "project_id" {
  type = string
}
