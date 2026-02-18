package readsshpublickey

import (
	"context"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

var _ function.Function = &Function{}

// Function implements the read_ssh_publickey provider function.
type Function struct{}

// New creates a new instance of the function.
func New() function.Function {
	return &Function{}
}

// Metadata returns the function metadata.
func (f *Function) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "read_ssh_publickey"
}

// Definition returns the function definition.
func (f *Function) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Reads an SSH public key from a file",
		Description: "Reads an SSH public key from a file and returns only the key type and base64-encoded key, stripping any trailing comment and whitespace. This is useful when configuring SSH users, as SSH key files typically contain a trailing comment that should be provided separately.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "path",
				Description: "The path to the SSH public key file (e.g., `~/.ssh/id_rsa.pub`)",
			},
		},
		Return: function.StringReturn{},
	}
}

// Run executes the function.
func (f *Function) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var path string

	resp.Error = function.ConcatFuncErrors(req.Arguments.Get(ctx, &path))
	if resp.Error != nil {
		return
	}

	// Expand ~ to home directory
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			resp.Error = function.NewFuncError("failed to get home directory: " + err.Error())
			return
		}
		path = home + path[1:]
	}

	// Read the file
	content, err := os.ReadFile(path)
	if err != nil {
		resp.Error = function.NewFuncError("failed to read SSH public key file: " + err.Error())
		return
	}

	// Parse the SSH public key
	// Format: <key-type> <base64-key> [comment]
	key := strings.TrimSpace(string(content))
	parts := strings.SplitN(key, " ", 3)

	if len(parts) < 2 {
		resp.Error = function.NewFuncError("invalid SSH public key format: expected '<key-type> <base64-key> [comment]'")
		return
	}

	// Return only the key type and base64 key
	result := parts[0] + " " + parts[1]

	resp.Error = function.ConcatFuncErrors(resp.Result.Set(ctx, result))
}
