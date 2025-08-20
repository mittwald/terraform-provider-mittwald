resource "mittwald_remote_file" "example" {
  container_id = "<container-id>" # either container_id or app_id
  # app_id = "<app-id>"            # either container_id or app_id

  # Optional: SSH username (defaults to "<email>@<container-short-id>" if omitted)
  # ssh_user = "ssh-XXXXX"

  path     = "/path/to/file.txt"
  contents = "This is the content of the file"

  # Alternatively, you can use the file function to read content from a local file
  # contents = file("${path.module}/local_file.txt")
}
