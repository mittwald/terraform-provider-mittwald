package sshutil

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// HostKeyInfo contains information about an SSH host key.
type HostKeyInfo struct {
	KeyType string
	Key     string
}

// FetchHostKey connects to an SSH server and retrieves its host key.
// The address should be in the format "hostname:port" (e.g., "ssh.example.com:22").
func FetchHostKey(ctx context.Context, address string) (*HostKeyInfo, error) {
	var hostKey ssh.PublicKey

	// Create a custom host key callback that captures the key
	hostKeyCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		hostKey = key
		// Return an error to abort the connection after getting the key
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
		return nil, fmt.Errorf("failed to connect to %s: %w", address, err)
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
		keyType := hostKey.Type()
		keyData := ssh.MarshalAuthorizedKey(hostKey)
		keyStr := strings.TrimSpace(string(keyData))

		// Extract just the base64 key part (remove type prefix and any comment)
		parts := strings.Fields(keyStr)
		keyBase64 := ""
		if len(parts) >= 2 {
			keyBase64 = parts[1]
		}

		return &HostKeyInfo{
			KeyType: keyType,
			Key:     keyBase64,
		}, nil
	}

	// If we didn't get a host key, return the actual error
	if err != nil {
		return nil, fmt.Errorf("SSH handshake failed: %w", err)
	}

	return nil, fmt.Errorf("no host key received from %s", address)
}
