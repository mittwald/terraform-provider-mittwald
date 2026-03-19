package datasource

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providertesting"
)

const testAccArticleDataSourceConfigWithFilterOnly = `
data "mittwald_article" "test" {
	filter = {
		id = "PS23-BASE"
	}
}
`

const testAccArticleDataSourceConfigWithFilterAndSelect = `
data "mittwald_article" "test" {
	filter = {
		id = "PS23-*"
		attributes = {
			machineType = "shared.xlarge"
		}
	}
	select = {
		by_price = "lowest"
	}
}
`

func TestAccArticleDataSource_withFilterOnly(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			providertesting.TestAccPreCheck(t)
		},
		ProtoV6ProviderFactories: providertesting.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccArticleDataSourceConfigWithFilterOnly,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.mittwald_article.test", "id", "PS23-BASE"),
					resource.TestCheckResourceAttrSet("data.mittwald_article.test", "orderable"),
					resource.TestCheckResourceAttrSet("data.mittwald_article.test", "price"),
				),
			},
		},
	})
}

func TestAccArticleDataSource_withFilterAndSelect(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			providertesting.TestAccPreCheck(t)
		},
		ProtoV6ProviderFactories: providertesting.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccArticleDataSourceConfigWithFilterAndSelect,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.mittwald_article.test", "id"),
					resource.TestCheckResourceAttrSet("data.mittwald_article.test", "orderable"),
					resource.TestCheckResourceAttrSet("data.mittwald_article.test", "price"),
				),
			},
		},
	})
}
