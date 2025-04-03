package containerstackresource_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/containerv2"
	containerstackresource "github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/containerstack"
	. "github.com/mittwald/terraform-provider-mittwald/internal/testingutils"
	. "github.com/onsi/gomega"
)

var apiModel = &containerv2.StackResponse{
	Id:        "stack-123",
	ProjectId: "project-xyz",
	Services: []containerv2.ServiceResponse{
		{
			ServiceName: "nginx",
			PendingState: containerv2.ServiceState{
				Image: "nginx:latest",
				Ports: []string{"80:8080/tcp", "443:8443/tcp", "3000/tcp"},
				Envs:  map[string]string{"ENV_VAR": "value"},
				Volumes: []string{
					"/project/web:/usr/share/nginx/html",
					"data-volume:/data",
				},
				Command:    []string{"nginx", "-g", "daemon off;"},
				Entrypoint: []string{"/bin/sh", "-c"},
			},
		},
	},
	Volumes: []containerv2.VolumeResponse{
		{
			Name: "data-volume",
		},
	},
}

func TestFromAPIModelWithEmptyStack(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	var apiModel containerv2.StackResponse
	err := json.Unmarshal([]byte("{\"description\":\"default\",\"disabled\":false,\"id\":\"10184af5-6716-4e82-81d7-4b1cd317d147\",\"prefix\":\"p-su6uw7\",\"projectId\":\"10184af5-6716-4e82-81d7-4b1cd317d147\"}"), &apiModel)
	g.Expect(err).NotTo(HaveOccurred())

	var model containerstackresource.ContainerStackModel
	diags := model.FromAPIModel(ctx, &apiModel)

	// Ensure no errors occurred during conversion
	g.Expect(diags).To(BeNil())

	// Validate top-level fields
	g.Expect(model.ID).To(Equal(types.StringValue("10184af5-6716-4e82-81d7-4b1cd317d147")))
	g.Expect(model.ProjectID).To(Equal(types.StringValue("10184af5-6716-4e82-81d7-4b1cd317d147")))

	// Validate Containers
	containers := model.Containers.Elements()
	g.Expect(containers).To(HaveLen(0))

	// Validate Volumes
	g.Expect(model.Volumes.Elements()).To(HaveLen(0))
}

func TestFromAPIModel(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	var model containerstackresource.ContainerStackModel
	diags := model.FromAPIModel(ctx, apiModel)

	// Ensure no errors occurred during conversion
	g.Expect(diags).To(BeNil())

	// Validate top-level fields
	g.Expect(model.ID).To(Equal(types.StringValue("stack-123")))
	g.Expect(model.ProjectID).To(Equal(types.StringValue("project-xyz")))

	// Validate Containers
	containers := model.Containers.Elements()
	g.Expect(containers).To(And(HaveLen(1), HaveKey("nginx")))

	nginxContainer := containers["nginx"].(types.Object)
	g.Expect(nginxContainer).To(HaveStringAttr("image", "nginx:latest"))

	// Validate ports
	portsSet := nginxContainer.Attributes()["ports"].(types.Set)
	g.Expect(portsSet.Elements()).To(And(
		HaveLen(3),
		ContainElement(And(
			HaveInt32Attr("public_port", int32(80)),
			HaveInt32Attr("container_port", int32(8080)),
			HaveStringAttr("protocol", "tcp"),
		)),
		ContainElement(And(
			HaveInt32Attr("public_port", int32(443)),
			HaveInt32Attr("container_port", int32(8443)),
			HaveStringAttr("protocol", "tcp"),
		)),
		ContainElement(And(
			HaveInt32Attr("public_port", int32(3000)),
			HaveInt32Attr("container_port", int32(3000)),
			HaveStringAttr("protocol", "tcp"),
		)),
	))

	// Validate environment variables
	envVars := nginxContainer.Attributes()["environment"].(types.Map)
	g.Expect(envVars.Elements()).To(HaveKeyWithValue("ENV_VAR", types.StringValue("value")))

	containerVolumes := nginxContainer.Attributes()["volumes"].(types.Set)
	g.Expect(containerVolumes.Elements()).To(And(
		HaveLen(2),
		ContainElement(And(
			HaveStringAttr("project_path", "/project/web"),
			HaveStringAttr("mount_path", "/usr/share/nginx/html"),
			HaveStringAttr("volume", types.StringNull()),
		)),
		ContainElement(And(
			HaveStringAttr("project_path", types.StringNull()),
			HaveStringAttr("mount_path", "/data"),
			HaveStringAttr("volume", "data-volume"),
		)),
	))

	// Validate Volumes
	g.Expect(model.Volumes.Elements()).To(And(
		HaveLen(1),
		HaveKey("data-volume"),
	))
}

func TestFromAPIModelAndReverse(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()
	var model containerstackresource.ContainerStackModel

	diags := model.FromAPIModel(ctx, apiModel)
	g.Expect(diags).To(BeNil())

	out := model.ToDeclareRequest(ctx, &diags)
	g.Expect(diags).To(BeNil())

	g.Expect(out.StackID).To(Equal("stack-123"))
	g.Expect(out.Body.Services).To(And(
		HaveLen(1),
		HaveKeyWithValue("nginx", And(
			HaveField("Image", "nginx:latest"),
			HaveField("Command", []string{"nginx", "-g", "daemon off;"}),
			HaveField("Entrypoint", []string{"/bin/sh", "-c"}),
			HaveField("Envs", HaveKeyWithValue("ENV_VAR", "value")),
			HaveField("Ports", ContainElements("80:8080/tcp", "443:8443/tcp")),
			HaveField("Volumes", ContainElements("/project/web:/usr/share/nginx/html", "data-volume:/data")),
		)),
	))
	g.Expect(out.Body.Volumes).To(And(
		HaveKey("data-volume"),
	))
}
