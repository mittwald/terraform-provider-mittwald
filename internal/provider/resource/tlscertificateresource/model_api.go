package tlscertificateresource

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/domainclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/sslv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/ptrutil"
)

func (m *ResourceModel) ToCreateRequest() domainclientv2.CreateCertificateRequestRequest {
	return domainclientv2.CreateCertificateRequestRequest{
		Body: domainclientv2.CreateCertificateRequestRequestBody{
			AlternativeCertificateRequestCreateWithDNSRequest: &sslv2.CertificateRequestCreateWithDNSRequest{
				CommonName: m.CommonName.ValueString(),
				ProjectId:  m.ProjectID.ValueString(),
			},
		},
	}
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
		return
	}

	m.ID = types.StringValue(cert.Id)
	m.CertificateRequestID = types.StringValue(cert.CertificateRequestId)
	m.ProjectID = types.StringValue(cert.ProjectId)

	if cert.CommonName != nil {
		m.CommonName = types.StringValue(*cert.CommonName)
	}
}
