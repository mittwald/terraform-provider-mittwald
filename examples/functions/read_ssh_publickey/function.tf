# Read an SSH public key from a file
output "ssh_key" {
  value = provider::mittwald::read_ssh_publickey("~/.ssh/id_rsa.pub")
}

# Use it with an SSH user resource
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
