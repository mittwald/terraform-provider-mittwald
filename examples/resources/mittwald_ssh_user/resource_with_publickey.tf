# Create an SSH user with public key authentication
resource "mittwald_ssh_user" "deploy" {
  project_id  = mittwald_project.example.id
  description = "Deployment SSH user"

  public_keys = [
    {
      key     = provider::mittwald::read_ssh_publickey("~/.ssh/id_rsa.pub")
      comment = "deploy@example.com"
    }
  ]
}

# Output the generated username for use in CI/CD pipelines
output "deploy_ssh_username" {
  value = mittwald_ssh_user.deploy.username
}
