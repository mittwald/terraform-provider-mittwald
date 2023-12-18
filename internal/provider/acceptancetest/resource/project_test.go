package resource

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/mittwald/terraform-provider-mittwald/api/mittwaldv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providertesting"
	"os"
	"testing"
	"time"
)

func TestAccProjectResourceCreated(t *testing.T) {
	var project mittwaldv2.DeMittwaldV1ProjectProject

	serverID := config.StringVariable(os.Getenv("MITTWALD_ACCTEST_SERVER_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			providertesting.TestAccPreCheck(t)
		},
		ProtoV6ProviderFactories: providertesting.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectResourceConfig("Foobar"),
				ConfigVariables: map[string]config.Variable{
					"server_id": serverID,
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mittwald_project.test", "description", "Foobar"),
					resource.TestCheckResourceAttrWith("mittwald_project.test", "id", providertesting.MatchUUID),
					testAccAssertProjectIsPresent("mittwald_project.test", &project),
					testAccAssertProjectDescriptionMatches(&project, "Foobar"),
				),
			},
			{
				Config: testAccProjectResourceConfig("Barbaz"),
				ConfigVariables: map[string]config.Variable{
					"server_id": serverID,
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mittwald_project.test", "description", "Barbaz"),
					resource.TestCheckResourceAttrWith("mittwald_project.test", "id", providertesting.MatchUUID),
					testAccAssertProjectIsPresent("mittwald_project.test", &project),
					testAccAssertProjectDescriptionMatches(&project, "Barbaz"),
				),
			},
		},
		CheckDestroy: testAccProjectResourceDestroyed,
	})
}

func testAccProjectResourceConfig(desc string) string {
	return fmt.Sprintf(`
variable "server_id" {
  type = string
}

resource "mittwald_project" "test" {
	server_id = var.server_id
	description = "%[1]s"
}
`, desc)
}

func testAccProjectResourceDestroyed(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "mittwald_project" {
			continue
		}

		if err := testAccAssertProjectIsAbsent(rs.Primary.ID); err != nil {
			return err
		}
	}

	return nil
}

func testAccAssertProjectIsPresent(resourceName string, out *mittwaldv2.DeMittwaldV1ProjectProject) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found", resourceName)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		project, err := providertesting.TestClient().Project().PollProject(ctx, rs.Primary.ID)
		if err != nil {
			return err
		}

		*out = *project
		return nil
	}
}

func testAccAssertProjectDescriptionMatches(project *mittwaldv2.DeMittwaldV1ProjectProject, desc string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if project.Description != desc {
			return fmt.Errorf("expected project description to be '%s', got %s", desc, project.Description)
		}

		return nil
	}
}

func testAccAssertProjectIsAbsent(projectID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := providertesting.TestClient().Project().PollProject(ctx, projectID)
	if mittwaldv2.IsNotFound(err) {
		return nil
	}

	return fmt.Errorf("expected project %s to return ErrNotFound, but got %s", projectID, err)
}
