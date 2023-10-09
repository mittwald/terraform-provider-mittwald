variable "server_id" {
  type = string
}

resource "mittwald_project" "foobar" {
  server_id   = var.server_id
  description = "Test project"
}
