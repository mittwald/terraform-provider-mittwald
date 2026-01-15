resource "mittwald_ai_api_key" "example_for_project" {
  project_id = mittwald_project.example.id
  name       = "my-project-api-key"
}
