package backends

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
)

// DropboxBackend represents a Dropbox storage backend using rclone.
// This backend requires rclone to be installed and configured.
type DropboxBackend struct {
	RemoteName string `json:"remote_name"`
	Path       string `json:"path"`
	Token      string `json:"token,omitempty"`
	AppKey     string `json:"app_key,omitempty"`
	AppSecret  string `json:"app_secret,omitempty"`
}

// Type returns the repository type.
func (b *DropboxBackend) Type() models.RepositoryType {
	return models.RepositoryTypeDropbox
}

// ToResticConfig converts the backend to a ResticConfig.
// Dropbox is accessed via rclone, so we use the rclone backend format.
func (b *DropboxBackend) ToResticConfig(password string) ResticConfig {
	// Format: rclone:remote:path
	path := b.Path
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	repository := fmt.Sprintf("rclone:%s:%s", b.RemoteName, path)

	env := make(map[string]string)

	// If token is provided, set it via environment
	if b.Token != "" {
		env["RCLONE_CONFIG_"+strings.ToUpper(b.RemoteName)+"_TYPE"] = "dropbox"
		env["RCLONE_CONFIG_"+strings.ToUpper(b.RemoteName)+"_TOKEN"] = b.Token
	}

	// If app credentials are provided
	if b.AppKey != "" {
		env["RCLONE_CONFIG_"+strings.ToUpper(b.RemoteName)+"_CLIENT_ID"] = b.AppKey
	}
	if b.AppSecret != "" {
		env["RCLONE_CONFIG_"+strings.ToUpper(b.RemoteName)+"_CLIENT_SECRET"] = b.AppSecret
	}

	return ResticConfig{
		Repository: repository,
		Password:   password,
		Env:        env,
	}
}

// Validate checks if the configuration is valid.
func (b *DropboxBackend) Validate() error {
	if b.RemoteName == "" {
		return errors.New("dropbox backend: remote_name is required")
	}
	// Either token or preconfigured rclone remote must exist
	return nil
}

// TestConnection tests the Dropbox backend connection via rclone.
func (b *DropboxBackend) TestConnection() error {
	if err := b.Validate(); err != nil {
		return err
	}

	// Check if rclone is installed
	if _, err := exec.LookPath("rclone"); err != nil {
		return errors.New("dropbox backend: rclone is not installed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build the remote path
	path := b.Path
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	remotePath := fmt.Sprintf("%s:%s", b.RemoteName, path)

	// Use rclone lsd to test connection
	cmd := exec.CommandContext(ctx, "rclone", "lsd", remotePath, "--max-depth", "1")

	// Set environment variables for rclone config
	if b.Token != "" {
		cmd.Env = append(cmd.Environ(),
			"RCLONE_CONFIG_"+strings.ToUpper(b.RemoteName)+"_TYPE=dropbox",
			"RCLONE_CONFIG_"+strings.ToUpper(b.RemoteName)+"_TOKEN="+b.Token,
		)
	}
	if b.AppKey != "" {
		cmd.Env = append(cmd.Environ(),
			"RCLONE_CONFIG_"+strings.ToUpper(b.RemoteName)+"_CLIENT_ID="+b.AppKey,
		)
	}
	if b.AppSecret != "" {
		cmd.Env = append(cmd.Environ(),
			"RCLONE_CONFIG_"+strings.ToUpper(b.RemoteName)+"_CLIENT_SECRET="+b.AppSecret,
		)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return fmt.Errorf("dropbox backend: connection test failed: %s", errMsg)
	}

	return nil
}
