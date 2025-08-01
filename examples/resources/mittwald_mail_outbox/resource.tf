resource "mittwald_mail_outbox" "example" {
  project_id  = "p-12345"
  description = "Example mail outbox"
  password    = "SecurePassword123!"
}

# Output the mail outbox ID
output "mail_outbox_id" {
  value = mittwald_mail_outbox.example.id
}
