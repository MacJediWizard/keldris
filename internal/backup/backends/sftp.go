package backends

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// SFTPBackend represents an SFTP storage backend.
type SFTPBackend struct {
	Host           string `json:"host"`
	Port           int    `json:"port,omitempty"`
	User           string `json:"user"`
	Path           string `json:"path"`
	Password       string `json:"password,omitempty"`
	PrivateKey     string `json:"private_key,omitempty"`
	HostKey        string `json:"host_key,omitempty"`         // Base64-encoded SSH public key
	KnownHostsFile string `json:"known_hosts_file,omitempty"` // Path to known_hosts file
	Host       string `json:"host"`
	Port       int    `json:"port,omitempty"`
	User       string `json:"user"`
	Path       string `json:"path"`
	Password   string `json:"password,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`
}

// Type returns the repository type.
func (b *SFTPBackend) Type() models.RepositoryType {
	return models.RepositoryTypeSFTP
}

// ToResticConfig converts the backend to a ResticConfig.
func (b *SFTPBackend) ToResticConfig(password string) ResticConfig {
	port := b.Port
	if port == 0 {
		port = 22
	}

	repository := fmt.Sprintf("sftp:%s@%s:%d%s", b.User, b.Host, port, b.Path)

	env := make(map[string]string)
	if b.PrivateKey != "" {
		// Restic uses SSH_AUTH_SOCK or standard SSH key locations
		// Private key can be passed via environment or SSH agent
		env["RESTIC_SFTP_ARGS"] = fmt.Sprintf("-i %s", b.PrivateKey)
	}

	return ResticConfig{
		Repository: repository,
		Password:   password,
		Env:        env,
	}
}

// Validate checks if the configuration is valid.
func (b *SFTPBackend) Validate() error {
	if b.Host == "" {
		return errors.New("sftp backend: host is required")
	}
	if b.User == "" {
		return errors.New("sftp backend: user is required")
	}
	if b.Path == "" {
		return errors.New("sftp backend: path is required")
	}
	if !filepath.IsAbs(b.Path) {
		return errors.New("sftp backend: path must be absolute")
	}
	return nil
}

// hostKeyCallback builds an ssh.HostKeyCallback based on the backend configuration.
// Priority: HostKey field > KnownHostsFile > error.
func (b *SFTPBackend) hostKeyCallback() (ssh.HostKeyCallback, error) {
	if b.HostKey != "" {
		hostKeyBytes, err := base64.StdEncoding.DecodeString(b.HostKey)
		if err != nil {
			return nil, fmt.Errorf("sftp backend: failed to decode host key: %w", err)
		}
		expectedKey, err := ssh.ParsePublicKey(hostKeyBytes)
		if err != nil {
			return nil, fmt.Errorf("sftp backend: failed to parse host key: %w", err)
		}
		return ssh.FixedHostKey(expectedKey), nil
	}

	if b.KnownHostsFile != "" {
		if _, err := os.Stat(b.KnownHostsFile); err != nil {
			return nil, fmt.Errorf("sftp backend: known_hosts file not found: %w", err)
		}
		callback, err := knownhosts.New(b.KnownHostsFile)
		if err != nil {
			return nil, fmt.Errorf("sftp backend: failed to parse known_hosts: %w", err)
		}
		return callback, nil
	}

	return nil, errors.New("sftp backend: host key verification required; provide host_key or known_hosts_file")
}

// TestConnection tests the SFTP backend connection by attempting to connect via SSH.
func (b *SFTPBackend) TestConnection() error {
	if err := b.Validate(); err != nil {
		return err
	}

	port := b.Port
	if port == 0 {
		port = 22
	}

	// Build auth methods
	var authMethods []ssh.AuthMethod

	if b.Password != "" {
		authMethods = append(authMethods, ssh.Password(b.Password))
	}

	if b.PrivateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(b.PrivateKey))
		if err != nil {
			return fmt.Errorf("sftp backend: failed to parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	hostKeyCallback, err := b.hostKeyCallback()
	if err != nil {
		return err
	}

	config := &ssh.ClientConfig{
		User:            b.User,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
	config := &ssh.ClientConfig{
		User:            b.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	addr := net.JoinHostPort(b.Host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", addr, 30*time.Second)
	if err != nil {
		return fmt.Errorf("sftp backend: failed to connect to %s: %w", addr, err)
	}
	defer conn.Close()

	c, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return fmt.Errorf("sftp backend: SSH handshake failed: %w", err)
	}
	client := ssh.NewClient(c, chans, reqs)
	defer client.Close()

	// Connection successful
	return nil
}

// FetchHostKey connects to an SSH server and returns its host public key
// as a base64-encoded string. This is used by the UI "Test Connection"
// flow to retrieve the key for user approval before saving.
func FetchHostKey(host string, port int) (string, error) {
	if host == "" {
		return "", errors.New("host is required")
	}
	if port <= 0 {
		port = 22
	}

	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	var capturedKey ssh.PublicKey
	config := &ssh.ClientConfig{
		User: "probe",
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			capturedKey = key
			return errors.New("host key captured") // abort after capturing
		},
		Timeout: 10 * time.Second,
	}

	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return "", fmt.Errorf("failed to connect to %s: %w", addr, err)
	}
	defer conn.Close()

	// The handshake will fail because our callback returns an error,
	// but we will have captured the host key.
	_, _, _, _ = ssh.NewClientConn(conn, addr, config)

	if capturedKey == nil {
		return "", fmt.Errorf("failed to retrieve host key from %s", addr)
	}

	return base64.StdEncoding.EncodeToString(capturedKey.Marshal()), nil
}
