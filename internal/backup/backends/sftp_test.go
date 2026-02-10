package backends

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"net"
	"testing"

	"github.com/MacJediWizard/keldris/internal/models"
	"golang.org/x/crypto/ssh"
)

func TestSFTPBackend_Init(t *testing.T) {
	b := &SFTPBackend{
		Host: "backup.example.com",
		User: "backupuser",
		Path: "/var/backups/restic",
	}
	if b.Type() != models.RepositoryTypeSFTP {
		t.Errorf("Type() = %v, want %v", b.Type(), models.RepositoryTypeSFTP)
	}
}

func TestSFTPBackend_Validate(t *testing.T) {
	tests := []struct {
		name    string
		backend SFTPBackend
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid minimal",
			backend: SFTPBackend{
				Host: "backup.example.com",
				User: "backupuser",
				Path: "/var/backups/restic",
			},
			wantErr: false,
		},
		{
			name: "valid with port and password",
			backend: SFTPBackend{
				Host:     "backup.example.com",
				Port:     2222,
				User:     "backupuser",
				Path:     "/var/backups",
				Password: "secret",
			},
			wantErr: false,
		},
		{
			name: "valid with private key",
			backend: SFTPBackend{
				Host:       "backup.example.com",
				User:       "backupuser",
				Path:       "/backups",
				PrivateKey: "-----BEGIN OPENSSH PRIVATE KEY-----\nfake\n-----END OPENSSH PRIVATE KEY-----",
			},
			wantErr: false,
		},
		{
			name: "missing host",
			backend: SFTPBackend{
				User: "backupuser",
				Path: "/var/backups/restic",
			},
			wantErr: true,
			errMsg:  "host is required",
		},
		{
			name: "missing user",
			backend: SFTPBackend{
				Host: "backup.example.com",
				Path: "/var/backups/restic",
			},
			wantErr: true,
			errMsg:  "user is required",
		},
		{
			name: "missing path",
			backend: SFTPBackend{
				Host: "backup.example.com",
				User: "backupuser",
			},
			wantErr: true,
			errMsg:  "path is required",
		},
		{
			name: "relative path",
			backend: SFTPBackend{
				Host: "backup.example.com",
				User: "backupuser",
				Path: "backups/restic",
			},
			wantErr: true,
			errMsg:  "path must be absolute",
		},
		{
			name:    "all fields empty",
			backend: SFTPBackend{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.backend.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if got := err.Error(); !contains(got, tt.errMsg) {
					t.Errorf("Validate() error = %q, want to contain %q", got, tt.errMsg)
				}
			}
		})
	}
}

func TestSFTPBackend_KeyAuth(t *testing.T) {
	t.Run("config with private key sets RESTIC_SFTP_ARGS", func(t *testing.T) {
		b := &SFTPBackend{
			Host:       "backup.example.com",
			User:       "backupuser",
			Path:       "/backups",
			PrivateKey: "/home/user/.ssh/id_rsa",
		}

		cfg := b.ToResticConfig("pass")

		if cfg.Env["RESTIC_SFTP_ARGS"] != "-i /home/user/.ssh/id_rsa" {
			t.Errorf("RESTIC_SFTP_ARGS = %v, want -i /home/user/.ssh/id_rsa", cfg.Env["RESTIC_SFTP_ARGS"])
		}
	})

	t.Run("config without private key has no RESTIC_SFTP_ARGS", func(t *testing.T) {
		b := &SFTPBackend{
			Host: "backup.example.com",
			User: "backupuser",
			Path: "/backups",
		}

		cfg := b.ToResticConfig("pass")

		if _, ok := cfg.Env["RESTIC_SFTP_ARGS"]; ok {
			t.Error("expected RESTIC_SFTP_ARGS to not be set when no private key")
		}
	})

	t.Run("test connection fails with invalid private key", func(t *testing.T) {
		b := &SFTPBackend{
			Host:       "localhost",
			User:       "user",
			Path:       "/backups",
			PrivateKey: "not-a-valid-key",
		}

		err := b.TestConnection()
		if err == nil {
			t.Error("TestConnection() expected error with invalid private key, got nil")
		}
		if !contains(err.Error(), "failed to parse private key") {
			t.Errorf("TestConnection() error = %q, want to contain 'failed to parse private key'", err.Error())
		}
	})
}

