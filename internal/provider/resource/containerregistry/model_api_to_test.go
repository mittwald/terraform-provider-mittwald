package containerregistryresource_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	containerregistryresource "github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/containerregistry"
	. "github.com/onsi/gomega"
)

var credentialsAttrTypes = map[string]attr.Type{
	"username":            types.StringType,
	"password_wo":         types.StringType,
	"password_wo_version": types.Int64Type,
}

func TestToUpdateRequestWithoutCredentials(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	model := containerregistryresource.ContainerRegistryModel{
		ID:          types.StringValue("registry-123"),
		ProjectID:   types.StringValue("project-456"),
		Description: types.StringValue("Test Registry"),
		URI:         types.StringValue("registry.example.com"),
		Credentials: types.ObjectNull(credentialsAttrTypes),
	}

	var diags diag.Diagnostics
	password := types.StringNull()

	req := model.ToUpdateRequest(ctx, &diags, password)

	// Ensure no errors occurred during conversion
	g.Expect(diags.HasError()).To(BeFalse(), "expected no diagnostics errors")

	// Validate request fields
	g.Expect(req.RegistryID).To(Equal("registry-123"))
	g.Expect(req.Body.Description).To(HaveValue(Equal("Test Registry")))
	g.Expect(req.Body.Uri).To(HaveValue(Equal("registry.example.com")))

	// Validate that credentials are set to nil (removing credentials)
	g.Expect(req.Body.Credentials).NotTo(BeNil(), "credentials should be present in request")
	g.Expect(req.Body.Credentials.Value).To(BeNil(), "credentials value should be nil")
}

func TestToUpdateRequestWithCredentials(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	credentialsObj, diags := types.ObjectValue(
		credentialsAttrTypes,
		map[string]attr.Value{
			"username":            types.StringValue("testuser"),
			"password_wo":         types.StringValue("oldpassword"),
			"password_wo_version": types.Int64Value(1),
		},
	)
	g.Expect(diags.HasError()).To(BeFalse(), "failed to create credentials object")

	model := containerregistryresource.ContainerRegistryModel{
		ID:          types.StringValue("registry-123"),
		ProjectID:   types.StringValue("project-456"),
		Description: types.StringValue("Test Registry"),
		URI:         types.StringValue("registry.example.com"),
		Credentials: credentialsObj,
	}

	var reqDiags diag.Diagnostics
	password := types.StringValue("newpassword")

	req := model.ToUpdateRequest(ctx, &reqDiags, password)

	// Ensure no errors occurred during conversion
	g.Expect(reqDiags.HasError()).To(BeFalse(), "expected no diagnostics errors")

	// Validate request fields
	g.Expect(req.RegistryID).To(Equal("registry-123"))
	g.Expect(req.Body.Description).To(HaveValue(Equal("Test Registry")))
	g.Expect(req.Body.Uri).To(HaveValue(Equal("registry.example.com")))

	// Validate that credentials are properly set
	g.Expect(req.Body.Credentials).NotTo(BeNil(), "credentials should be present in request")
	g.Expect(req.Body.Credentials.Value).NotTo(BeNil(), "credentials value should not be nil")
	g.Expect(req.Body.Credentials.Value.Username).To(Equal("testuser"))
	g.Expect(req.Body.Credentials.Value.Password).To(Equal("newpassword"))
}

func TestToUpdateRequestWithCredentialsButNullPassword(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	credentialsObj, diagsObj := types.ObjectValue(
		credentialsAttrTypes,
		map[string]attr.Value{
			"username":            types.StringValue("testuser"),
			"password_wo":         types.StringValue("oldpassword"),
			"password_wo_version": types.Int64Value(1),
		},
	)
	g.Expect(diagsObj.HasError()).To(BeFalse(), "failed to create credentials object")

	model := containerregistryresource.ContainerRegistryModel{
		ID:          types.StringValue("registry-123"),
		ProjectID:   types.StringValue("project-456"),
		Description: types.StringValue("Test Registry"),
		URI:         types.StringValue("registry.example.com"),
		Credentials: credentialsObj,
	}

	var reqDiags diag.Diagnostics
	password := types.StringNull()

	req := model.ToUpdateRequest(ctx, &reqDiags, password)

	// Ensure no errors occurred during conversion
	g.Expect(reqDiags.HasError()).To(BeFalse(), "expected no diagnostics errors")

	// Validate that credentials are set to nil (removing credentials when password is null)
	g.Expect(req.Body.Credentials).NotTo(BeNil(), "credentials should be present in request")
	g.Expect(req.Body.Credentials.Value).To(BeNil(), "credentials value should be nil when password is null")
}
