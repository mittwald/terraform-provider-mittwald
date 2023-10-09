variable "mittwald_api_key" {
  type      = string
  sensitive = true
}

provider "mittwald" {
  # NOTE: You can also use the environment variable MITTWALD_API_TOKEN, instead.
  # In this case, you don't need to specify the api_key variable.
  api_key = var.mittwald_api_key
}