func TestSFTPBackend_PasswordAuth(t *testing.T) {
	t.Run("default port 22", func(t *testing.T) {
		b := &SFTPBackend{
			Host:     "backup.example.com",
			User:     "backupuser",
			Path:     "/var/backups/restic",
			Password: "mypassword",
		}

		cfg := b.ToResticConfig("repopass")

		expected := "sftp:backupuser@backup.example.com:22/var/backups/restic"
		if cfg.Repository != expected {
			t.Errorf("Repository = %v, want %v", cfg.Repository, expected)
		}
		if cfg.Password != "repopass" {
			t.Errorf("Password = %v, want repopass", cfg.Password)
		}
	})

	t.Run("custom port", func(t *testing.T) {
		b := &SFTPBackend{
			Host:     "backup.example.com",
			Port:     2222,
			User:     "backupuser",
			Path:     "/var/backups/restic",
			Password: "mypassword",
		}

		cfg := b.ToResticConfig("repopass")

		expected := "sftp:backupuser@backup.example.com:2222/var/backups/restic"
		if cfg.Repository != expected {
			t.Errorf("Repository = %v, want %v", cfg.Repository, expected)
		}
	})

	t.Run("password auth does not set RESTIC_SFTP_ARGS", func(t *testing.T) {
		b := &SFTPBackend{
			Host:     "backup.example.com",
			User:     "user",
			Path:     "/backups",
			Password: "pass",
		}

		cfg := b.ToResticConfig("repopass")

		if _, ok := cfg.Env["RESTIC_SFTP_ARGS"]; ok {
			t.Error("expected RESTIC_SFTP_ARGS to not be set for password auth")
		}
	})

	t.Run("test connection fails with invalid config", func(t *testing.T) {
		b := &SFTPBackend{}

		err := b.TestConnection()
		if err == nil {
			t.Error("TestConnection() expected error for invalid config, got nil")
		}
	})
}

// startTestSSHServer starts a local SSH server for testing. It accepts password
// authentication with the given user/password. Returns the host:port and a
// cleanup function.
func startTestSSHServer(t *testing.T, wantUser, wantPass string) (string, func()) {
	t.Helper()

	// Generate host key
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate host key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == wantUser && string(pass) == wantPass {
				return nil, nil
			}
			return nil, fmt.Errorf("invalid credentials")
		},
	}
	config.AddHostKey(signer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // listener closed
			}

			go func(c net.Conn) {
				defer c.Close()
				sconn, chans, reqs, err := ssh.NewServerConn(c, config)
				if err != nil {
					return
				}
				defer sconn.Close()
				go ssh.DiscardRequests(reqs)
				for ch := range chans {
					ch.Reject(ssh.Prohibited, "no channels allowed in test")
				}
			}(conn)
		}
	}()

	return listener.Addr().String(), func() { listener.Close() }
}

func TestSFTPBackend_TestConnection_PasswordSuccess(t *testing.T) {
	addr, cleanup := startTestSSHServer(t, "testuser", "testpass")
	defer cleanup()

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("failed to split host:port: %v", err)
	}
	var portNum int
	fmt.Sscanf(port, "%d", &portNum)

	b := &SFTPBackend{
		Host:     host,
		Port:     portNum,
		User:     "testuser",
		Path:     "/backups",
		Password: "testpass",
	}

	err = b.TestConnection()
	if err != nil {
		t.Errorf("TestConnection() error = %v, want nil", err)
	}
}

func TestSFTPBackend_TestConnection_WrongPassword(t *testing.T) {
	addr, cleanup := startTestSSHServer(t, "testuser", "correct-pass")
	defer cleanup()

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("failed to split host:port: %v", err)
	}
	var portNum int
	fmt.Sscanf(port, "%d", &portNum)

	b := &SFTPBackend{
		Host:     host,
		Port:     portNum,
		User:     "testuser",
		Path:     "/backups",
		Password: "wrong-pass",
	}

	err = b.TestConnection()
	if err == nil {
		t.Error("TestConnection() expected error for wrong password, got nil")
	}
	if !contains(err.Error(), "SSH handshake failed") {
		t.Errorf("TestConnection() error = %q, want to contain 'SSH handshake failed'", err.Error())
	}
}

func TestSFTPBackend_TestConnection_RefusedPort(t *testing.T) {
	// Listen and immediately close to get a port that's not in use
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	host, port, _ := net.SplitHostPort(addr)
	var portNum int
	fmt.Sscanf(port, "%d", &portNum)

	b := &SFTPBackend{
		Host:     host,
		Port:     portNum,
		User:     "user",
		Path:     "/backups",
		Password: "pass",
	}

	err = b.TestConnection()
	if err == nil {
		t.Error("TestConnection() expected error for refused connection, got nil")
	}
	if !contains(err.Error(), "failed to connect") {
		t.Errorf("TestConnection() error = %q, want to contain 'failed to connect'", err.Error())
	}
}
