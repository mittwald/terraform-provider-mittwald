package resource

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/mailclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/mailv2"
	"github.com/mittwald/api-client-go/pkg/httperr"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiutils"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providertesting"
)

func TestAccEmailOutboxResourceCreated(t *testing.T) {
	var emailOutbox mailv2.Deliverybox

	serverID := config.StringVariable(os.Getenv("MITTWALD_ACCTEST_SERVER_ID"))
	emailPassword := config.StringVariable(providertesting.TestRandomPassword(t))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			providertesting.TestAccPreCheck(t)
		},
		ProtoV6ProviderFactories: providertesting.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEmailOutboxResourceConfig("Test Outbox"),
				ConfigVariables: map[string]config.Variable{
					"server_id":      serverID,
					"email_password": emailPassword,
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mittwald_email_outbox.test", "description", "Test Outbox"),
					resource.TestCheckResourceAttrWith("mittwald_email_outbox.test", "id", providertesting.MatchUUID),
					testAccAssertEmailOutboxIsPresent("mittwald_email_outbox.test", &emailOutbox),
					testAccAssertEmailOutboxDescriptionMatches(&emailOutbox, "Test Outbox"),
				),
			},
			{
				Config: testAccEmailOutboxResourceConfig("Updated Outbox"),
				ConfigVariables: map[string]config.Variable{
					"server_id":      serverID,
					"email_password": emailPassword,
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mittwald_email_outbox.test", "description", "Updated Outbox"),
					resource.TestCheckResourceAttrWith("mittwald_email_outbox.test", "id", providertesting.MatchUUID),
					testAccAssertEmailOutboxIsPresent("mittwald_email_outbox.test", &emailOutbox),
					testAccAssertEmailOutboxDescriptionMatches(&emailOutbox, "Updated Outbox"),
				),
			},
		},
		CheckDestroy: testAccEmailOutboxResourceDestroyed,
	})
}

func testAccEmailOutboxResourceConfig(desc string) string {
	return fmt.Sprintf(`
variable "server_id" {
  type = string
}

variable "email_password" {
  type = string
  sensitive = true
}

resource "mittwald_project" "test" {
	server_id = var.server_id
	description = "terraform_emailoutbox_test"
}

resource "mittwald_email_outbox" "test" {
  project_id  = mittwald_project.test.id
  description = "%[1]s"
  password    = var.email_password
}
`, desc)
}

func testAccEmailOutboxResourceDestroyed(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "mittwald_email_outbox" {
			continue
		}

		if err := testAccAssertEmailOutboxIsAbsent(rs.Primary.ID); err != nil {
			return err
		}
	}

	return nil
}

func testAccAssertEmailOutboxIsPresent(resourceName string, outboxOut *mailv2.Deliverybox) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found", resourceName)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		client := providertesting.TestClient().Mail()

		outbox, httpResp, err := client.GetDeliveryBox(ctx, mailclientv2.GetDeliveryBoxRequest{
			DeliveryBoxID: rs.Primary.ID,
		})
		if err != nil {
			return fmt.Errorf("error getting email outbox: %s, response: %v", err, httpResp)
		}

		*outboxOut = *outbox

		return nil
	}
}

func testAccAssertEmailOutboxDescriptionMatches(outbox *mailv2.Deliverybox, desc string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if outbox.Description != desc {
			return fmt.Errorf("expected email outbox description to be '%s', got %s", desc, outbox.Description)
		}

		return nil
	}
}

func testAccAssertEmailOutboxIsAbsent(outboxID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	client := providertesting.TestClient().Mail()

	_, err := apiutils.Poll(ctx, apiutils.PollOpts{}, client.GetDeliveryBox, mailclientv2.GetDeliveryBoxRequest{
		DeliveryBoxID: outboxID,
	})
	if notFound := new(httperr.ErrNotFound); errors.As(err, &notFound) {
		return nil
	}

	return fmt.Errorf("expected email outbox %s to return ErrNotFound, but got %s", outboxID, err)
}
