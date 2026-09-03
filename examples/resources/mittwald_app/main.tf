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

resource "mittwald_project" "foobar" {
  server_id   = var.server_id
  description = "Test project"
}

ephemeral "mittwald_mysql_password" "password" {
  length = 24
}

resource "mittwald_mysql_database" "foobar_database" {
  project_id  = mittwald_project.foobar.id
  version     = "8.4"
  description = "Foo"

  character_settings = {
    character_set = "utf8mb4"
    collation     = "utf8mb4_general_ci"
  }

  user = {
    access_level        = "full"
    password_wo         = ephemeral.mittwald_mysql_password.password.password
    password_wo_version = 1
    external_access     = false
  }
}
