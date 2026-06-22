# Look up an existing project by its short ID...
data "mittwald_project" "by_short_id" {
  short_id = "p-abcdef"
}

# ...or by its full ID.
data "mittwald_project" "by_id" {
  id = "f0596955-cf90-4ba7-a0a5-32b40240e0c1"
}

/**
 * The data source exposes the same attributes as the `mittwald_project`
 * resource. This is useful for referencing a project that is not managed by
 * this Terraform configuration -- for example to attach a virtual host to the
 * project's default IP addresses.
 */
resource "mittwald_virtualhost" "example" {
  project_id = data.mittwald_project.by_short_id.id
  hostname   = "www.example.com"

  paths = {
    "/" = {
      app = mittwald_app.example.id
    }
  }
}
