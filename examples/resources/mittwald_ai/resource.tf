data "mittwald_article" "ai_starter" {
  filter = {
    id = "AI25-*"
    attributes = {
      category = "Starter"
    }
  }
}

resource "mittwald_ai" "example" {
  customer_id = var.customer_id
  article_id  = data.mittwald_article.ai_starter.id

  use_free_trial = true
}
