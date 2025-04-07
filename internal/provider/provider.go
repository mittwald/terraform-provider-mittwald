package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/mittwald/api-client-go/mittwaldv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/logadapter"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/datasource/appdatasource"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/datasource/projectdatasource"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/datasource/systemsoftwaredatasource"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/datasource/userdatasource"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/appresource"
	containerregistryresource "github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/containerregistry"
	containerstackresource "github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/containerstack"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/cronjobresource"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/mysqldatabaseresource"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/projectresource"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/redisdatabaseresource"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/virtualhostresource"
	"log/slog"
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
		MarkdownDescription: "The mittwald provider is used to manage mittwald mStudio resources.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "API endpoint for the mittwald API. Default to `https://api.mittwald.de/v2` if omitted. During regular usage, you probably won't need this. However, it can be useful for testing against a different API endpoint.",
				Optional:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "API key for the mittwald API; if omitted, the `MITTWALD_API_TOKEN` environment variable will be used.",
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

	opts := make([]mittwaldv2.ClientOption, 0)

	if apiKey != "" {
		opts = append(opts, mittwaldv2.WithAccessToken(apiKey))
	} else {
		resp.Diagnostics.AddAttributeError(path.Root("api_key"), "unknown mittwald API key", "cannot create the mittwald API client because no API key was supplied")
	}

	if !data.Endpoint.IsNull() {
		opts = append(opts, mittwaldv2.WithBaseURL(data.Endpoint.ValueString()))
	}

	logger := slog.New(&logadapter.TFLHandler{})
	opts = append(opts, mittwaldv2.WithRequestLogging(logger, data.DebugRequestBodies.ValueBool(), data.DebugRequestBodies.ValueBool()))

	client, err := mittwaldv2.New(ctx, opts...)
	if err != nil {
		resp.Diagnostics.AddError("error initializing API client", err.Error())
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *MittwaldProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		projectresource.New,
		appresource.New,
		mysqldatabaseresource.New,
		redisdatabaseresource.New,
		cronjobresource.New,
		virtualhostresource.New,
		containerstackresource.New,
		containerregistryresource.New,
	}
}

func (p *MittwaldProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		projectdatasource.NewByShortIdDataSource,
		systemsoftwaredatasource.New,
		appdatasource.New,
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
