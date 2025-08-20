package resource_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providertesting"
)

func TestAccRemoteFileResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providertesting.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccRemoteFileResourceConfig("test-file.txt", "Hello, World!"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mittwald_remote_file.test", "path", "/tmp/test-file.txt"),
					resource.TestCheckResourceAttr("mittwald_remote_file.test", "contents", "Hello, World!"),
				),
			},
			// Update and Read testing
			{
				Config: testAccRemoteFileResourceConfig("test-file.txt", "Updated content"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mittwald_remote_file.test", "path", "/tmp/test-file.txt"),
					resource.TestCheckResourceAttr("mittwald_remote_file.test", "contents", "Updated content"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccRemoteFileResourceConfig(filename string, content string) string {
	return fmt.Sprintf(`
resource "mittwald_remote_file" "test" {
  container_id = "c-abc123"  # This would be a real container ID in a real test
  path         = "/tmp/%s"
  contents     = "%s"
}
`, filename, content)
}
