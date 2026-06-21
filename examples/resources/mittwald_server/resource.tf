data "mittwald_article" "server" {
  filter = {
    tags = ["vserver", "balance-optimized"]
    attributes = {
      "ram"  = "2"
      "vcpu" = "1"
    }
  }
}

resource "mittwald_server" "example" {
  customer_id = var.customer_id
  article_id  = data.mittwald_article.server.id

  description  = "My first server"
  diskspace_gb = 50

  use_free_trial = true
}
