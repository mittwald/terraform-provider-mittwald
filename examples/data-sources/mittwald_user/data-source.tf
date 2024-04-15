data "mittwald_user" "self" {}

/**
 * When provisioning an app, you can user the `mittwald_user` data source to
 * get the email of the user that is currently authenticated.
 *
 * You can then use this email to configure the SSH connection that should be
 * used to provision the app.
 */
resource "null_resource" "provisioning" {
  connection {
    type        = "ssh"
    host        = mittwald_app.test.ssh_host
    user        = "${data.mittwald_user.self.email}@${mittwald_app.test.short_id}"
    private_key = file("/Users/mhelmich/.ssh/id_rsa")
  }

  provisioner "file" {
    source      = "app/config.php"
    destination = "${mittwald_app.test.installation_path_absolute}/config.php"
  }
}
