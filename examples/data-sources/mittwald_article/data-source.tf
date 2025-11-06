data "mittwald_article" "vserver" {
  filter_tags      = ["vserver"]
  filter_orderable = ["full"]
  filter_attributes = {
    ram  = "8"
    vcpu = "2"
  }
}

/**
 * The mittwald_article data source can be used to select a hosting plan article
 * based on tags, template, orderable status, or specific attributes.
 *
 * The selected article's ID can then be used when creating projects or servers
 * to specify which hosting plan should be used.
 *
 * In this example, we're selecting a vServer article that is fully orderable
 * with 8 GB of RAM and 2 vCPUs, and using its ID, price, and attribute
 * information in outputs.
 */
output "selected_article_id" {
  description = "The ID of the selected vServer article"
  value       = data.mittwald_article.vserver.id
}

output "article_price" {
  description = "The monthly price of the selected article"
  value       = data.mittwald_article.vserver.price
}

output "article_orderable_status" {
  description = "The orderable status of the article"
  value       = data.mittwald_article.vserver.orderable
}

output "article_attributes" {
  description = "All attributes of the selected article"
  value       = data.mittwald_article.vserver.attributes
}
