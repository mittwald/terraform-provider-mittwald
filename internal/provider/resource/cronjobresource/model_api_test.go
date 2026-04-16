package cronjobresource

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/cronjobv2"
	. "github.com/onsi/gomega"
)

func TestToCreateRequestUsesContainerTarget(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	containerObj, containerDiags := types.ObjectValue(
		map[string]attr.Type{
			"stack_id":   types.StringType,
			"service_id": types.StringType,
		},
		map[string]attr.Value{
			"stack_id":   types.StringValue("10184af5-6716-4e82-81d7-4b1cd317d147"),
			"service_id": types.StringValue("nginx"),
		},
	)
	g.Expect(containerDiags.HasError()).To(BeFalse())

	destinationObj, destinationDiags := types.ObjectValue(
		map[string]attr.Type{
			"url":               types.StringType,
			"command":           types.ObjectType{AttrTypes: map[string]attr.Type{"interpreter": types.StringType, "path": types.StringType, "parameters": types.ListType{ElemType: types.StringType}}},
			"container_command": types.ListType{ElemType: types.StringType},
		},
		map[string]attr.Value{
			"url":               types.StringNull(),
			"command":           types.ObjectNull(map[string]attr.Type{"interpreter": types.StringType, "path": types.StringType, "parameters": types.ListType{ElemType: types.StringType}}),
			"container_command": mustListValue(t, []string{"echo", "Hello World"}),
		},
	)
	g.Expect(destinationDiags.HasError()).To(BeFalse())

	model := ResourceModel{
		ProjectID:   types.StringValue("10184af5-6716-4e82-81d7-4b1cd317d147"),
		Description: types.StringValue("Demo Cronjob"),
		Interval:    types.StringValue("*/5 * * * *"),
		AppID:       types.StringNull(),
		Container:   containerObj,
		Destination: destinationObj,
		Email:       types.StringNull(),
		Timezone:    types.StringNull(),
	}

	var diags diag.Diagnostics
	req := model.ToCreateRequest(ctx, &diags)
	g.Expect(diags.HasError()).To(BeFalse())
	g.Expect(req.Body.Target).NotTo(BeNil())
	g.Expect(req.Body.Target.AlternativeServiceTarget).NotTo(BeNil())
	g.Expect(req.Body.Target.AlternativeServiceTarget.StackId).To(Equal("10184af5-6716-4e82-81d7-4b1cd317d147"))
	g.Expect(req.Body.Target.AlternativeServiceTarget.ServiceIdentifier).To(Equal("nginx"))
	g.Expect(req.Body.Target.AlternativeServiceTarget.Command).To(Equal("echo 'Hello World'"))
	g.Expect(req.Body.Target.AlternativeAppInstallationTarget).To(BeNil())
	g.Expect(req.Body.AppId).To(BeNil())
	g.Expect(req.Body.Destination).To(BeNil())
}

func TestFromAPIModelReadsContainerTarget(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	apiModel := &cronjobv2.Cronjob{
		Id:          "8d446005-e368-43e2-9681-a8f5b922fca2",
		Active:      true,
		AppId:       "",
		Description: "Demo Cronjob",
		Interval:    "*/5 * * * *",
		Target: &cronjobv2.CronjobTarget{
			AlternativeServiceTargetResponse: &cronjobv2.ServiceTargetResponse{
				StackId:        "10184af5-6716-4e82-81d7-4b1cd317d147",
				ServiceShortId: "nginx",
				Command:        "echo 'Hello World'",
			},
		},
	}

	var model ResourceModel
	diags := model.FromAPIModel(ctx, apiModel)
	g.Expect(diags.HasError()).To(BeFalse())
	g.Expect(model.AppID.IsNull()).To(BeTrue())

	var containerModel ResourceContainerModel
	diags.Append(model.Container.As(ctx, &containerModel, basetypes.ObjectAsOptions{})...)
	g.Expect(diags.HasError()).To(BeFalse())
	g.Expect(containerModel.StackID).To(Equal(types.StringValue("10184af5-6716-4e82-81d7-4b1cd317d147")))
	g.Expect(containerModel.ServiceID).To(Equal(types.StringValue("nginx")))

	dest := model.GetDestination(ctx, &diags)
	g.Expect(diags.HasError()).To(BeFalse())
	var commandParts []string
	diags.Append(dest.ContainerCommand.ElementsAs(ctx, &commandParts, false)...)
	g.Expect(diags.HasError()).To(BeFalse())
	g.Expect(commandParts).To(Equal([]string{"echo", "Hello World"}))
}

func TestSplitShellCommand(t *testing.T) {
	g := NewWithT(t)
	parts, err := splitShellCommand("echo 'Hello World' \"foo bar\" plain\\ value")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(parts).To(Equal([]string{"echo", "Hello World", "foo bar", "plain value"}))
}

func mustListValue(t *testing.T, values []string) types.List {
	t.Helper()
	v, diags := types.ListValueFrom(context.Background(), types.StringType, values)
	if diags.HasError() {
		t.Fatalf("failed to build list value: %v", diags)
	}
	return v
}
