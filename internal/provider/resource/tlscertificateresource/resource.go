package tlscertificateresource

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
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
var _ resource.ResourceWithConfigValidators = &Resource{}

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
		MarkdownDescription: "Models a TLS certificate on the mittwald platform.\n\n" +
			"Certificates can be created in two ways:\n\n" +
			"1. **DNS validation** (for wildcard certificates like `*.foobar.example`): " +
			"provide only `common_name` and `project_id`. The certificate is requested " +
			"using DNS validation and provisioned automatically.\n\n" +
			"2. **Certificate import**: provide `certificate`, `private_key_wo`, " +
			"`private_key_wo_version`, and `common_name`. An existing PEM-encoded " +
			"certificate and private key are imported.\n\n" +
			"After the certificate is provisioned, it is used automatically by any " +
			"`mittwald_virtualhost` resource in the same project. Use `depends_on` to ensure " +
			"the certificate is ready before creating the virtual host.",

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
				Optional: true,
				Computed: true,
				MarkdownDescription: "The common name (domain) for the certificate, e.g. `*.foobar.example`. " +
					"Required when using DNS validation. For certificate import, this is " +
					"derived from the certificate's CN and populated automatically. " +
					"Changing this value forces a new certificate to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"certificate": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "PEM-encoded certificate to import. When set (together with " +
					"`private_key_wo` and `private_key_wo_version`), the certificate is imported " +
					"instead of being provisioned via DNS validation. " +
					"This value can be updated in place to renew the certificate. " +
					"Note: switching between DNS validation and certificate import modes forces " +
					"a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					// Force replacement when switching between DNS validation (null) and import (non-null).
					stringplanmodifier.RequiresReplaceIf(
						func(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
							resp.RequiresReplace = req.StateValue.IsNull() != req.PlanValue.IsNull()
						},
						"Switching between DNS validation and certificate import modes requires replacement.",
						"Switching between DNS validation and certificate import modes requires replacement.",
					),
				},
			},
			"private_key_wo": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				WriteOnly: true,
				MarkdownDescription: "PEM-encoded private key for the certificate. " +
					"This is a write-only attribute. To trigger a private key update, " +
					"change the value of `private_key_wo_version`.",
			},
			"private_key_wo_version": schema.Int64Attribute{
				Optional: true,
				MarkdownDescription: "Version counter for the private key. Increment this value " +
					"to trigger an in-place update of the certificate's private key.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
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
	var privateKey types.String

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("private_key_wo"), &privateKey)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createRes, _, err := r.client.Domain().CreateCertificateRequest(ctx, data.ToCreateRequest(privateKey.ValueString()))
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
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		resp.Diagnostics.AddError(
			"Certificate not found after request completion",
			fmt.Sprintf("No certificate could be found for certificate request %s after it completed.", data.CertificateRequestID.ValueString()),
		)
		return
	}

	// For DNS-validated certificates, also poll until the certificate itself is in the "ready" state.
	resp.Diagnostics.Append(r.waitForCertificateReady(pollCtx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// waitForCertificateReady polls the certificate until its DNS status is "ready".
// For imported certificates (which have no DnsCertSpec), this is a no-op.
func (r *Resource) waitForCertificateReady(ctx context.Context, data *ResourceModel) (res diag.Diagnostics) {
	cert, err := apiutils.Poll(ctx, apiutils.PollOpts{
		InitialDelay:  5 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 1.5,
	}, func(ctx context.Context, certID string) (*sslv2.Certificate, error) {
		c, _, err := r.client.Domain().GetCertificate(ctx, domainclientv2.GetCertificateRequest{
			CertificateID: certID,
		})
		if err != nil {
			return nil, err
		}

		if c.DnsCertSpec == nil || c.DnsCertSpec.Status == nil {
			// No DNS cert spec (e.g. imported certificate) — nothing to wait for.
			return c, nil
		}

		switch c.DnsCertSpec.Status.Status {
		case sslv2.ProjectCertificateStatusReady:
			return c, nil
		case sslv2.ProjectCertificateStatusError, sslv2.ProjectCertificateStatusCnameError:
			msg := string(c.DnsCertSpec.Status.Status)
			if c.DnsCertSpec.Status.Message != nil {
				msg = *c.DnsCertSpec.Status.Message
			}
			return nil, fmt.Errorf("certificate entered error state %q: %s", c.DnsCertSpec.Status.Status, msg)
		default:
			return nil, apiutils.ErrPollShouldRetry
		}
	}, data.ID.ValueString())

	if err != nil {
		res.AddError(
			"Error waiting for certificate to become ready",
			fmt.Sprintf("Certificate %s did not reach ready state in time: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	data.FromCertificate(cert)
	return
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

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData, stateData ResourceModel
	var privateKey types.String

	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("private_key_wo"), &privateKey)...)
	if resp.Diagnostics.HasError() {
		return
	}

	certChanged := !planData.Certificate.Equal(stateData.Certificate)
	versionChanged := !planData.PrivateKeyWOVersion.Equal(stateData.PrivateKeyWOVersion)

	if certChanged || versionChanged {
		pk := ""
		if !privateKey.IsNull() {
			pk = privateKey.ValueString()
		}

		providerutil.
			Try[any](&resp.Diagnostics, "Error replacing certificate").
			DoResp(r.client.Domain().ReplaceCertificate(ctx, planData.ToReplaceCertificateRequest(pk)))

		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(r.read(ctx, &planData)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
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

func (r *Resource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		certImportFieldsValidator{},
	}
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
