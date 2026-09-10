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
  name        = "Example AI hosting plan"

  use_free_trial = true
}

# A customer may have several AI hosting plans; each mittwald_ai resource
# manages one of them.
resource "mittwald_ai" "second_plan" {
  customer_id = var.customer_id
  article_id  = data.mittwald_article.ai_starter.id
  name        = "Second AI hosting plan"
}
