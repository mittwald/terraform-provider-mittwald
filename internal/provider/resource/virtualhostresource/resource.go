package virtualhostresource

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/domainclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/ingressv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiext"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiutils"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/common"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &Resource{}
var _ resource.ResourceWithImportState = &Resource{}

func New() resource.Resource {
	return &Resource{}
}

type Resource struct {
	client mittwaldv2.Client
}

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtualhost"
}

func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	builder := common.AttributeBuilderFor("virtualhost")
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource models a virtualhost.",

		Attributes: map[string]schema.Attribute{
			"id":         builder.Id(),
			"project_id": builder.ProjectId(),
			"hostname": schema.StringAttribute{
				Description: "The desired hostname for the virtualhost.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"default": schema.BoolAttribute{
				MarkdownDescription: "Describes if this vhost is the project's default virtual host. The default virtual host will never be deleted. If you attempt to delete this resource via terraform, it will simply revert to an unmanaged state.",
				Computed:            true,
			},
			"acme": schema.BoolAttribute{
				MarkdownDescription: "Controls if a TLS certificate should be automatically issued for this virtual host via ACME (Let's Encrypt). Set this to `false` to disable automatic certificate issuance, for example if you want to assign a certificate manually at a later point. If omitted, the virtual host will use ACME when it is created, and this attribute will reflect the current state of the virtual host afterwards. Note that this attribute will also read as `false` when a certificate has been assigned to the virtual host manually.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"paths": schema.MapNestedAttribute{
				Description: "The desired paths for the virtualhost.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"app": schema.StringAttribute{
							MarkdownDescription: "The ID of an app installation that this path should point to.",
							Optional:            true,
						},
						"redirect": schema.StringAttribute{
							MarkdownDescription: "The URL to redirect to.",
							Optional:            true,
						},
						"container": schema.SingleNestedAttribute{
							Optional: true,
							Attributes: map[string]schema.Attribute{
								"container_id": schema.StringAttribute{
									MarkdownDescription: "The ID of a container (!= the ID of a container *stack*) that this path should point to. Must be a full UUID (not a short ID like c-XXXXXX).",
									Required:            true,
									Validators: []validator.String{
										&common.UUIDValidator{},
									},
								},
								"port": schema.StringAttribute{
									MarkdownDescription: "A port number/protocol combination of the referenced container that traffic should be redirected to (example: `8080/tcp`)",
									Required:            true,
								},
							},
						},
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

	client := apiext.NewDomainClient(r.client)

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// ignoring error on purpose
	existing, _ := client.GetIngressByName(ctx, data.ProjectID.ValueString(), data.Hostname.ValueString())
	if existing != nil && existing.IsDefault {
		var current ResourceModel

		resp.Diagnostics.Append(current.FromAPIModel(ctx, existing)...)
		if resp.Diagnostics.HasError() {
			return
		}

		data.ID = types.StringValue(existing.Id)

		body := data.ToUpdateRequest(ctx, &resp.Diagnostics, &current)
		if resp.Diagnostics.HasError() {
			return
		}

		providerutil.
			Try[any](&resp.Diagnostics, "API error while updating virtual host").
			DoResp(r.client.Domain().UpdateIngressPaths(ctx, body))
	} else {
		ingress := providerutil.
			Try[*domainclientv2.CreateIngressResponse](&resp.Diagnostics, "API error while creating virtual host").
			DoValResp(client.CreateIngress(ctx, data.ToCreateRequest(ctx, &resp.Diagnostics)))
		if resp.Diagnostics.HasError() {
			return
		}

		data.ID = types.StringValue(ingress.Id)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	desiredAcme := data.Acme

	readCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp.Diagnostics.Append(r.read(readCtx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The create endpoint does not accept TLS settings, so if the configured
	// ACME setting deviates from what the API created, it needs to be updated
	// in a separate step.
	if !desiredAcme.IsNull() && !desiredAcme.IsUnknown() && !desiredAcme.Equal(data.Acme) {
		data.Acme = desiredAcme
		resp.Diagnostics.Append(r.updateTLS(readCtx, &data)...)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// updateTLS updates the TLS settings of the ingress to match the ACME setting
// of the given model, waits until the change is reflected by the API, and
// re-reads the ingress into the model afterwards.
func (r *Resource) updateTLS(ctx context.Context, data *ResourceModel) (res diag.Diagnostics) {
	providerutil.
		Try[*domainclientv2.UpdateIngressTLSResponse](&res, "API error while updating virtual host TLS settings").
		DoValResp(r.client.Domain().UpdateIngressTLS(ctx, data.ToUpdateTLSRequest()))
	if res.HasError() {
		return
	}

	desiredAcme := data.Acme.ValueBool()
	waitForTLSUpdate := func(ctx context.Context, req domainclientv2.GetIngressRequest) (*ingressv2.Ingress, error) {
		ingress, _, err := r.client.Domain().GetIngress(ctx, req)
		if err != nil {
			return nil, err
		}

		if acmeEnabled(ingress) != desiredAcme {
			return nil, apiutils.ErrPollShouldRetry
		}

		return ingress, nil
	}

	ingress := providerutil.
		Try[*ingressv2.Ingress](&res, "API error while waiting for virtual host TLS settings to be updated").
		DoVal(apiutils.Poll(ctx, apiutils.PollOpts{}, waitForTLSUpdate, domainclientv2.GetIngressRequest{IngressID: data.ID.ValueString()}))
	if res.HasError() {
		return
	}

	res.Append(data.FromAPIModel(ctx, ingress)...)

	return
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data := ResourceModel{}

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
	ingress := providerutil.
		Try[*ingressv2.Ingress](&res, "API error while fetching ingress").
		DoVal(apiutils.PollRequest(ctx, apiutils.PollOpts{}, r.client.Domain().GetIngress, domainclientv2.GetIngressRequest{IngressID: data.ID.ValueString()}))

	if res.HasError() {
		return
	}

	res.Append(data.FromAPIModel(ctx, ingress)...)

	return
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData, stateData ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)

	body := planData.ToUpdateRequest(ctx, &resp.Diagnostics, &stateData)
	if resp.Diagnostics.HasError() {
		return
	}

	providerutil.
		Try[any](&resp.Diagnostics, "API error while updating virtual host").
		DoResp(r.client.Domain().UpdateIngressPaths(ctx, body))
	if resp.Diagnostics.HasError() {
		return
	}

	if !planData.Acme.IsNull() && !planData.Acme.IsUnknown() && !planData.Acme.Equal(stateData.Acme) {
		stateData.Acme = planData.Acme

		tlsCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		resp.Diagnostics.Append(r.updateTLS(tlsCtx, &stateData)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(r.read(ctx, &stateData)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if data.Default.ValueBool() {
		tflog.Debug(ctx, "virtualhost is default host; not deleting")
		return
	}

	providerutil.
		Try[any](&resp.Diagnostics, "API error while deleting virtual host").
		DoResp(r.client.Domain().DeleteIngress(ctx, data.ToDeleteRequest()))
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
