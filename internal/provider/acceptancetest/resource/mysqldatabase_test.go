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

func TestAccMySQLDatabaseResourceCreated(t *testing.T) {
	var database mittwaldv2.DeMittwaldV1DatabaseMySqlDatabase
	var user mittwaldv2.DeMittwaldV1DatabaseMySqlUser

	serverID := config.StringVariable(os.Getenv("MITTWALD_ACCTEST_SERVER_ID"))
	databasePassword := config.StringVariable(providertesting.TestRandomPassword(t))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			providertesting.TestAccPreCheck(t)
		},
		ProtoV6ProviderFactories: providertesting.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMySQLDatabaseResourceConfig("Foobar"),
				ConfigVariables: map[string]config.Variable{
					"server_id":         serverID,
					"database_password": databasePassword,
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mittwald_mysql_database.test", "description", "Foobar"),
					resource.TestCheckResourceAttrWith("mittwald_mysql_database.test", "id", providertesting.MatchUUID),
					testAccAssertMySQLDatabaseIsPresent("mittwald_mysql_database.test", &database, &user),
					testAccAssertMySQLDatabaseDescriptionMatches(&database, "Foobar"),
					testAccAssertMySQLUsernameMatchesState("mittwald_mysql_database.test", &user),
				),
			},
			{
				Config: testAccMySQLDatabaseResourceConfig("Barbaz"),
				ConfigVariables: map[string]config.Variable{
					"server_id":         serverID,
					"database_password": databasePassword,
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mittwald_mysql_database.test", "description", "Barbaz"),
					resource.TestCheckResourceAttrWith("mittwald_mysql_database.test", "id", providertesting.MatchUUID),
					testAccAssertMySQLDatabaseIsPresent("mittwald_mysql_database.test", &database, &user),
					testAccAssertMySQLDatabaseDescriptionMatches(&database, "Barbaz"),
				),
			},
		},
		CheckDestroy: testAccDatabaseResourceDestroyed,
	})
}

func testAccMySQLDatabaseResourceConfig(desc string) string {
	return fmt.Sprintf(`
variable "server_id" {
  type = string
}

variable "database_password" {
  type = string
  sensitive = true
}

resource "mittwald_project" "test" {
	server_id = var.server_id
	description = "terraform_mysqldatabase_test"
}

resource "mittwald_mysql_database" "test" {
  project_id  = mittwald_project.test.id
  version     = "8.0"
  description = "%[1]s"

  character_settings = {
    character_set = "utf8mb4"
    collation     = "utf8mb4_general_ci"
  }

  user = {
    access_level    = "full"
    password        = var.database_password
    external_access = false
  }
}
`, desc)
}

func testAccDatabaseResourceDestroyed(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "mittwald_mysql_database" {
			continue
		}

		if err := testAccAssertMySQLDatabaseIsAbsent(rs.Primary.ID); err != nil {
			return err
		}

		userID := rs.Primary.Attributes["user.id"]
		if err := testAccAssertMySQLUserIsAbsent(userID); err != nil {
			return err
		}
	}

	return nil
}

func testAccAssertMySQLDatabaseIsPresent(resourceName string, databaseOut *mittwaldv2.DeMittwaldV1DatabaseMySqlDatabase, userOut *mittwaldv2.DeMittwaldV1DatabaseMySqlUser) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found", resourceName)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		client := providertesting.TestClient().Database()

		database, err := client.PollMySQLDatabase(ctx, rs.Primary.ID)
		if err != nil {
			return err
		}

		userID := rs.Primary.Attributes["user.id"]

		user, err := client.PollMySQLUser(ctx, userID)
		if err != nil {
			return err
		}

		*databaseOut = *database
		*userOut = *user

		return nil
	}
}

func testAccAssertMySQLDatabaseDescriptionMatches(database *mittwaldv2.DeMittwaldV1DatabaseMySqlDatabase, desc string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if database.Description != desc {
			return fmt.Errorf("expected database description to be '%s', got %s", desc, database.Description)
		}

		return nil
	}
}

func testAccAssertMySQLUsernameMatchesState(resourceName string, user *mittwaldv2.DeMittwaldV1DatabaseMySqlUser) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		databaseResource, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found", resourceName)
		}

		userName := databaseResource.Primary.Attributes["user.name"]
		if userName != user.Name {
			return fmt.Errorf("expected database user name to be '%s', got %s", userName, user.Name)
		}

		return nil
	}
}

func testAccAssertMySQLDatabaseIsAbsent(databaseID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := providertesting.TestClient().Database().PollMySQLDatabase(ctx, databaseID)
	if mittwaldv2.IsNotFound(err) {
		return nil
	}

	return fmt.Errorf("expected database %s to return ErrNotFound, but got %s", databaseID, err)
}

func testAccAssertMySQLUserIsAbsent(userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := providertesting.TestClient().Database().PollMySQLUser(ctx, userID)
	if mittwaldv2.IsNotFound(err) {
		return nil
	}

	return fmt.Errorf("expected MySQL user %s to return ErrNotFound, but got %s", userID, err)
}
