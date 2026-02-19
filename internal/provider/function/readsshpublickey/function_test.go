package readsshpublickey_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	readsshpublickey "github.com/mittwald/terraform-provider-mittwald/internal/provider/function/readsshpublickey"
	. "github.com/onsi/gomega"
)

func TestReadSSHPublicKey(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	tests := []struct {
		name          string
		fileContent   string
		expectedKey   string
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid RSA key with comment",
			fileContent: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC user@example.com",
			expectedKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC",
		},
		{
			name:        "valid ED25519 key with comment",
			fileContent: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGQwGnQqhKJJnKqJf user@host",
			expectedKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGQwGnQqhKJJnKqJf",
		},
		{
			name:        "valid ECDSA key with comment",
			fileContent: "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTY my-key",
			expectedKey: "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTY",
		},
		{
			name:        "key without comment",
			fileContent: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC",
			expectedKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC",
		},
		{
			name:        "key with trailing newline",
			fileContent: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC user@example.com\n",
			expectedKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC",
		},
		{
			name:        "key with multiple trailing newlines",
			fileContent: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC user@example.com\n\n\n",
			expectedKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC",
		},
		{
			name:        "key with leading and trailing whitespace",
			fileContent: "  \t ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC user@example.com  \t\n",
			expectedKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC",
		},
		{
			name:        "key with tabs between fields",
			fileContent: "ssh-rsa\t\tAAAAB3NzaC1yc2EAAAADAQABAAABgQC\t\tuser@example.com",
			expectedKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC",
		},
		{
			name:        "key with multiple spaces between fields",
			fileContent: "ssh-rsa     AAAAB3NzaC1yc2EAAAADAQABAAABgQC     user@example.com",
			expectedKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC",
		},
		{
			name:        "key with mixed tabs and spaces",
			fileContent: "ssh-rsa  \t  AAAAB3NzaC1yc2EAAAADAQABAAABgQC  \t  user@example.com",
			expectedKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC",
		},
		{
			name:        "key with multi-word comment",
			fileContent: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC user@example.com extra comment words",
			expectedKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC",
		},
		{
			name:          "invalid format - only one field",
			fileContent:   "ssh-rsa",
			expectError:   true,
			errorContains: "invalid SSH public key format",
		},
		{
			name:          "invalid format - empty file",
			fileContent:   "",
			expectError:   true,
			errorContains: "invalid SSH public key format",
		},
		{
			name:          "invalid format - only whitespace",
			fileContent:   "   \n\t   ",
			expectError:   true,
			errorContains: "invalid SSH public key format",
		},
		{
			name:          "invalid format - only comment",
			fileContent:   "user@example.com",
			expectError:   true,
			errorContains: "invalid SSH public key format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file with test content
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test_key.pub")
			err := os.WriteFile(tmpFile, []byte(tt.fileContent), 0600)
			g.Expect(err).To(BeNil(), "failed to create test file")

			// Execute the function
			fn := readsshpublickey.New()
			req := function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{basetypes.NewStringValue(tmpFile)}),
			}
			resp := &function.RunResponse{
				Result: function.NewResultData(basetypes.NewStringNull()),
			}

			fn.Run(ctx, req, resp)

			if tt.expectError {
				g.Expect(resp.Error).NotTo(BeNil(), "expected an error but got none")
				if tt.errorContains != "" {
					g.Expect(resp.Error.Error()).To(ContainSubstring(tt.errorContains))
				}
			} else {
				g.Expect(resp.Error).To(BeNil(), "did not expect an error: %v", resp.Error)

				result := resp.Result.Value()
				g.Expect(result).NotTo(BeNil(), "expected a result value")
				stringResult, ok := result.(basetypes.StringValue)
				g.Expect(ok).To(BeTrue(), "expected result to be a StringValue")
				g.Expect(stringResult.ValueString()).To(Equal(tt.expectedKey))
			}
		})
	}
}

func TestReadSSHPublicKeyTildeExpansion(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	// Get the user's home directory
	homeDir, err := os.UserHomeDir()
	g.Expect(err).To(BeNil(), "failed to get home directory")

	// Create a temporary SSH key file in a subdirectory of home
	testSubDir := ".test-ssh-keys-" + t.Name()
	testDir := filepath.Join(homeDir, testSubDir)
	err = os.MkdirAll(testDir, 0700)
	g.Expect(err).To(BeNil(), "failed to create test directory")
	defer os.RemoveAll(testDir)

	keyFile := filepath.Join(testDir, "id_rsa.pub")
	keyContent := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC test@example.com"
	err = os.WriteFile(keyFile, []byte(keyContent), 0600)
	g.Expect(err).To(BeNil(), "failed to create test key file")

	// Test with tilde path
	tildePath := "~/" + testSubDir + "/id_rsa.pub"

	fn := readsshpublickey.New()
	req := function.RunRequest{
		Arguments: function.NewArgumentsData([]attr.Value{basetypes.NewStringValue(tildePath)}),
	}
	resp := &function.RunResponse{
		Result: function.NewResultData(basetypes.NewStringNull()),
	}

	fn.Run(ctx, req, resp)

	g.Expect(resp.Error).To(BeNil(), "did not expect an error: %v", resp.Error)

	result := resp.Result.Value()
	g.Expect(result).NotTo(BeNil(), "expected a result value")
	stringResult, ok := result.(basetypes.StringValue)
	g.Expect(ok).To(BeTrue(), "expected result to be a StringValue")
	g.Expect(stringResult.ValueString()).To(Equal("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC"))
}

func TestReadSSHPublicKeyFileNotFound(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	fn := readsshpublickey.New()
	req := function.RunRequest{
		Arguments: function.NewArgumentsData([]attr.Value{basetypes.NewStringValue("/nonexistent/path/to/key.pub")}),
	}
	resp := &function.RunResponse{
		Result: function.NewResultData(basetypes.NewStringNull()),
	}

	fn.Run(ctx, req, resp)

	g.Expect(resp.Error).NotTo(BeNil(), "expected an error for non-existent file")
	g.Expect(resp.Error.Error()).To(ContainSubstring("failed to read SSH public key file"))
}
