package datasource

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providertesting"
	"strings"
	"testing"
)

const testAccSystemSoftwareDataSourceConfig = `
data "mittwald_systemsoftware" "test" {
	name = "php"
	selector = "~8.2"
}
`

func TestAccSystemSoftwareDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			providertesting.TestAccPreCheck(t)
		},
		ProtoV6ProviderFactories: providertesting.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSystemSoftwareDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.mittwald_systemsoftware.test", "name", "php"),
					resource.TestCheckResourceAttrWith("data.mittwald_systemsoftware.test", "version_id", providertesting.MatchUUID),
					resource.TestCheckResourceAttrWith("data.mittwald_systemsoftware.test", "version", func(value string) error {
						if !strings.HasPrefix(value, "8.2.") {
							return fmt.Errorf("expected version to start with 8.2, got %s", value)
						}
						return nil
					}),
				),
			},
		},
	})
}
