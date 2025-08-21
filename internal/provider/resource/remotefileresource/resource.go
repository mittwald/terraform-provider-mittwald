package remotefileresource

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/appclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/containerclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/projectclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/userclientv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"path/filepath"
)

var _ resource.Resource = &Resource{}
var _ resource.ResourceWithImportState = &Resource{}

func New() resource.Resource {
	return &Resource{}
}

type Resource struct {
	client mittwaldv2.Client
}

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_remote_file"
}

func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource allows you to create and manage files on a remote server via SSH.\n\n" +
			"You can specify either a container_id or an app_id to determine which server to connect to. " +
			"The SSH hostname is dynamically determined from the project that the app or container belongs to, " +
			"and the SSH username defaults to the currently authenticated user if not specified.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the remote file resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"container_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The ID of the container to connect to. Either container_id or app_id must be specified.",
			},
			"stack_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The ID of the stack that the container belongs to. Required when container_id is specified.",
			},
			"app_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The ID of the app to connect to. Either container_id or app_id must be specified.",
			},
			"ssh_user": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The SSH username to use for the connection. If not specified, it will default to the currently authenticated user.",
			},
			"ssh_private_key": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The SSH private key to use for the connection. If not specified, it will default to the contents ~/.ssh/id_rsa; use the file function to specify a file path instead.",
			},
			"path": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The path of the file on the remote server.",
			},
			"contents": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The contents of the file.",
			},
		},
	}
}

func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.ClientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that either container_id or app_id is specified, but not both
	if data.ContainerID.IsNull() && data.AppID.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("container_id"),
			"Missing Resource Reference",
			"Either container_id or app_id must be specified.",
		)
		return
	}
	if !data.ContainerID.IsNull() && !data.AppID.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("container_id"),
			"Invalid Resource Reference",
			"Only one of container_id or app_id can be specified.",
		)
		return
	}

	// Validate that stack_id is specified when container_id is specified
	if !data.ContainerID.IsNull() && data.StackID.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("stack_id"),
			"Missing Stack ID",
			"stack_id must be specified when container_id is specified.",
		)
		return
	}

	// Create the file on the remote server
	err := r.createOrUpdateFile(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Remote File",
			fmt.Sprintf("Could not create file at %s: %s", data.Path.ValueString(), err),
		)
		return
	}

	// Generate a unique ID for the resource
	var resourceID string
	if !data.ContainerID.IsNull() {
		resourceID = fmt.Sprintf("container-%s-%s-%s", data.StackID.ValueString(), data.ContainerID.ValueString(), data.Path.ValueString())
	} else {
		resourceID = fmt.Sprintf("app-%s-%s", data.AppID.ValueString(), data.Path.ValueString())
	}
	data.ID = types.StringValue(resourceID)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if the file exists on the remote server
	exists, contents, err := r.readFile(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Remote File",
			fmt.Sprintf("Could not read file at %s: %s", data.Path.ValueString(), err),
		)
		return
	}

	if !exists {
		// File doesn't exist, remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	// Update the contents in the state
	data.Contents = types.StringValue(contents)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the file on the remote server
	err := r.createOrUpdateFile(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Remote File",
			fmt.Sprintf("Could not update file at %s: %s", data.Path.ValueString(), err),
		)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the file from the remote server
	err := r.deleteFile(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Remote File",
			fmt.Sprintf("Could not delete file at %s: %s", data.Path.ValueString(), err),
		)
		return
	}
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helper functions for SSH operations

func (r *Resource) getSSHConnectionDetails(ctx context.Context, data *ResourceModel) (string, string, error) {
	projectID, shortID, err := r.determineProjectAndTargetID(ctx, data)
	if err != nil {
		return "", "", err
	}

	sshHost, err := r.determineSSHHostname(ctx, projectID)
	if err != nil {
		return "", "", err
	}

	username, err := r.determineSSHUsername(ctx, data)
	if err != nil {
		return "", "", fmt.Errorf("error determining SSH username: %w", err)
	}

	// Always format as "<username>@<shortid>"
	sshUser := fmt.Sprintf("%s@%s", username, shortID)

	tflog.Debug(ctx, "Using SSH connection details", map[string]interface{}{
		"host": sshHost,
		"user": sshUser,
	})

	return sshHost, sshUser, nil
}

func (r *Resource) determineSSHHostname(ctx context.Context, projectID string) (string, error) {
	// Get project details to determine SSH host
	projectRequest := projectclientv2.GetProjectRequest{ProjectID: projectID}
	project, _, err := r.client.Project().GetProject(ctx, projectRequest)
	if err != nil {
		return "", fmt.Errorf("error getting project details: %w", err)
	}

	// Determine SSH host from project details
	if project.ClusterID != nil && project.ClusterDomain != nil {
		return fmt.Sprintf("ssh.%s.%s", *project.ClusterID, *project.ClusterDomain), nil
	}

	return "", fmt.Errorf("project %s does not have cluster information", projectID)
}

func (r *Resource) determineSSHUsername(ctx context.Context, data *ResourceModel) (string, error) {
	if !data.SSHUser.IsNull() {
		return data.SSHUser.ValueString(), nil
	}

	// Use the currently logged-in user's email address
	// Get the user from the client
	userClient := r.client.User()
	user, _, err := userClient.GetUser(ctx, userclientv2.GetUserRequest{UserID: "self"})
	if err != nil {
		return "", fmt.Errorf("error getting user details: %w", err)
	}

	if user.Email != nil {
		return *user.Email, nil
	}

	return "", fmt.Errorf("user email is not available, cannot determine SSH username")
}

