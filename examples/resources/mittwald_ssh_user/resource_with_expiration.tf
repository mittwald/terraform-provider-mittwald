# Create an SSH user with expiration date
resource "mittwald_ssh_user" "temporary" {
  project_id  = mittwald_project.example.id
  description = "Temporary access for contractor"
  expires_at  = "2026-12-31T23:59:59Z"

  public_keys = [
    {
      key     = provider::mittwald::read_ssh_publickey("~/.ssh/id_rsa.pub")
      comment = "contractor@company.com"
    }
  ]
}
