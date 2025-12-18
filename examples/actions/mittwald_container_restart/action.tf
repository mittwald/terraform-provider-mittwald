// In this example, we define an action to restart the Nginx container
// whenever the Nginx configuration file is updated.

action "mittwald_container_restart" "restart_nginx" {
  config {
    container_id = mittwald_container_stack.nginx.containers.nginx.id
    stack_id     = mittwald_container_stack.nginx.id
  }
}

resource "mittwald_remote_file" "nginx_config" {
  container_id = mittwald_container_stack.nginx.containers.nginx.id
  stack_id     = mittwald_container_stack.nginx.id

  path     = "/etc/nginx/conf.d/default.conf"
  contents = file("${path.module}/nginx.conf")
  depends_on = [
    mittwald_container_stack.nginx
  ]

  lifecycle {
    action_trigger {
      events  = [after_create, after_update]
      actions = [action.mittwald_container_restart.restart_nginx]
    }
  }
}
