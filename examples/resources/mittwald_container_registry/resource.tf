variable "registry_credentials" {
  sensitive = true
  type = object({
    username = string
    password = string
  })
}

resource "mittwald_container_registry" "custom_registry" {
  project_id  = mittwald_project.test.id
  description = "My custom registry"
  uri         = "registry.company.example"

  credentials = {
    username = var.registry_credentials.username
    password = var.registry_credentials.password
  }
}
