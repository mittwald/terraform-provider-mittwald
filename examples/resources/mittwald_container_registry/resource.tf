variable "registry_credentials" {
  sensitive = true
  type = object({
    username         = string
    password         = string
    password_version = number
  })
}

resource "mittwald_container_registry" "custom_registry" {
  project_id  = mittwald_project.test.id
  description = "My custom registry"
  uri         = "registry.company.example"

  credentials = {
    username = var.registry_credentials.username

    // password_wo is a write-only attribute, which will not be persisted
    // in the state file. You will need to increase password_wo_version
    // whenever the password changes.
    password_wo         = var.registry_credentials.password
    password_wo_version = var.registry_credentials.password_version
  }
}
