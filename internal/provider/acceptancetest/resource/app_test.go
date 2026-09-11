package resource

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/appclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/appv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiutils"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providertesting"
)

// TestAccAppResourceInstallationPath verifies that a user-provided
// installation_path is passed through to the API on creation, that it is
// reflected back in installation_path_absolute, and that changing it forces
// replacement of the app resource (since the API only accepts
// installation_path on create).
func TestAccAppResourceInstallationPath(t *testing.T) {
	var app appv2.AppInstallation

	serverID := config.StringVariable(os.Getenv("MITTWALD_ACCTEST_SERVER_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			providertesting.TestAccPreCheck(t)
		},
		ProtoV6ProviderFactories: providertesting.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAppResourceInstallationPathConfig("Test Static App", "custom-install-path"),
				ConfigVariables: map[string]config.Variable{
					"server_id": serverID,
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mittwald_app.test", "installation_path", "custom-install-path"),
					resource.TestCheckResourceAttrWith("mittwald_app.test", "installation_path_absolute", func(value string) error {
						if value == "" {
							return fmt.Errorf("expected installation_path_absolute to be set")
						}
						return nil
					}),
					testAccAssertAppInstallationPathMatches("mittwald_app.test", &app, "custom-install-path"),
				),
			},
			// Changing installation_path must force replacement, since the API
			// only accepts it on create.
			{
				Config: testAccAppResourceInstallationPathConfig("Test Static App", "other-install-path"),
				ConfigVariables: map[string]config.Variable{
					"server_id": serverID,
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("mittwald_app.test", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mittwald_app.test", "installation_path", "other-install-path"),
					testAccAssertAppInstallationPathMatches("mittwald_app.test", &app, "other-install-path"),
				),
			},
		},
	})
}

func testAccAppResourceInstallationPathConfig(appDesc, installationPath string) string {
	return fmt.Sprintf(`
variable "server_id" {
  type = string
}

resource "mittwald_project" "test" {
	server_id   = var.server_id
	description = "terraform_app_installation_path_test"
}

resource "mittwald_app" "test" {
	project_id         = mittwald_project.test.id
	description        = "%[1]s"
	app                = "static"
	version            = "1.0.0"
	update_policy      = "none"
	installation_path  = "%[2]s"
}
`, appDesc, installationPath)
}

func testAccAssertAppInstallationPathMatches(resourceName string, out *appv2.AppInstallation, expectedPath string) resource.TestCheckFunc {
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

		if app.InstallationPath != expectedPath {
			return fmt.Errorf("expected installation path to be '%s', got '%s'", expectedPath, app.InstallationPath)
		}

		*out = *app
		return nil
	}
}
