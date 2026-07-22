resource "mittwald_project" "foobar" {
  server_id   = var.server_id
  description = "Test project"

  # A project's default ingress (and with it, the `default_ips` attribute) is
  # provisioned asynchronously; if this regularly takes longer than the default
  # timeouts, you can adjust them here.
  timeouts {
    create = "10m"
    read   = "2m"
  }
}

output "project_ips" {
  value = mittwald_project.foobar.default_ips
}
