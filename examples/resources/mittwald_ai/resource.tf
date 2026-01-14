variable "customer_id" {
  type = string
}

resource "mittwald_ai" "example" {
  customer_id = var.customer_id
  article_id  = "AI25-0001"

  use_free_trial = true
}
