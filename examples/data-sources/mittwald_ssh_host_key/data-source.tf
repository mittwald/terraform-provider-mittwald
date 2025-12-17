# Fetch the SSH host key from a mittwald app's SSH host
data "mittwald_ssh_host_key" "main" {
  hostname = mittwald_app.api.ssh_host
}

# Use the host key in a Bitbucket known host configuration
resource "bitbucket_pipeline_ssh_known_host" "mittwald" {
  workspace  = "my-workspace"
  repository = "my-repo"
  hostname   = "[${mittwald_app.api.ssh_host}]:22"

  public_key {
    key_type = data.mittwald_ssh_host_key.main.key_type
    key      = data.mittwald_ssh_host_key.main.key
  }
}

# Output the fingerprint for verification
output "ssh_host_fingerprint" {
  value = data.mittwald_ssh_host_key.main.fingerprint
}
