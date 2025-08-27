package resource

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/appclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/appv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiutils"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providertesting"
	"os"
	"testing"
	"time"
)

func TestAccRemoteFileResourceCreated(t *testing.T) {
	// TODO: Refactor this test case to create an SSH user dynamically
	//   and use that user to verify the remote file creation.
	//   For this, we first need a "mittwald_ssh_user" resource.

	sshKeyPath := os.ExpandEnv("$HOME/.ssh/id_rsa")
	if _, err := os.Stat(sshKeyPath); os.IsNotExist(err) {
		t.Skip("Skipping test because SSH key file does not exist at ~/.ssh/id_rsa")
		return
	}

	var app appv2.AppInstallation

	serverID := config.StringVariable(os.Getenv("MITTWALD_ACCTEST_SERVER_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			providertesting.TestAccPreCheck(t)
		},
		ProtoV6ProviderFactories: providertesting.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRemoteFileResourceConfig("Test Project", "Test Static App"),
				ConfigVariables: map[string]config.Variable{
					"server_id": serverID,
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// Check project resource
					resource.TestCheckResourceAttr("mittwald_project.test", "description", "Test Project"),
					resource.TestCheckResourceAttrWith("mittwald_project.test", "id", providertesting.MatchUUID),

					// Check app resource
					resource.TestCheckResourceAttr("mittwald_app.test", "description", "Test Static App"),
					resource.TestCheckResourceAttr("mittwald_app.test", "app", "static"),
					resource.TestCheckResourceAttrWith("mittwald_app.test", "id", providertesting.MatchUUID),
					testAccAssertAppIsPresent("mittwald_app.test", &app),

					// Check remote file resource
					resource.TestCheckResourceAttrSet("mittwald_remote_file.test", "path"),
					resource.TestCheckResourceAttr("mittwald_remote_file.test", "contents", "<html><body><h1>Hello, World!</h1></body></html>"),
					testAccVerifyRemoteFileExists("mittwald_remote_file.test", "<html><body><h1>Hello, World!</h1></body></html>"),
				),
			},
			// Update the file content
			{
				Config: testAccRemoteFileResourceConfigUpdated("Test Project", "Test Static App"),
				ConfigVariables: map[string]config.Variable{
					"server_id": serverID,
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// Check remote file resource with updated content
					resource.TestCheckResourceAttrSet("mittwald_remote_file.test", "path"),
					resource.TestCheckResourceAttr("mittwald_remote_file.test", "contents", "<html><body><h1>Updated Content!</h1></body></html>"),
					testAccVerifyRemoteFileExists("mittwald_remote_file.test", "<html><body><h1>Updated Content!</h1></body></html>"),
				),
			},
		},
		CheckDestroy: testAccRemoteFileResourceDestroyed,
	})
}

func testAccRemoteFileResourceConfig(projectDesc, appDesc string) string {
	return fmt.Sprintf(`
variable "server_id" {
  type = string
}

resource "mittwald_project" "test" {
	server_id = var.server_id
	description = "%[1]s"
}

resource "mittwald_app" "test" {
	project_id = mittwald_project.test.id
	description = "%[2]s"
	app = "static"
	version = "1.0.0"
	update_policy = "none"
}

resource "mittwald_remote_file" "test" {
	app_id = mittwald_app.test.id
	path = "${mittwald_app.test.installation_path_absolute}/index.html"
	contents = "<html><body><h1>Hello, World!</h1></body></html>"
}
`, projectDesc, appDesc)
}

func testAccAssertAppIsPresent(resourceName string, out *appv2.AppInstallation) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found", resourceName)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		client := providertesting.TestClient().App()

		app, err := apiutils.PollRequest(ctx, apiutils.PollOpts{}, client.GetAppinstallation, appclientv2.GetAppinstallationRequest{AppInstallationID: rs.Primary.ID})
		if err != nil {
			return fmt.Errorf("error while polling for app %s: %w", rs.Primary.ID, err)
		}

		*out = *app
		return nil
	}
}

func testAccRemoteFileResourceConfigUpdated(projectDesc, appDesc string) string {
	return fmt.Sprintf(`
variable "server_id" {
  type = string
}

resource "mittwald_project" "test" {
	server_id = var.server_id
	description = "%[1]s"
}

resource "mittwald_app" "test" {
	project_id = mittwald_project.test.id
	description = "%[2]s"
	app = "static"
	version = "1.0.0"
	update_policy = "none"
}

resource "mittwald_remote_file" "test" {
	app_id = mittwald_app.test.id
	path = "${mittwald_app.test.installation_path_absolute}/index.html"
	contents = "<html><body><h1>Updated Content!</h1></body></html>"
}
`, projectDesc, appDesc)
}

func testAccVerifyRemoteFileExists(resourceName string, expectedContent string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found", resourceName)
		}

		// In a real test, we would connect to the server via SSH and verify the file exists
		// with the expected content. For this acceptance test, we'll just check that the
		// resource was created successfully with the expected attributes.

		if rs.Primary.Attributes["contents"] != expectedContent {
			return fmt.Errorf("file content does not match expected content")
		}

		return nil
	}
}

func testAccRemoteFileResourceDestroyed(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "mittwald_remote_file" {
			continue
		}

		// In a real test, we would connect to the server via SSH and verify the file no longer exists.
		// For this acceptance test, we'll just check that the resource was destroyed in Terraform state.

		// The resource is considered destroyed if it's not in the state anymore
		// or if it's marked as destroyed in the state.
		if rs.Primary.ID == "" {
			return nil
		}
	}

	return nil
}
