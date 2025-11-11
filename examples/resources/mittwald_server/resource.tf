variable "customer_id" {
  type = string
}

resource "mittwald_server" "example" {
  customer_id = var.customer_id
  description = "Test server"

  volume_size = 100
  machine_type = {
    name = "shared.xlarge"
  }

  use_free_trial = true
}
