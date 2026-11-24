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
}

# contract_id scopes the key to a specific AI hosting plan. It may be omitted
# if the customer has exactly one AI hosting plan; if the customer has more
# than one, it is required.
resource "mittwald_ai_api_key" "example" {
  customer_id = var.customer_id
  contract_id = mittwald_ai.example.contract_id
  name        = "my-api-key"
}
