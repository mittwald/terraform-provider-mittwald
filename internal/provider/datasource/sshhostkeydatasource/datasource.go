package sshhostkeydatasource

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
		MarkdownDescription: "Fetches the SSH host key from a mittwald server. This is useful for configuring known hosts in CI/CD systems like Bitbucket Pipelines without manual `ssh-keyscan` steps.",

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

	// Fetch the host key by attempting an SSH connection
	hostKey, err := fetchHostKey(ctx, address)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to fetch SSH host key",
			fmt.Sprintf("Could not fetch SSH host key from %s: %s", address, err),
		)
		return
	}

	// Parse the key type and key data
	keyType := hostKey.Type()
	keyData := ssh.MarshalAuthorizedKey(hostKey)
	keyStr := strings.TrimSpace(string(keyData))

	// Extract just the base64 key part (remove type prefix and any comment)
	parts := strings.Fields(keyStr)
	keyBase64 := ""
	if len(parts) >= 2 {
		keyBase64 = parts[1]
	}

	// Calculate fingerprint
	fingerprint := ssh.FingerprintSHA256(hostKey)

	data.KeyType = types.StringValue(keyType)
	data.Key = types.StringValue(keyBase64)
	data.Fingerprint = types.StringValue(fingerprint)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// fetchHostKey connects to the SSH server and retrieves its host key.
func fetchHostKey(ctx context.Context, address string) (ssh.PublicKey, error) {
	var hostKey ssh.PublicKey

	// Create a custom host key callback that captures the key
	hostKeyCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		hostKey = key
		// Return an error to abort the connection after getting the key
		// We don't actually want to authenticate
		return fmt.Errorf("host key captured")
	}

	config := &ssh.ClientConfig{
		User:            "probe",
		HostKeyCallback: hostKeyCallback,
		Auth:            []ssh.AuthMethod{}, // No auth methods - we just want the host key
		Timeout:         10 * time.Second,
	}

	// Create a connection with context timeout
	dialer := net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	// Perform SSH handshake - this will fail after getting the host key
	// but that's expected and we capture the key in the callback
	sshConn, _, _, err := ssh.NewClientConn(conn, address, config)
	if sshConn != nil {
		sshConn.Close()
	}

	// If we got the host key, the error is expected ("host key captured")
	if hostKey != nil {
		return hostKey, nil
	}

	// If we didn't get a host key, return the actual error
	if err != nil {
		return nil, fmt.Errorf("SSH handshake failed: %w", err)
	}

	return nil, fmt.Errorf("no host key received")
}
