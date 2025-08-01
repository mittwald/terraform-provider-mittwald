resource "random_password" "mailbox_password" {
  length           = 16
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

resource "mittwald_email_outbox" "example" {
  project_id  = "p-12345"
  description = "Example mail outbox"
  password    = resource.random_password.mailbox_password.result
}

output "email_outbox_id" {
  value = mittwald_email_outbox.example.id
}

output "email_outbox_name" {
  value = mittwald_email_outbox.example.name
}
