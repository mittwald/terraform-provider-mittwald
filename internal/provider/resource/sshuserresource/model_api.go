package sshuserresource

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/sshsftpuserclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/sshuserv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/ptrutil"
)

// FromAPIModel populates the ResourceModel from the API response.
func (m *ResourceModel) FromAPIModel(ctx context.Context, apiModel *sshuserv2.SshUser) (res diag.Diagnostics) {
	m.ID = types.StringValue(apiModel.Id)
	m.ProjectID = types.StringValue(apiModel.ProjectId)
	m.Description = types.StringValue(apiModel.Description)
	m.Username = types.StringValue(apiModel.UserName)
	m.CreatedAt = types.StringValue(apiModel.CreatedAt.Format(time.RFC3339))

	if apiModel.Active != nil {
		m.Active = types.BoolValue(*apiModel.Active)
	} else {
		m.Active = types.BoolValue(true)
	}

	if apiModel.ExpiresAt != nil {
		m.ExpiresAt = types.StringValue(apiModel.ExpiresAt.Format(time.RFC3339))
	} else {
		m.ExpiresAt = types.StringNull()
	}

	// Map public keys from API
	if len(apiModel.PublicKeys) > 0 {
		keys := make([]PublicKeyModel, 0, len(apiModel.PublicKeys))
		for _, pk := range apiModel.PublicKeys {
			keys = append(keys, PublicKeyModel{
				Key:     types.StringValue(pk.Key),
				Comment: types.StringValue(pk.Comment),
			})
		}
		m.SetPublicKeys(ctx, &res, keys)
	} else {
		m.SetPublicKeys(ctx, &res, nil)
	}

	return
}

// ToCreateRequest creates the API request for creating an SSH user.
func (m *ResourceModel) ToCreateRequest(ctx context.Context, d *diag.Diagnostics) sshsftpuserclientv2.CreateSSHUserRequest {
	body := sshsftpuserclientv2.CreateSSHUserRequestBody{
		Description: m.Description.ValueString(),
	}

	// Set authentication - either password or public keys
	auth := sshuserv2.Authentication{}

	if !m.Password.IsNull() && !m.Password.IsUnknown() && m.Password.ValueString() != "" {
		auth.AlternativeAuthenticationAlternative1 = &sshuserv2.AuthenticationAlternative1{
			Password: m.Password.ValueString(),
		}
	} else {
		// Use public keys
		publicKeys := m.GetPublicKeys(ctx, d)
		if len(publicKeys) > 0 {
			keys := make([]sshuserv2.PublicKey, 0, len(publicKeys))
			for _, pk := range publicKeys {
				keys = append(keys, sshuserv2.PublicKey{
					Key:     pk.Key.ValueString(),
					Comment: pk.Comment.ValueString(),
				})
			}
			auth.AlternativeAuthenticationAlternative2 = &sshuserv2.AuthenticationAlternative2{
				PublicKeys: keys,
			}
		} else {
			// Default to empty public keys if neither password nor keys are provided
			auth.AlternativeAuthenticationAlternative2 = &sshuserv2.AuthenticationAlternative2{
				PublicKeys: []sshuserv2.PublicKey{},
			}
		}
	}
	body.Authentication = auth

	// Set expiration if provided
	if !m.ExpiresAt.IsNull() && !m.ExpiresAt.IsUnknown() {
		expiresAt, err := time.Parse(time.RFC3339, m.ExpiresAt.ValueString())
		if err == nil {
			body.ExpiresAt = &expiresAt
		}
	}

	return sshsftpuserclientv2.CreateSSHUserRequest{
		ProjectID: m.ProjectID.ValueString(),
		Body:      body,
	}
}

// ToUpdateRequest creates the API request for updating an SSH user.
func (m *ResourceModel) ToUpdateRequest(ctx context.Context, d *diag.Diagnostics, current *ResourceModel) sshsftpuserclientv2.UpdateSSHUserRequest {
	body := sshsftpuserclientv2.UpdateSSHUserRequestBody{}

	// Update description if changed
	if !m.Description.Equal(current.Description) {
		body.Description = ptrutil.To(m.Description.ValueString())
	}

	// Update active state if changed
	if !m.Active.Equal(current.Active) {
		body.Active = ptrutil.To(m.Active.ValueBool())
	}

	// Update expiration if changed
	if !m.ExpiresAt.Equal(current.ExpiresAt) {
		if !m.ExpiresAt.IsNull() && !m.ExpiresAt.IsUnknown() {
			expiresAt, err := time.Parse(time.RFC3339, m.ExpiresAt.ValueString())
			if err == nil {
				body.ExpiresAt = &expiresAt
			}
		}
	}

	// Update password if set (password is write-only, so always update if provided)
	if !m.Password.IsNull() && !m.Password.IsUnknown() && m.Password.ValueString() != "" {
		body.Password = ptrutil.To(m.Password.ValueString())
	}

	// Update public keys if changed
	if !m.PublicKeys.Equal(current.PublicKeys) {
		publicKeys := m.GetPublicKeys(ctx, d)
		if publicKeys != nil {
			keys := make([]sshuserv2.PublicKey, 0, len(publicKeys))
			for _, pk := range publicKeys {
				keys = append(keys, sshuserv2.PublicKey{
					Key:     pk.Key.ValueString(),
					Comment: pk.Comment.ValueString(),
				})
			}
			body.PublicKeys = keys
		}
	}

	return sshsftpuserclientv2.UpdateSSHUserRequest{
		SSHUserID: m.ID.ValueString(),
		Body:      body,
	}
}

// ToGetRequest creates the API request for getting an SSH user.
func (m *ResourceModel) ToGetRequest() sshsftpuserclientv2.GetSSHUserRequest {
	return sshsftpuserclientv2.GetSSHUserRequest{
		SSHUserID: m.ID.ValueString(),
	}
}

// ToDeleteRequest creates the API request for deleting an SSH user.
func (m *ResourceModel) ToDeleteRequest() sshsftpuserclientv2.DeleteSSHUserRequest {
	return sshsftpuserclientv2.DeleteSSHUserRequest{
		SSHUserID: m.ID.ValueString(),
	}
}
