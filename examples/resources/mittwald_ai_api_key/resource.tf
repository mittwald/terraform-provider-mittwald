resource "mittwald_ai_api_key" "example" {
  customer_id = var.customer_id
  name        = "my-api-key"
}

data "mittwald_article" "ai_starter" {
  filter = {
    id = "AI25-*"
    attributes = {
      category = "Starter"
    }
  }
}

resource "mittwald_ai" "second_plan" {
  customer_id = var.customer_id
  article_id  = data.mittwald_article.ai_starter.id
  name        = "Second AI hosting plan"
}

# If the customer has more than one AI hosting plan, contract_id must be set
# to disambiguate which plan the key should be scoped to.
resource "mittwald_ai_api_key" "scoped_example" {
  customer_id = var.customer_id
  contract_id = mittwald_ai.second_plan.contract_id
  name        = "my-scoped-api-key"
}
