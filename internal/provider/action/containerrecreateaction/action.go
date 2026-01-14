package containerrecreateaction

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/containerclientv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
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
	PullImage   types.Bool   `tfsdk:"pull_image"`
}

func (a *Action) Schema(_ context.Context, _ action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Recreates a container of a given stack.",
		Attributes: map[string]schema.Attribute{
			"stack_id": schema.StringAttribute{
				Description: "ID of the stack in which to recreate a given container",
				Required:    true,
			},
			"container_id": schema.StringAttribute{
				Description: "ID of the container to recreate",
				Required:    true,
			},
			"pull_image": schema.BoolAttribute{
				Description: "Whether to pull the latest image before recreating the container",
				Optional:    true,
			},
		},
	}
}

func (a *Action) Configure(_ context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.client = providerutil.ClientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (a *Action) Metadata(_ context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_recreate"
}

func (a *Action) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	params := &RecreateModel{}

	resp.Diagnostics.Append(req.Config.Get(ctx, &params)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var err error

	if params.PullImage.ValueBool() {
		pullRequest := containerclientv2.PullImageForServiceRequest{
			StackID:   params.StackID.ValueString(),
			ServiceID: params.ContainerID.ValueString(),
		}

		_, err = a.client.Container().PullImageForService(ctx, pullRequest)
	} else {
		recreateRequest := containerclientv2.RecreateServiceRequest{
			StackID:   params.StackID.ValueString(),
			ServiceID: params.ContainerID.ValueString(),
		}

		_, err = a.client.Container().RecreateService(ctx, recreateRequest)
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Container Recreate Error",
			"An error was encountered while recreating the container: "+err.Error(),
		)
		return
	}
}
