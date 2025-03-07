package redisdatabaseresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/databaseclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/databasev2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiutils"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/common"
	"time"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &Resource{}
var _ resource.ResourceWithImportState = &Resource{}

func New() resource.Resource {
	return &Resource{}
}

// Resource defines the resource implementation.
type Resource struct {
	client mittwaldv2.Client
}

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_redis_database"
}

func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	builder := common.AttributeBuilderFor("redis_database")
	resp.Schema = schema.Schema{
		MarkdownDescription: "Models a Redis database on the mittwald plattform",

		Attributes: map[string]schema.Attribute{
			"id": builder.Id(),
			"version": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Version of the database, e.g. `7.0`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Name of the database, e.g. `redis_XXXXX`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id":  builder.ProjectId(),
			"description": builder.Description(),
			"hostname": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Hostname of the database; this is the hostname that you should use within the platform to connect to the database.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"configuration": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"additional_flags": schema.ListAttribute{
						Description: "Additional command-line flags that should be passed to the Redis container",
						ElementType: types.StringType,
						Optional:    true,
					},
					"max_memory_mb": schema.Int64Attribute{
						MarkdownDescription: "The database's maximum memory in MiB",
						Optional:            true,
					},
					"max_memory_policy": schema.StringAttribute{
						MarkdownDescription: "The database's key eviction policy. See the Redis documentation on key evictions for more information.",
						Optional:            true,
					},
					"persistent": schema.BoolAttribute{
						MarkdownDescription: "Enable persistent storage for this database",
						Optional:            true,
					},
				},
			},
		},
	}
}

func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.ClientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ResourceModel

	client := r.client.Database()

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	databaseResponse := providerutil.
		Try[*databaseclientv2.CreateRedisDatabaseResponse](&resp.Diagnostics, "error while creating database").
		DoValResp(client.CreateRedisDatabase(
			ctx,
			data.ToCreateRequest(ctx, &resp.Diagnostics),
		))

	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(databaseResponse.Id)

	resp.Diagnostics.Append(r.read(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	readCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	resp.Diagnostics.Append(r.read(readCtx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) read(ctx context.Context, data *ResourceModel) (res diag.Diagnostics) {
	client := r.client.Database()

	database := providerutil.
		Try[*databasev2.RedisDatabase](&res, "error while reading database").
		IgnoreNotFound().
		DoVal(apiutils.Poll(ctx, apiutils.PollOpts{}, client.GetRedisDatabase, databaseclientv2.GetRedisDatabaseRequest{RedisDatabaseID: data.ID.ValueString()}))

	if res.HasError() {
		return
	}

	res.Append(data.FromAPIModel(ctx, database)...)

	return
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var dataPlan, dataState ResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &dataPlan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &dataState)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !dataPlan.Description.Equal(dataState.Description) {
		updateReq := dataPlan.ToUpdateDescriptionRequest()
		if _, err := r.client.Database().UpdateRedisDatabaseDescription(ctx, updateReq); err != nil {
			resp.Diagnostics.AddError("Error while updating database description", err.Error())
		}
	}

	if !dataPlan.Configuration.Equal(dataState.Configuration) {
		updateReq := dataPlan.ToUpdateConfigurationRequest(ctx, &resp.Diagnostics)
		if _, err := r.client.Database().UpdateRedisDatabaseConfiguration(ctx, updateReq); err != nil {
			resp.Diagnostics.AddError("Error while updating database configuration", err.Error())
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &dataPlan)...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	providerutil.
		Try[any](&resp.Diagnostics, "error while deleting database").
		IgnoreNotFound().
		DoResp(r.client.Database().DeleteRedisDatabase(ctx, data.ToDeleteRequest()))
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
