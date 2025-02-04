package virtualhostresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/domainclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/ingressv2"
)

func (m *ResourceModel) FromAPIModel(ctx context.Context, apiModel *ingressv2.Ingress) (res diag.Diagnostics) {
	m.ID = types.StringValue(apiModel.Id)
	m.ProjectID = types.StringValue(apiModel.ProjectId)
	m.Hostname = types.StringValue(apiModel.Hostname)

	pathObjs := make(map[string]attr.Value)
	for _, ingressPath := range apiModel.Paths {
		attrs := map[string]attr.Value{
			"app":      types.StringNull(),
			"redirect": types.StringNull(),
		}

		if inst := ingressPath.Target.AlternativeTargetInstallation; inst != nil && inst.InstallationId != "" {
			attrs["app"] = types.StringValue(inst.InstallationId)
		}

		if url := ingressPath.Target.AlternativeTargetUrl; url != nil && url.Url != "" {
			attrs["redirect"] = types.StringValue(url.Url)
		}

		obj, d := types.ObjectValue(pathType.AttrTypes, attrs)
		res.Append(d...)

		pathObjs[ingressPath.Path] = obj
	}

	p, d := types.MapValue(pathType, pathObjs)
	res.Append(d...)

	m.Paths = p

	return
}

func (m *ResourceModel) ToCreateRequest(ctx context.Context, d *diag.Diagnostics) domainclientv2.CreateIngressRequest {
	return domainclientv2.CreateIngressRequest{
		Body: domainclientv2.CreateIngressRequestBody{
			Hostname: m.Hostname.ValueString(),
			Paths:    m.pathsAsAPIModel(ctx, d),
		},
	}
}

func (m *ResourceModel) ToUpdateRequest(ctx context.Context, d *diag.Diagnostics, current *ResourceModel) domainclientv2.UpdateIngressPathsRequest {
	return domainclientv2.UpdateIngressPathsRequest{
		IngressID: m.ID.ValueString(),
		Body:      m.pathsAsAPIModel(ctx, d),
	}
}

func (m *ResourceModel) ToDeleteRequest() domainclientv2.DeleteIngressRequest {
	return domainclientv2.DeleteIngressRequest{
		IngressID: m.ID.ValueString(),
	}
}

func (m *PathModel) toAPIModel(p path.Path, urlPathPrefix string, d *diag.Diagnostics) ingressv2.Path {
	model := ingressv2.Path{
		Path: urlPathPrefix,
	}

	if !m.App.IsNull() {
		model.Target.AlternativeTargetInstallation = &ingressv2.TargetInstallation{
			InstallationId: m.App.ValueString(),
		}
	} else if !m.Redirect.IsNull() {
		model.Target.AlternativeTargetUrl = &ingressv2.TargetUrl{
			Url: m.Redirect.ValueString(),
		}
	} else {
		model.Target.AlternativeTargetUseDefaultPage = &ingressv2.TargetUseDefaultPage{
			UseDefaultPage: true,
		}
	}

	return model
}

func (m *ResourceModel) pathsAsAPIModel(ctx context.Context, res *diag.Diagnostics) []ingressv2.Path {
	out := make([]ingressv2.Path, 0)
	intermediate := map[string]PathModel{}

	res.Append(m.Paths.ElementsAs(ctx, &intermediate, false)...)

	attrPath := path.Root("paths")

	for p, model := range intermediate {
		out = append(out, model.toAPIModel(attrPath.AtMapKey(p), p, res))
	}

	return out
}
