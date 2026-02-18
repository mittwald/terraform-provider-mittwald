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
	diags := model.FromAPIModel(ctx, &apiModel, &model, false)

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
	diags := model.FromAPIModel(ctx, apiModel, &model, false)

	// Ensure no errors occurred during conversion
	g.Expect(diags).To(BeNil())

	// Validate top-level fields
	g.Expect(model.ID).To(Equal(types.StringValue("stack-123")))
	g.Expect(model.ProjectID).To(Equal(types.StringValue("project-xyz")))

	// Validate Containers
	containers := model.Containers.Elements()
	g.Expect(containers).To(And(HaveLen(1), HaveKey("nginx")))

	nginxContainer, ok := containers["nginx"].(types.Object)
	g.Expect(ok).To(BeTrue())
	g.Expect(nginxContainer).To(HaveStringAttr("image", "nginx:latest"))

	// Validate ports
	portsSet, ok := nginxContainer.Attributes()["ports"].(types.Set)
	g.Expect(ok).To(BeTrue())
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
	envVars, ok := nginxContainer.Attributes()["environment"].(types.Map)
	g.Expect(ok).To(BeTrue())
	g.Expect(envVars.Elements()).To(HaveKeyWithValue("ENV_VAR", types.StringValue("value")))

	containerVolumes, ok := nginxContainer.Attributes()["volumes"].(types.Set)
	g.Expect(ok).To(BeTrue())
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

	diags := model.FromAPIModel(ctx, apiModel, &model, false)
	g.Expect(diags).To(BeNil())

	g.Expect(model.Volumes.IsNull()).To(BeFalse())
	g.Expect(model.Volumes.Elements()).To(HaveLen(1))

	out := model.ToDeclareRequest(ctx, &diags)
	g.Expect(diags).To(BeNil())

	g.Expect(out.StackID).To(Equal("stack-123"))
	g.Expect(out.Body.Services).To(And(
		HaveLen(1),
		HaveKeyWithValue("nginx", And(
			HaveField("Image", "nginx:latest"),
			HaveField("Command", []string{"nginx", "-g", "daemon off;"}),
			HaveField("Entrypoint", []string{"/bin/sh", "-c"}),
			HaveField("Environment", HaveKeyWithValue("ENV_VAR", "value")),
			HaveField("Ports", ContainElements("80:8080/tcp", "443:8443/tcp")),
			HaveField("Volumes", ContainElements("/project/web:/usr/share/nginx/html", "data-volume:/data")),
		)),
	))
	g.Expect(out.Body.Volumes).To(And(
		HaveKey("data-volume"),
	))
}

func TestFromAPIModelWithLimits(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	cpusValue := "0.5"
	memoryValue := "512M"

	apiModelWithLimits := &containerv2.StackResponse{
		Id:        "stack-123",
		ProjectId: "project-xyz",
		Services: []containerv2.ServiceResponse{
			{
				ServiceName: "nginx",
				Id:          "service-abc",
				Description: "Test container",
				PendingState: containerv2.ServiceState{
					Image:      "nginx:latest",
					Command:    []string{"nginx", "-g", "daemon off;"},
					Entrypoint: []string{"/bin/sh", "-c"},
				},
				Deploy: &containerv2.Deploy{
					Resources: &containerv2.Resources{
						Limits: &containerv2.ResourceSpec{
							Cpus:   &cpusValue,
							Memory: &memoryValue,
						},
					},
				},
			},
		},
	}

	var model containerstackresource.ContainerStackModel
	diags := model.FromAPIModel(ctx, apiModelWithLimits, &model, false)
	g.Expect(diags).To(BeNil())

	containers := model.Containers.Elements()
	g.Expect(containers).To(HaveLen(1))

	nginxContainer, ok := containers["nginx"].(types.Object)
	g.Expect(ok).To(BeTrue())

	// Validate limits
	limitsObj, ok := nginxContainer.Attributes()["limits"].(types.Object)
	g.Expect(ok).To(BeTrue())
	g.Expect(limitsObj.IsNull()).To(BeFalse())

	limitsAttrs := limitsObj.Attributes()
	g.Expect(limitsAttrs["cpus"]).To(Equal(types.Float64Value(0.5)))
	g.Expect(limitsAttrs["memory"]).To(Equal(types.StringValue("512M")))
}

