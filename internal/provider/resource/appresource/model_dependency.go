package appresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/appv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiext"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
)

type DependencyModel struct {
	Version      types.String `tfsdk:"version"`
	UpdatePolicy types.String `tfsdk:"update_policy"`
}

var dependencyType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"version":       types.StringType,
		"update_policy": types.StringType,
	},
}

func InstalledSystemSoftwareToDependencyModelMap(
	ctx context.Context,
	res diag.Diagnostics,
	appClient apiext.AppClient,
	systemSoftwares []appv2.InstalledSystemSoftware,
) types.Map {
	dependencyMapValues := make(map[string]attr.Value)
	for _, dep := range systemSoftwares {
		systemSoftware, version, err := appClient.GetSystemsoftwareAndVersion(
			ctx,
			dep.SystemSoftwareId,
			dep.SystemSoftwareVersion.Desired,
		)

		if err != nil {
			providerutil.ErrorToDiag(err)(&res, "API Error")
			return types.Map{}
		}

		mod := types.Object{}

		tfsdk.ValueFrom(ctx, DependencyModel{
			Version:      types.StringValue(version.InternalVersion),
			UpdatePolicy: types.StringValue(string(dep.UpdatePolicy)),
		}, dependencyType, &mod)

		dependencyMapValues[systemSoftware.Name] = mod
	}

	dependencyMap, d := basetypes.NewMapValue(dependencyType, dependencyMapValues)
	res.Append(d...)

	return dependencyMap
}
