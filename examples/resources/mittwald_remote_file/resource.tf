resource "mittwald_remote_file" "example" {
  container_id = "<container-id>" # either container_id or app_id
  stack_id     = "<stack-id>"     # required when container_id is specified
  # app_id = "<app-id>"            # either container_id or app_id

  # Optional: SSH username (defaults to "<email>@<container-short-id>" if omitted)
  # ssh_user = "ssh-XXXXX"
  # ssh_private_key = file("~/.ssh/id_rsa")

  path     = "/path/to/file.txt"
  contents = "This is the content of the file"

  # Alternatively, you can use the file function to read content from a local file
  # contents = file("${path.module}/local_file.txt")

  # Alternatively, use the contents_from_url attribute to fetch content from a URL
  # contents_from_url = "https://example.com/file.txt"
}