func (r *Resource) determineProjectAndTargetID(ctx context.Context, data *ResourceModel) (string, string, error) {
	// Get project ID from either container ID or app ID
	if !data.ContainerID.IsNull() && !data.StackID.IsNull() {
		containerClient := r.client.Container()
		container, _, err := containerClient.GetService(ctx, containerclientv2.GetServiceRequest{
			ServiceID: data.ContainerID.ValueString(),
			StackID:   data.StackID.ValueString(),
		})

		if err != nil {
			return "", "", fmt.Errorf("error getting container details: %w", err)
		}

		return container.ProjectId, container.ShortId, nil
	}

	if !data.AppID.IsNull() {
		// Get project ID from app ID
		appClient := r.client.App()
		appInstallation, _, err := appClient.GetAppinstallation(ctx, appclientv2.GetAppinstallationRequest{
			AppInstallationID: data.AppID.ValueString(),
		})
		if err != nil {
			return "", "", fmt.Errorf("error getting app installation details: %w", err)
		}

		return appInstallation.ProjectId, appInstallation.ShortId, nil
	}

	return "", "", fmt.Errorf("either container_id+stack_id or app_id must be specified")
}

func (r *Resource) createSSHClient(ctx context.Context, data *ResourceModel) (*ssh.Client, error) {
	// Get SSH connection details
	host, user, err := r.getSSHConnectionDetails(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("failed to get SSH connection details: %w", err)
	}

	// Get private key
	privateKey := ""
	if !data.SSHPrivateKey.IsNull() {
		privateKey = data.SSHPrivateKey.ValueString()
	}

	// If privateKey is empty, use the default ~/.ssh/id_rsa
	if privateKey == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("could not get user home directory: %w", err)
		}
		keyBytes, err := os.ReadFile(fmt.Sprintf("%s/.ssh/id_rsa", homeDir))
		if err != nil {
			return nil, fmt.Errorf("unable to read default private key: %w", err)
		}
		privateKey = string(keyBytes)
	}

	tflog.Debug(ctx, "Using SSH private key")

	signer, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return nil, fmt.Errorf("unable to parse private key: %w", err)
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Note: In production, use proper host key verification
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", host), config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}

	return client, nil
}

func (r *Resource) createOrUpdateFile(ctx context.Context, resource ResourceModel) error {
	filePath := resource.Path.ValueString()
	contents := resource.Contents.ValueString()

	tflog.Debug(ctx, "Creating/updating remote file", map[string]interface{}{
		"path": filePath,
	})

	client, err := r.createSSHClient(ctx, &resource)
	if err != nil {
		return err
	}
	defer client.Close()

	// Create an SFTP client
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	// Ensure the directory exists
	dir := filepath.Dir(filePath)
	if dir != "." {
		err = sftpClient.MkdirAll(dir)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create or truncate the file
	file, err := sftpClient.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Write the contents to the file
	_, err = file.Write([]byte(contents))
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

func (r *Resource) readFile(ctx context.Context, resource ResourceModel) (bool, string, error) {
	filePath := resource.Path.ValueString()

	tflog.Debug(ctx, "Reading remote file", map[string]interface{}{
		"path": filePath,
	})

	client, err := r.createSSHClient(ctx, &resource)
	if err != nil {
		return false, "", err
	}
	defer client.Close()

	// Create an SFTP client
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return false, "", fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	// Check if the file exists
	fileInfo, err := sftpClient.Stat(filePath)
	if err != nil {
		// File doesn't exist or other error
		if os.IsNotExist(err) {
			return false, "", nil
		}
		return false, "", fmt.Errorf("failed to check if file exists: %w", err)
	}

	// Make sure it's a regular file
	if !fileInfo.Mode().IsRegular() {
		return false, "", fmt.Errorf("%s is not a regular file", filePath)
	}

	// Open the file for reading
	file, err := sftpClient.Open(filePath)
	if err != nil {
		return false, "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read the file contents
	contents, err := io.ReadAll(file)
	if err != nil {
		return true, "", fmt.Errorf("failed to read file: %w", err)
	}

	return true, string(contents), nil
}

func (r *Resource) deleteFile(ctx context.Context, resource ResourceModel) error {
	filePath := resource.Path.ValueString()

	tflog.Debug(ctx, "Deleting remote file", map[string]interface{}{
		"path": filePath,
	})

	client, err := r.createSSHClient(ctx, &resource)
	if err != nil {
		return err
	}
	defer client.Close()

	// Create an SFTP client
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	// Check if the file exists before attempting to remove it
	_, err = sftpClient.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, nothing to do
			return nil
		}
		return fmt.Errorf("failed to check if file exists: %w", err)
	}

	// Remove the file
	err = sftpClient.Remove(filePath)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// generateResourceID creates a unique ID for the resource
func generateResourceID(stackID, containerID, appID, path string) string {
	h := sha256.New()
	if containerID != "" {
		h.Write([]byte(fmt.Sprintf("container:%s:%s:%s", stackID, containerID, path)))
	} else {
		h.Write([]byte(fmt.Sprintf("app:%s:%s", appID, path)))
	}
	return hex.EncodeToString(h.Sum(nil))
}
