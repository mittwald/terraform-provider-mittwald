variable "server_id" {
  type = string
}

resource "mittwald_virtualhost" "foobar" {
  hostname   = "test.example"
  project_id = mittwald_project.foobar.id

  paths = {
    "/" = {
      app = mittwald_app.foobar.id
    }

    "/redirect" = {
      redirect = "https://redirect.example"
    }
  }
}
