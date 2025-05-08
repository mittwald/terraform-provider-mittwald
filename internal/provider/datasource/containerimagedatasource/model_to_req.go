package containerimagedatasource

import "github.com/mittwald/api-client-go/mittwaldv2/generated/clients/containerclientv2"

func (c *ContainerImageDataSourceModel) ToRequest() *containerclientv2.GetContainerImageConfigRequest {
	req := containerclientv2.GetContainerImageConfigRequest{
		ImageReference: c.Image.ValueString(),
	}

	if !c.RegistryID.IsNull() {
		req.UseCredentialsForRegistryID = c.RegistryID.ValueStringPointer()
	}

	if !c.ProjectID.IsNull() {
		req.UseCredentialsForProjectID = c.ProjectID.ValueStringPointer()
	}

	return &req
}
