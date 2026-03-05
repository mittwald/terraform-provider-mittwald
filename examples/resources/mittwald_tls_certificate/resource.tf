variable "project_id" {
  type        = string
  description = "The ID of the mittwald project"
}

# Create a wildcard TLS certificate using DNS validation.
# This is required for wildcard domains (e.g. *.foobar.example) since
# wildcard certificates cannot be created implicitly with a virtual host.
resource "mittwald_tls_certificate" "wildcard_example" {
  project_id  = var.project_id
  common_name = "*.foobar.example"
}

# The virtual host will automatically use the wildcard certificate once
# the certificate has been provisioned. Use depends_on to ensure the
# certificate is ready before the virtual host is created.
resource "mittwald_virtualhost" "example" {
  project_id = var.project_id
  hostname   = "app.foobar.example"

  paths = {
    "/" = {}
  }

  depends_on = [mittwald_tls_certificate.wildcard_example]
}
