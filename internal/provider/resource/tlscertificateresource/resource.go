package tlscertificateresource

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/domainclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/sslv2"
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
	resp.TypeName = req.ProviderTypeName + "_tls_certificate"
}

func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	builder := common.AttributeBuilderFor("certificate")
	resp.Schema = schema.Schema{
		MarkdownDescription: "Models a TLS certificate on the mittwald platform. " +
			"This resource is suitable for wildcard certificates (e.g. `*.foobar.example`) " +
			"that require DNS validation and cannot be created implicitly together with a virtual host.\n\n" +
			"After the certificate is created, it can be used with a `mittwald_virtualhost` resource " +
			"by adding a `depends_on` reference.",

		Attributes: map[string]schema.Attribute{
			"id": builder.Id(),
			"certificate_request_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the certificate request that was used to issue this certificate.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": builder.ProjectId(),
			"common_name": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "The common name for the certificate, e.g. `*.foobar.example`. " +
					"Changing this value forces a new certificate to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
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

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createRes, _, err := r.client.Domain().CreateCertificateRequest(ctx, data.ToCreateRequest())
	if err != nil {
		resp.Diagnostics.AddError("Error creating certificate request", err.Error())
		return
	}

	data.CertificateRequestID = types.StringValue(createRes.Id)

	// Poll until the certificate request is completed.
	// Certificate requests can take several minutes to complete, so use a long timeout.
	pollCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	_, err = apiutils.Poll(pollCtx, apiutils.PollOpts{
		InitialDelay:  5 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 1.5,
	}, func(ctx context.Context, certRequestID string) (*sslv2.CertificateRequest, error) {
		certReq, _, err := r.client.Domain().GetCertificateRequest(ctx, domainclientv2.GetCertificateRequestRequest{
			CertificateRequestID: certRequestID,
		})
		if err != nil {
			return nil, err
		}
		if !certReq.IsCompleted {
			return nil, apiutils.ErrPollShouldRetry
		}
		return certReq, nil
	}, createRes.Id)

	if err != nil {
		resp.Diagnostics.AddError("Error waiting for certificate request completion", fmt.Sprintf(
			"The certificate request %s did not complete in time: %s", createRes.Id, err.Error(),
		))
		return
	}

	resp.Diagnostics.Append(r.readCertificateByRequestID(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.read(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the certificate was not found, remove it from state.
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) read(ctx context.Context, data *ResourceModel) (res diag.Diagnostics) {
	// Prefer reading the certificate directly by ID if we have it.
	if !data.ID.IsNull() && !data.ID.IsUnknown() {
		cert := providerutil.
			Try[*sslv2.Certificate](&res, "Error reading certificate").
			IgnoreNotFound().
			DoValResp(r.client.Domain().GetCertificate(ctx, domainclientv2.GetCertificateRequest{
				CertificateID: data.ID.ValueString(),
			}))

		if res.HasError() {
			return
		}

		data.FromCertificate(cert)
		return
	}

	// Fall back to searching by certificate request ID.
	if !data.CertificateRequestID.IsNull() && !data.CertificateRequestID.IsUnknown() {
		res.Append(r.readCertificateByRequestID(ctx, data)...)
	}

	return
}

func (r *Resource) readCertificateByRequestID(ctx context.Context, data *ResourceModel) (res diag.Diagnostics) {
	certs, _, err := r.client.Domain().ListCertificates(ctx, data.ToListCertificatesRequest())
	if err != nil {
		res.AddError("Error listing certificates", err.Error())
		return
	}

	for _, cert := range *certs {
		if cert.CertificateRequestId == data.CertificateRequestID.ValueString() {
			certCopy := cert
			data.FromCertificate(&certCopy)
			return
		}
	}

	// Certificate not found; mark as removed.
	data.ID = types.StringNull()
	return
}

func (r *Resource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// No-op since all attributes have RequiresReplace plan modifiers.
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the certificate if we have its ID.
	if !data.ID.IsNull() && !data.ID.IsUnknown() {
		providerutil.
			Try[any](&resp.Diagnostics, "Error deleting certificate").
			IgnoreNotFound().
			DoResp(r.client.Domain().DeleteCertificate(ctx, domainclientv2.DeleteCertificateRequest{
				CertificateID: data.ID.ValueString(),
			}))
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the certificate request.
	if !data.CertificateRequestID.IsNull() && !data.CertificateRequestID.IsUnknown() {
		providerutil.
			Try[any](&resp.Diagnostics, "Error deleting certificate request").
			IgnoreNotFound().
			DoResp(r.client.Domain().DeleteCertificateRequest(ctx, domainclientv2.DeleteCertificateRequestRequest{
				CertificateRequestID: data.CertificateRequestID.ValueString(),
			}))
	}
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
