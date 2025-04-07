data "mittwald_app" "typo3_13" {
  name     = "typo3"
  selector = "13.*"
}

resource "mittwald_app" "example" {
  project_id = mittwald_project.example.id

  app     = data.mittwald_app.typo3_13.name
  version = data.mittwald_app.typo3_13.version
}