func TestLimitsRoundTrip(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	cpusValue := "2"
	memoryValue := "1G"

	apiModelWithLimits := &containerv2.StackResponse{
		Id:        "stack-456",
		ProjectId: "project-abc",
		Services: []containerv2.ServiceResponse{
			{
				ServiceName: "app",
				Id:          "service-xyz",
				Description: "Test app",
				PendingState: containerv2.ServiceState{
					Image:      "app:latest",
					Command:    []string{"app"},
					Entrypoint: []string{"/bin/sh"},
				},
				Deploy: &containerv2.Deploy{
					Resources: &containerv2.Resources{
						Limits: &containerv2.ResourceSpec{
							Cpus:   &cpusValue,
							Memory: &memoryValue,
						},
					},
				},
			},
		},
	}

	// Convert API model to Terraform model
	var model containerstackresource.ContainerStackModel
	diags := model.FromAPIModel(ctx, apiModelWithLimits, &model, false)
	g.Expect(diags).To(BeNil())

	// Convert back to API request
	declareRequest := model.ToDeclareRequest(ctx, &diags)
	g.Expect(diags).To(BeNil())

	// Validate the limits in the DeclareRequest
	g.Expect(declareRequest.Body.Services).To(HaveLen(1))
	appService := declareRequest.Body.Services["app"]
	
	g.Expect(appService.Deploy).NotTo(BeNil())
	g.Expect(appService.Deploy.Resources).NotTo(BeNil())
	g.Expect(appService.Deploy.Resources.Limits).NotTo(BeNil())
	g.Expect(appService.Deploy.Resources.Limits.Cpus).NotTo(BeNil())
	g.Expect(*appService.Deploy.Resources.Limits.Cpus).To(Equal("2"))
	g.Expect(appService.Deploy.Resources.Limits.Memory).NotTo(BeNil())
	g.Expect(*appService.Deploy.Resources.Limits.Memory).To(Equal("1G"))
}

func TestFromAPIModelWithoutLimits(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	apiModelWithoutLimits := &containerv2.StackResponse{
		Id:        "stack-123",
		ProjectId: "project-xyz",
		Services: []containerv2.ServiceResponse{
			{
				ServiceName: "nginx",
				Id:          "service-abc",
				Description: "Test container",
				PendingState: containerv2.ServiceState{
					Image:      "nginx:latest",
					Command:    []string{"nginx"},
					Entrypoint: []string{"/bin/sh"},
				},
				Deploy: nil,
			},
		},
	}

	var model containerstackresource.ContainerStackModel
	diags := model.FromAPIModel(ctx, apiModelWithoutLimits, &model, false)
	g.Expect(diags).To(BeNil())

	containers := model.Containers.Elements()
	g.Expect(containers).To(HaveLen(1))

	nginxContainer, ok := containers["nginx"].(types.Object)
	g.Expect(ok).To(BeTrue())

	// Validate that limits is null when not provided
	limitsObj, ok := nginxContainer.Attributes()["limits"].(types.Object)
	g.Expect(ok).To(BeTrue())
	g.Expect(limitsObj.IsNull()).To(BeTrue())
}

func TestParsePortString(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedPort  containerstackresource.ContainerPortModel
		expectError   bool
		errorContains string
	}{
		{
			name:  "valid port with public and container ports",
			input: "80:8080/tcp",
			expectedPort: containerstackresource.ContainerPortModel{
				PublicPort:    types.Int32Value(80),
				ContainerPort: types.Int32Value(8080),
				Protocol:      types.StringValue("tcp"),
			},
			expectError: false,
		},
		{
			name:  "valid port with only container port",
			input: "3000/tcp",
			expectedPort: containerstackresource.ContainerPortModel{
				PublicPort:    types.Int32Value(3000),
				ContainerPort: types.Int32Value(3000),
				Protocol:      types.StringValue("tcp"),
			},
			expectError: false,
		},
		{
			name:  "valid port with UDP protocol",
			input: "53:5353/udp",
			expectedPort: containerstackresource.ContainerPortModel{
				PublicPort:    types.Int32Value(53),
				ContainerPort: types.Int32Value(5353),
				Protocol:      types.StringValue("udp"),
			},
			expectError: false,
		},
		{
			name:          "invalid port format - missing protocol",
			input:         "8080",
			expectError:   true,
			errorContains: "invalid port format",
		},
		{
			name:          "invalid port format - too many parts",
			input:         "80:8080:9090/tcp",
			expectError:   true,
			errorContains: "invalid port mapping",
		},
		{
			name:          "invalid port format - non-numeric container port",
			input:         "abc/tcp",
			expectError:   true,
			errorContains: "invalid port value",
		},
		{
			name:          "invalid port format - non-numeric public port",
			input:         "abc:8080/tcp",
			expectError:   true,
			errorContains: "invalid public port",
		},
		{
			name:          "invalid port format - zero port",
			input:         "0/tcp",
			expectError:   true,
			errorContains: "invalid port value",
		},
		{
			name:          "invalid port format - negative port",
			input:         "-1/tcp",
			expectError:   true,
			errorContains: "invalid port value",
		},
		{
			name:          "invalid port format - port too large",
			input:         "99999999999/tcp",
			expectError:   true,
			errorContains: "invalid port value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			result, err := containerstackresource.ParsePortString(tt.input)

			if tt.expectError {
				g.Expect(err).To(HaveOccurred())
				if tt.errorContains != "" {
					g.Expect(err.Error()).To(ContainSubstring(tt.errorContains))
				}
			} else {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(result.PublicPort).To(Equal(tt.expectedPort.PublicPort))
				g.Expect(result.ContainerPort).To(Equal(tt.expectedPort.ContainerPort))
				g.Expect(result.Protocol).To(Equal(tt.expectedPort.Protocol))
			}
		})
	}
}
