package tlscertificateresource

import (
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/domainclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/sslv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/ptrutil"
	"github.com/mittwald/terraform-provider-mittwald/internal/valueutil"
)

// ToCreateRequest returns the appropriate CreateCertificateRequestRequest based on
// whether a pre-existing certificate is being imported or a new DNS-validated
// certificate is being requested.
func (m *ResourceModel) ToCreateRequest(privateKey string) domainclientv2.CreateCertificateRequestRequest {
	if !m.Certificate.IsNull() && !m.Certificate.IsUnknown() {
		return domainclientv2.CreateCertificateRequestRequest{
			Body: domainclientv2.CreateCertificateRequestRequestBody{
				AlternativeCertificateRequestCreateRequest: &sslv2.CertificateRequestCreateRequest{
					Certificate: m.Certificate.ValueString(),
					PrivateKey:  privateKey,
					ProjectId:   m.ProjectID.ValueString(),
				},
			},
		}
	}

	return domainclientv2.CreateCertificateRequestRequest{
		Body: domainclientv2.CreateCertificateRequestRequestBody{
			AlternativeCertificateRequestCreateWithDNSRequest: &sslv2.CertificateRequestCreateWithDNSRequest{
				CommonName: m.CommonName.ValueString(),
				ProjectId:  m.ProjectID.ValueString(),
			},
		},
	}
}

func (m *ResourceModel) ToReplaceCertificateRequest(privateKey string) domainclientv2.ReplaceCertificateRequest {
	req := domainclientv2.ReplaceCertificateRequest{
		CertificateID: m.ID.ValueString(),
		Body: domainclientv2.ReplaceCertificateRequestBody{
			Certificate: m.Certificate.ValueString(),
		},
	}
	if privateKey != "" {
		req.Body.PrivateKey = ptrutil.To(privateKey)
	}
	return req
}

func (m *ResourceModel) ToListCertificatesRequest() domainclientv2.ListCertificatesRequest {
	return domainclientv2.ListCertificatesRequest{
		ProjectID: ptrutil.To(m.ProjectID.ValueString()),
	}
}

func (m *ResourceModel) FromCertificate(cert *sslv2.Certificate) {
	if cert == nil {
		m.ID = types.StringNull()
		m.CertificateRequestID = types.StringNull()
		m.ProjectID = types.StringNull()
		m.CommonName = types.StringNull()
		m.ValidFrom = types.StringNull()
		m.ValidTo = types.StringNull()
		m.CaBundle = types.StringNull()
		m.Issuer = types.StringNull()
		m.DnsNames = types.ListNull(types.StringType)
		return
	}

	m.ID = types.StringValue(cert.Id)
	m.CertificateRequestID = types.StringValue(cert.CertificateRequestId)
	m.ProjectID = types.StringValue(cert.ProjectId)

	if cert.CommonName != nil {
		m.CommonName = types.StringValue(*cert.CommonName)
	} else {
		m.CommonName = types.StringNull()
	}

	m.ValidFrom = timePtrOrRFC3339Null(cert.ValidFrom)
	m.ValidTo = timePtrOrRFC3339Null(cert.ValidTo)
	m.CaBundle = valueutil.StringPtrOrNull(cert.CaBundle)
	m.Issuer = valueutil.StringPtrOrNull(cert.Issuer)

	if cert.DnsNames == nil {
		m.DnsNames = types.ListNull(types.StringType)
	} else {
		m.DnsNames = valueutil.ConvertStringSliceToList(cert.DnsNames)
	}
}

func timePtrOrRFC3339Null(t *time.Time) types.String {
	if t == nil {
		return types.StringNull()
	}
	return types.StringValue(t.Format(time.RFC3339))
}
