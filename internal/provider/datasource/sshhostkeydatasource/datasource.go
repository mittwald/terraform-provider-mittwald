package sshhostkeydatasource

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/terraform-provider-mittwald/internal/sshutil"
	"golang.org/x/crypto/ssh"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &DataSource{}

// New creates a new SSH host key data source.
func New() datasource.DataSource {
	return &DataSource{}
}

// DataSource defines the data source implementation.
type DataSource struct{}

// DataSourceModel describes the data source data model.
type DataSourceModel struct {
	Hostname    types.String `tfsdk:"hostname"`
	Port        types.Int64  `tfsdk:"port"`
	KeyType     types.String `tfsdk:"key_type"`
	Key         types.String `tfsdk:"key"`
	Fingerprint types.String `tfsdk:"fingerprint"`
}

// Metadata returns the data source type name.
func (d *DataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_host_key"
}

// Schema returns the data source schema.
func (d *DataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches the SSH host key from a mittwald server. This is useful for configuring known hosts in CI/CD systems like Bitbucket Pipelines without manual `ssh-keyscan` steps.\n\n" +
			"-> **Note:** In most cases, you can use `mittwald_app.<name>.ssh_host_key` and `mittwald_app.<name>.ssh_host_key_type` directly instead of this data source.",

		Attributes: map[string]schema.Attribute{
			"hostname": schema.StringAttribute{
				MarkdownDescription: "The SSH hostname to fetch the host key from (e.g., `ssh.xxx.mittwald.net`). Use `mittwald_app.<name>.ssh_host` to get this automatically.",
				Required:            true,
			},
			"port": schema.Int64Attribute{
				MarkdownDescription: "The SSH port. Defaults to `22`.",
				Optional:            true,
			},
			"key_type": schema.StringAttribute{
				MarkdownDescription: "The type of the host key (e.g., `ssh-ed25519`, `ssh-rsa`, `ecdsa-sha2-nistp256`).",
				Computed:            true,
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "The host's public key in base64 format (without the key type prefix).",
				Computed:            true,
			},
			"fingerprint": schema.StringAttribute{
				MarkdownDescription: "The SHA256 fingerprint of the host key.",
				Computed:            true,
			},
		},
	}
}

// Read fetches the SSH host key.
func (d *DataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hostname := data.Hostname.ValueString()
	port := int64(22)
	if !data.Port.IsNull() {
		port = data.Port.ValueInt64()
	}

	address := fmt.Sprintf("%s:%d", hostname, port)

	// Fetch the host key using the shared utility
	hostKeyInfo, err := sshutil.FetchHostKey(ctx, address)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to fetch SSH host key",
			fmt.Sprintf("Could not fetch SSH host key from %s: %s", address, err),
		)
		return
	}

	// Calculate fingerprint separately since we need the raw key for that
	fingerprint, err := calculateFingerprint(ctx, address)
	if err != nil {
		// Not a critical error, just set empty fingerprint
		data.Fingerprint = types.StringNull()
	} else {
		data.Fingerprint = types.StringValue(fingerprint)
	}

	data.KeyType = types.StringValue(hostKeyInfo.KeyType)
	data.Key = types.StringValue(hostKeyInfo.Key)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// calculateFingerprint connects to the SSH server and calculates the fingerprint.
func calculateFingerprint(ctx context.Context, address string) (string, error) {
	var hostKey ssh.PublicKey

	hostKeyCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		hostKey = key
		return fmt.Errorf("host key captured")
	}

	config := &ssh.ClientConfig{
		User:            "probe",
		HostKeyCallback: hostKeyCallback,
		Auth:            []ssh.AuthMethod{},
		Timeout:         10 * time.Second,
	}

	dialer := net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	sshConn, _, _, _ := ssh.NewClientConn(conn, address, config)
	if sshConn != nil {
		sshConn.Close()
	}

	if hostKey != nil {
		return ssh.FingerprintSHA256(hostKey), nil
	}

	return "", fmt.Errorf("no host key received")
}
