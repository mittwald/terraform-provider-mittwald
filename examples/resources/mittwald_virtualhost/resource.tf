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

    "/api" = {
      container = {
        container_id = mittwald_container_stack.foobar.containers.api.id
        port         = "3000/tcp"
      }
    }

    "/redirect" = {
      redirect = "https://redirect.example"
    }

    "/default" = {
      // keep empty to redirect to empty default page
    }
  }
}
