package tlscertificateresource

import "github.com/hashicorp/terraform-plugin-framework/types"

// ResourceModel describes the resource data model.
type ResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	CertificateRequestID types.String `tfsdk:"certificate_request_id"`
	ProjectID            types.String `tfsdk:"project_id"`
	CommonName           types.String `tfsdk:"common_name"`
	Certificate          types.String `tfsdk:"certificate"`
	PrivateKeyWO         types.String `tfsdk:"private_key_wo"`
	PrivateKeyWOVersion  types.Int64  `tfsdk:"private_key_wo_version"`
}
