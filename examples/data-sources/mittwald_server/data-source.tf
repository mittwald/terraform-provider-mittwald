data "mittwald_server" "example" {
  id = "your-server-id"
}

output "server_machine_type" {
  value = data.mittwald_server.example.machine_type
}
