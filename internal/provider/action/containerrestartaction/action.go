package containerrestartaction

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/containerclientv2"
)

var _ action.Action = &Action{}

type Action struct {
	client mittwaldv2.Client
}

func New() action.Action {
	return &Action{}
}

type RecreateModel struct {
	StackID     types.String `tfsdk:"stack_id"`
	ContainerID types.String `tfsdk:"container_id"`
}

func (a *Action) Schema(_ context.Context, _ action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Restarts a container of a given stack.",
		Attributes: map[string]schema.Attribute{
			"stack_id": schema.StringAttribute{
				Description: "ID of the stack in which to recreate a given container",
				Required:    true,
			},
			"container_id": schema.StringAttribute{
				Description: "ID of the container to recreate",
				Required:    true,
			},
		},
	}
}

func (a *Action) Metadata(_ context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_restart"
}

func (a *Action) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	params := &RecreateModel{}

	resp.Diagnostics.Append(req.Config.Get(ctx, &params)...)
	if resp.Diagnostics.HasError() {
		return
	}

	restartRequest := containerclientv2.RestartServiceRequest{
		StackID:   params.StackID.ValueString(),
		ServiceID: params.ContainerID.ValueString(),
	}

	_, err := a.client.Container().RestartService(ctx, restartRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Container Restart Error",
			"An error was encountered while restarting the container: "+err.Error(),
		)
		return
	}
}
