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

func TestAccRedisDatabaseResourceCreated(t *testing.T) {
	var database databasev2.RedisDatabase

	serverID := config.StringVariable(os.Getenv("MITTWALD_ACCTEST_SERVER_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			providertesting.TestAccPreCheck(t)
		},
		ProtoV6ProviderFactories: providertesting.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRedisDatabaseResourceConfig("Foobar"),
				ConfigVariables: map[string]config.Variable{
					"server_id": serverID,
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mittwald_redis_database.test", "description", "Foobar"),
					resource.TestCheckResourceAttrWith("mittwald_redis_database.test", "id", providertesting.MatchUUID),
					testAccAssertRedisDatabaseIsPresent("mittwald_redis_database.test", &database),
					testAccAssertRedisDatabaseDescriptionMatches(&database, "Foobar"),
				),
			},
			{
				Config: testAccRedisDatabaseResourceConfig("Barbaz"),
				ConfigVariables: map[string]config.Variable{
					"server_id": serverID,
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mittwald_redis_database.test", "description", "Barbaz"),
					resource.TestCheckResourceAttrWith("mittwald_redis_database.test", "id", providertesting.MatchUUID),
					testAccAssertRedisDatabaseIsPresent("mittwald_redis_database.test", &database),
					testAccAssertRedisDatabaseDescriptionMatches(&database, "Barbaz"),
				),
			},
		},
		CheckDestroy: testAccRedisDatabaseResourceDestroyed,
	})
}

func testAccRedisDatabaseResourceConfig(desc string) string {
	return fmt.Sprintf(`
variable "server_id" {
  type = string
}

resource "mittwald_project" "test" {
  server_id = var.server_id
  description = "terraform_Redisdatabase_test"
}

resource "mittwald_redis_database" "test" {
  project_id = mittwald_project.test.id
  description = "%[1]s"
  version = "7.2"
  configuration = {
    max_memory_mb     = 64
    max_memory_policy = "volatile-lru"
    persistent        = true
  }
}
`, desc)
}

func testAccRedisDatabaseResourceDestroyed(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "mittwald_redis_database" {
			continue
		}

		if err := testAccAssertRedisDatabaseIsAbsent(rs.Primary.ID); err != nil {
			return err
		}
	}

	return nil
}

func testAccAssertRedisDatabaseIsPresent(resourceName string, databaseOut *databasev2.RedisDatabase) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found", resourceName)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		client := providertesting.TestClient().Database()

		database, err := apiutils.PollRequest(ctx, apiutils.PollOpts{}, client.GetRedisDatabase, databaseclientv2.GetRedisDatabaseRequest{RedisDatabaseID: rs.Primary.ID})
		if err != nil {
			return err
		}

		*databaseOut = *database

		return nil
	}
}

func testAccAssertRedisDatabaseDescriptionMatches(database *databasev2.RedisDatabase, desc string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if database.Description != desc {
			return fmt.Errorf("expected database description to be '%s', got %s", desc, database.Description)
		}

		return nil
	}
}

func testAccAssertRedisDatabaseIsAbsent(databaseID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	client := providertesting.TestClient().Database()

	_, err := apiutils.PollRequest(ctx, apiutils.PollOpts{}, client.GetRedisDatabase, databaseclientv2.GetRedisDatabaseRequest{RedisDatabaseID: databaseID})
	if notFound := new(httperr.ErrNotFound); errors.As(err, &notFound) {
		return nil
	}

	return fmt.Errorf("expected database %s to return ErrNotFound, but got %s", databaseID, err)
}
