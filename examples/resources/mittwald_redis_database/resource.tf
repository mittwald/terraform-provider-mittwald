resource "mittwald_redis_database" "foobar_database" {
  project_id  = mittwald_project.foobar.id
  version     = "7.2"
  description = "Foo"

  configuration = {
    max_memory_mb     = 256
    max_memory_policy = "allkeys-lru"
    persistent        = true
  }
}
