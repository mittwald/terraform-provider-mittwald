# Create an SSH user with public key authentication
resource "mittwald_ssh_user" "deploy" {
  project_id  = mittwald_project.example.id
  description = "Deployment SSH user"

  public_keys = [
    {
      key     = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExample..."
      comment = "deploy@example.com"
    }
  ]
}

# Create an SSH user with password authentication
resource "mittwald_ssh_user" "admin" {
  project_id  = mittwald_project.example.id
  description = "Admin SSH user"
  password    = var.ssh_admin_password
}

# Create an SSH user with expiration date
resource "mittwald_ssh_user" "temporary" {
  project_id  = mittwald_project.example.id
  description = "Temporary access for contractor"
  expires_at  = "2024-12-31T23:59:59Z"

  public_keys = [
    {
      key     = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQExample..."
      comment = "contractor@company.com"
    }
  ]
}

# Output the generated username for use in CI/CD pipelines
output "deploy_ssh_username" {
  value = mittwald_ssh_user.deploy.username
}
