package virtualhostresource

import (
	"context"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/terraform-provider-mittwald/api/mittwaldv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/valueutil"
)

func (m *ResourceModel) FromAPIModel(ctx context.Context, apiModel *mittwaldv2.DeMittwaldV1IngressIngress) (res diag.Diagnostics) {
	m.ID = valueutil.StringerOrNull(apiModel.Id)
	m.ProjectID = valueutil.StringerOrNull(apiModel.ProjectId)
	m.Hostname = types.StringValue(apiModel.Hostname)

	pathObjs := make(map[string]attr.Value)
	for _, ingressPath := range apiModel.Paths {
		attrs := map[string]attr.Value{
			"app":      types.StringNull(),
			"redirect": types.StringNull(),
		}

		if inst, err := ingressPath.Target.AsDeMittwaldV1IngressTargetInstallation(); err == nil && inst.InstallationId != uuid.Nil {
			attrs["app"] = types.StringValue(inst.InstallationId.String())
		}

		if url, err := ingressPath.Target.AsDeMittwaldV1IngressTargetUrl(); err == nil && url.Url != "" {
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

func (m *ResourceModel) ToCreateRequest(ctx context.Context, d *diag.Diagnostics) mittwaldv2.IngressCreateIngressJSONRequestBody {
	return mittwaldv2.IngressCreateIngressJSONRequestBody{
		Hostname: m.Hostname.ValueString(),
		Paths:    m.pathsAsAPIModel(ctx, d),
	}
}

func (m *ResourceModel) ToUpdateRequest(ctx context.Context, d *diag.Diagnostics, current *ResourceModel) mittwaldv2.IngressUpdateIngressPathsJSONRequestBody {
	return m.pathsAsAPIModel(ctx, d)
}

func (m *PathModel) toAPIModel(p path.Path, urlPathPrefix string, d *diag.Diagnostics) mittwaldv2.DeMittwaldV1IngressPath {
	model := mittwaldv2.DeMittwaldV1IngressPath{
		Path: urlPathPrefix,
	}

	_ = model.Target.FromDeMittwaldV1IngressTargetUseDefaultPage(mittwaldv2.DeMittwaldV1IngressTargetUseDefaultPage{
		UseDefaultPage: true,
	})

	if !m.App.IsNull() {
		err := model.Target.FromDeMittwaldV1IngressTargetInstallation(mittwaldv2.DeMittwaldV1IngressTargetInstallation{
			InstallationId: uuid.MustParse(m.App.ValueString()),
		})

		if err != nil {
			d.AddAttributeError(p.AtName("app"), "error while build app installation path target", err.Error())
		}
	}

	if !m.Redirect.IsNull() {
		err := model.Target.FromDeMittwaldV1IngressTargetUrl(mittwaldv2.DeMittwaldV1IngressTargetUrl{
			Url: m.Redirect.ValueString(),
		})

		if err != nil {
			d.AddAttributeError(p.AtName("redirect"), "error while build redirect path target", err.Error())
		}
	}

	return model
}

func (m *ResourceModel) pathsAsAPIModel(ctx context.Context, res *diag.Diagnostics) []mittwaldv2.DeMittwaldV1IngressPath {
	out := make([]mittwaldv2.DeMittwaldV1IngressPath, 0)
	intermediate := map[string]PathModel{}

	res.Append(m.Paths.ElementsAs(ctx, &intermediate, false)...)

	attrPath := path.Root("paths")

	for p, model := range intermediate {
		out = append(out, model.toAPIModel(attrPath.AtMapKey(p), p, res))
	}

	return out
}
