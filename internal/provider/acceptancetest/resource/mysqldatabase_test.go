package resource

import (
	"context"
	"errors"
	"fmt"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/databaseclientv2"
	"github.com/mittwald/api-client-go/pkg/httperr"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiutils"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/databasev2"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providertesting"
)

func TestAccMySQLDatabaseResourceCreated(t *testing.T) {
	var database databasev2.MySqlDatabase
	var user databasev2.MySqlUser

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
					testAccAssertMySQLDatabaseIsPresent(&database, &user),
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
					testAccAssertMySQLDatabaseIsPresent(&database, &user),
					testAccAssertMySQLDatabaseDescriptionMatches(&database, "Barbaz"),
				),
			},
			{
				Config: testAccMySQLDatabaseResourceConfigWithUserAccess("Barbaz", "readonly", true),
				ConfigVariables: map[string]config.Variable{
					"server_id":         serverID,
					"database_password": databasePassword,
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mittwald_mysql_database.test", "user.access_level", "readonly"),
					resource.TestCheckResourceAttr("mittwald_mysql_database.test", "user.external_access", "true"),
					testAccAssertMySQLDatabaseIsPresent(&database, &user),
					testAccAssertMySQLUserAccessLevelMatches(&user, "readonly"),
					testAccAssertMySQLUserExternalAccessMatches(&user, true),
				),
			},
		},
		CheckDestroy: testAccMySQLDatabaseResourceDestroyed,
	})
}

func TestAccMySQLDatabaseResourceCreatedWithGeneratedPassword(t *testing.T) {
	var database databasev2.MySqlDatabase
	var user databasev2.MySqlUser

	serverID := config.StringVariable(os.Getenv("MITTWALD_ACCTEST_SERVER_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			providertesting.TestAccPreCheck(t)
		},
		ProtoV6ProviderFactories: providertesting.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMySQLDatabaseResourceConfigWithGeneratedPassword("Foobar"),
				ConfigVariables: map[string]config.Variable{
					"server_id": serverID,
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mittwald_mysql_database.test", "description", "Foobar"),
					resource.TestCheckResourceAttrWith("mittwald_mysql_database.test", "id", providertesting.MatchUUID),
					testAccAssertMySQLDatabaseIsPresent(&database, &user),
					testAccAssertMySQLDatabaseDescriptionMatches(&database, "Foobar"),
					testAccAssertMySQLUsernameMatchesState("mittwald_mysql_database.test", &user),
				),
			},
		},
		CheckDestroy: testAccMySQLDatabaseResourceDestroyed,
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

func testAccMySQLDatabaseResourceConfigWithUserAccess(desc, accessLevel string, externalAccess bool) string {
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
    access_level    = "%[2]s"
    password        = var.database_password
    external_access = %[3]t
  }
}
`, desc, accessLevel, externalAccess)
}

func testAccMySQLDatabaseResourceConfigWithGeneratedPassword(desc string) string {
	return fmt.Sprintf(`
variable "server_id" {
  type = string
}

ephemeral "mittwald_mysql_password" "password" {
  length = 24
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
    access_level        = "full"
    password_wo         = ephemeral.mittwald_mysql_password.password.password
    password_wo_version = 1
    external_access     = false
  }
}
`, desc)
}

func testAccMySQLDatabaseResourceDestroyed(s *terraform.State) error {
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

func testAccAssertMySQLDatabaseIsPresent(databaseOut *databasev2.MySqlDatabase, userOut *databasev2.MySqlUser) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["mittwald_mysql_database.test"]
		if !ok {
			return fmt.Errorf("resource mittwald_mysql_database.test not found")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		client := providertesting.TestClient().Database()

		database, err := apiutils.PollRequest(ctx, apiutils.PollOpts{}, client.GetMysqlDatabase, databaseclientv2.GetMysqlDatabaseRequest{MysqlDatabaseID: rs.Primary.ID})
		if err != nil {
			return err
		}

		userID := rs.Primary.Attributes["user.id"]

		user, err := apiutils.PollRequest(ctx, apiutils.PollOpts{}, client.GetMysqlUser, databaseclientv2.GetMysqlUserRequest{MysqlUserID: userID})
		if err != nil {
			return err
		}

		*databaseOut = *database
		*userOut = *user

		return nil
	}
}

func testAccAssertMySQLDatabaseDescriptionMatches(database *databasev2.MySqlDatabase, desc string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if database.Description != desc {
			return fmt.Errorf("expected database description to be '%s', got %s", desc, database.Description)
		}

		return nil
	}
}

func testAccAssertMySQLUserAccessLevelMatches(user *databasev2.MySqlUser, accessLevel string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if string(user.AccessLevel) != accessLevel {
			return fmt.Errorf("expected database user access level to be '%s', got %s", accessLevel, user.AccessLevel)
		}

		return nil
	}
}

func testAccAssertMySQLUserExternalAccessMatches(user *databasev2.MySqlUser, externalAccess bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if user.ExternalAccess != externalAccess {
			return fmt.Errorf("expected database user external access to be '%t', got %t", externalAccess, user.ExternalAccess)
		}

		return nil
	}
}

func testAccAssertMySQLUsernameMatchesState(resourceName string, user *databasev2.MySqlUser) resource.TestCheckFunc {
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

	client := providertesting.TestClient().Database()

	_, err := apiutils.PollRequest(ctx, apiutils.PollOpts{}, client.GetMysqlDatabase, databaseclientv2.GetMysqlDatabaseRequest{MysqlDatabaseID: databaseID})
	if notFound := new(httperr.ErrNotFound); errors.As(err, &notFound) {
		return nil
	}

	return fmt.Errorf("expected database %s to return ErrNotFound, but got %s", databaseID, err)
}

func testAccAssertMySQLUserIsAbsent(userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	client := providertesting.TestClient().Database()

	_, err := apiutils.PollRequest(ctx, apiutils.PollOpts{}, client.GetMysqlUser, databaseclientv2.GetMysqlUserRequest{MysqlUserID: userID})
	if notFound := new(httperr.ErrNotFound); errors.As(err, &notFound) {
		return nil
	}

	return fmt.Errorf("expected MySQL user %s to return ErrNotFound, but got %s", userID, err)
}
