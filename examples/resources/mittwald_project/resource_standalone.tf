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
