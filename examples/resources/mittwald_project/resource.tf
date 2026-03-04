resource "mittwald_project" "foobar" {
  server_id   = var.server_id
  description = "Test project"
}

output "project_ips" {
  value = mittwald_project.foobar.default_ips
}
