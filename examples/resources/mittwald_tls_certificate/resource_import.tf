variable "project_id" {
  type        = string
  description = "The ID of the mittwald project"
}

# Import a pre-existing PEM-encoded TLS certificate and private key.
# This is useful when you already have a certificate (e.g. from your own CA or
# a third-party certificate authority) and want to use it on mittwald.
#
# Note: common_name is optional for certificate import; when omitted it is
# derived automatically from the certificate's CN.
resource "mittwald_tls_certificate" "imported" {
  project_id = var.project_id

  # PEM-encoded certificate (can be read from a file)
  certificate = file("${path.module}/cert.pem")

  # The private key is a write-only attribute. To trigger an in-place renewal,
  # update the certificate and increment private_key_wo_version.
  private_key_wo         = file("${path.module}/key.pem")
  private_key_wo_version = 1
}

# The virtual host will automatically use the imported certificate.
# Use depends_on to ensure the certificate is ready before the virtual host
# is created.
resource "mittwald_virtualhost" "example" {
  project_id = var.project_id
  hostname   = "foo.example"

  paths = {
    "/" = {}
  }

  depends_on = [mittwald_tls_certificate.imported]
}
