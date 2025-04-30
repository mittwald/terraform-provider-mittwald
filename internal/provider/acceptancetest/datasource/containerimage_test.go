package datasource

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providertesting"
)

const testAccContainerImageDataSourceConfig = `
data "mittwald_container_image" "test" {
	image = "nginx:1.28.0"
}
`

func TestAccContainerImageDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			providertesting.TestAccPreCheck(t)
		},
		ProtoV6ProviderFactories: providertesting.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccContainerImageDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.mittwald_container_image.test", "image", "nginx:1.28.0"),
					resource.TestCheckResourceAttr("data.mittwald_container_image.test", "command.#", "3"),
					resource.TestCheckResourceAttr("data.mittwald_container_image.test", "command.0", "nginx"),
					resource.TestCheckResourceAttr("data.mittwald_container_image.test", "command.1", "-g"),
					resource.TestCheckResourceAttr("data.mittwald_container_image.test", "command.2", "daemon off;"),
					resource.TestCheckResourceAttr("data.mittwald_container_image.test", "entrypoint.#", "1"),
					resource.TestCheckResourceAttr("data.mittwald_container_image.test", "entrypoint.0", "/docker-entrypoint.sh"),
				),
			},
		},
	})
}
