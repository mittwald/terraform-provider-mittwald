/**
 * A project is either provisioned on an existing server, in which case only a
 * `server_id` is needed...
 */
resource "mittwald_project" "foobar" {
  server_id   = var.server_id
  description = "Test project"
}

output "project_ips" {
  value = mittwald_project.foobar.default_ips
}

/**
 * ... or ordered as a stand-alone project, which is billed for a customer. In
 * this case, the hosting plan is selected by article.
 */
data "mittwald_article" "project" {
  filter = {
    tags      = ["webhosting"]
    orderable = ["full"]
    attributes = {
      ram  = "1"
      vcpu = "1"
    }
  }
}

resource "mittwald_project" "standalone" {
  customer_id = var.customer_id
  article_id  = data.mittwald_article.project.id

  description  = "Test project"
  diskspace_gb = 20

  use_free_trial = true
}
