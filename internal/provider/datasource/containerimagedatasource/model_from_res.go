package containerimagedatasource

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/containerv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/valueutil"
)

func (c *ContainerImageDataSourceModel) FromAPIModel(m *containerv2.ContainerImageConfig) (res diag.Diagnostics) {
	c.Command = valueutil.ConvertStringSliceToList(m.Command)
	c.Entrypoint = valueutil.ConvertStringSliceToList(m.Entrypoint)
	return
}
