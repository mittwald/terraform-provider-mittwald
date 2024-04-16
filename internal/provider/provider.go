package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/mittwald/terraform-provider-mittwald/api/mittwaldv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/datasource/projectdatasource"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/datasource/systemsoftwaredatasource"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/datasource/userdatasource"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/appresource"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/cronjobresource"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/mysqldatabaseresource"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/projectresource"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure MittwaldProvider satisfies various provider interfaces.
var _ provider.Provider = &MittwaldProvider{}

// MittwaldProvider defines the provider implementation.
type MittwaldProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// MittwaldProviderModel describes the provider data model.
type MittwaldProviderModel struct {
	Endpoint           types.String `tfsdk:"endpoint"`
	APIKey             types.String `tfsdk:"api_key"`
	DebugRequestBodies types.Bool   `tfsdk:"debug_request_bodies"`
}

func (p *MittwaldProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "mittwald"
	resp.Version = p.version
}

func (p *MittwaldProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "API endpoint for the Mittwald API. Default to `https://api.mittwald.de/v2` if omitted.",
				Optional:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "API key for the Mittwald API; if omitted, the `MITTWALD_API_TOKEN` environment variable will be used.",
				Optional:            true,
				Sensitive:           true,
			},
			"debug_request_bodies": schema.BoolAttribute{
				MarkdownDescription: "Whether to log request bodies when debugging is enabled. CAUTION: This will log sensitive data such as passwords in plain text!",
				Optional:            true,
			},
		},
	}
}

func (p *MittwaldProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data MittwaldProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.APIKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(path.Root("api_key"), "unknown mittwald API key", "cannot create the mittwald API client because an unknown value was supplied for the API key")
	}

	apiKey := os.Getenv("MITTWALD_API_TOKEN")
	if !data.APIKey.IsNull() {
		apiKey = data.APIKey.ValueString()
	}

	opts := make([]mittwaldv2.ClientBuilderOption, 0)

	if apiKey != "" {
		opts = append(opts, mittwaldv2.WithAPIToken(apiKey))
	} else {
		resp.Diagnostics.AddAttributeError(path.Root("api_key"), "unknown mittwald API key", "cannot create the mittwald API client because no API key was supplied")
	}

	if !data.Endpoint.IsNull() {
		opts = append(opts, mittwaldv2.WithEndpoint(data.Endpoint.ValueString()))
	}

	opts = append(opts, mittwaldv2.WithDebugging(data.DebugRequestBodies.ValueBool()))

	client := mittwaldv2.New(opts...)

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *MittwaldProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		projectresource.New,
		appresource.New,
		mysqldatabaseresource.New,
		cronjobresource.New,
	}
}

func (p *MittwaldProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		projectdatasource.NewByShortIdDataSource,
		systemsoftwaredatasource.New,
		userdatasource.New,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &MittwaldProvider{
			version: version,
		}
	}
}
