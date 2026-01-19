package backends

import (
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"golang.org/x/crypto/ssh"
)

// SFTPBackend represents an SFTP storage backend.
type SFTPBackend struct {
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
