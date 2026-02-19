# Create an SSH user with password authentication

variable "ssh_admin_password" {
  sensitive = true
  type      = string
}

resource "mittwald_ssh_user" "admin" {
  project_id          = mittwald_project.example.id
  description         = "Admin SSH user"
  password_wo         = var.ssh_admin_password
  password_wo_version = 1
}
