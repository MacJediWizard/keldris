package backends

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"path/filepath"
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
		// Generate a valid host key so host key validation passes
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}
		signer, err := ssh.NewSignerFromKey(key)
		if err != nil {
			t.Fatalf("failed to create signer: %v", err)
		}
		hostKey := base64.StdEncoding.EncodeToString(signer.PublicKey().Marshal())

		b := &SFTPBackend{
			Host:       "localhost",
			User:       "user",
			Path:       "/backups",
			HostKey:    hostKey,
			PrivateKey: "not-a-valid-key",
		}

		err = b.TestConnection()
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
// authentication with the given user/password. Returns the host:port, the
// base64-encoded host public key, and a cleanup function.
func startTestSSHServer(t *testing.T, wantUser, wantPass string) (string, string, func()) {
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

	hostKeyB64 := base64.StdEncoding.EncodeToString(signer.PublicKey().Marshal())

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

	return listener.Addr().String(), hostKeyB64, func() { listener.Close() }
}

func TestSFTPBackend_TestConnection_PasswordSuccess(t *testing.T) {
	addr, hostKey, cleanup := startTestSSHServer(t, "testuser", "testpass")
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
		HostKey:  hostKey,
	}

	err = b.TestConnection()
	if err != nil {
		t.Errorf("TestConnection() error = %v, want nil", err)
	}
}

func TestSFTPBackend_TestConnection_WrongPassword(t *testing.T) {
	addr, hostKey, cleanup := startTestSSHServer(t, "testuser", "correct-pass")
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
		HostKey:  hostKey,
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

	// Generate a valid host key so validation passes before connection attempt
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}
	hostKey := base64.StdEncoding.EncodeToString(signer.PublicKey().Marshal())

	host, port, _ := net.SplitHostPort(addr)
	var portNum int
	fmt.Sscanf(port, "%d", &portNum)

	b := &SFTPBackend{
		Host:     host,
		Port:     portNum,
		User:     "user",
		Path:     "/backups",
		Password: "pass",
		HostKey:  hostKey,
	}

	err = b.TestConnection()
	if err == nil {
		t.Error("TestConnection() expected error for refused connection, got nil")
	}
	if !contains(err.Error(), "failed to connect") {
		t.Errorf("TestConnection() error = %q, want to contain 'failed to connect'", err.Error())
	}
}

func TestSFTPBackend_HostKeyVerification(t *testing.T) {
	t.Run("rejects when no host key or known_hosts provided", func(t *testing.T) {
		b := &SFTPBackend{
			Host:     "localhost",
			User:     "user",
			Path:     "/backups",
			Password: "pass",
		}

		err := b.TestConnection()
		if err == nil {
			t.Fatal("TestConnection() expected error, got nil")
		}
		if !contains(err.Error(), "host key verification required") {
			t.Errorf("error = %q, want to contain 'host key verification required'", err.Error())
		}
	})

	t.Run("rejects invalid base64 host key", func(t *testing.T) {
		b := &SFTPBackend{
			Host:     "localhost",
			User:     "user",
			Path:     "/backups",
			Password: "pass",
			HostKey:  "not-valid-base64!!!",
		}

		err := b.TestConnection()
		if err == nil {
			t.Fatal("TestConnection() expected error, got nil")
		}
		if !contains(err.Error(), "failed to decode host key") {
			t.Errorf("error = %q, want to contain 'failed to decode host key'", err.Error())
		}
	})

	t.Run("rejects invalid public key bytes", func(t *testing.T) {
		b := &SFTPBackend{
			Host:     "localhost",
			User:     "user",
			Path:     "/backups",
			Password: "pass",
			HostKey:  base64.StdEncoding.EncodeToString([]byte("not-a-real-key")),
		}

		err := b.TestConnection()
		if err == nil {
			t.Fatal("TestConnection() expected error, got nil")
		}
		if !contains(err.Error(), "failed to parse host key") {
			t.Errorf("error = %q, want to contain 'failed to parse host key'", err.Error())
		}
	})

	t.Run("rejects mismatched host key", func(t *testing.T) {
		addr, _, cleanup := startTestSSHServer(t, "testuser", "testpass")
		defer cleanup()

		// Generate a different key for mismatch
		wrongKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}
		wrongSigner, err := ssh.NewSignerFromKey(wrongKey)
		if err != nil {
			t.Fatalf("failed to create signer: %v", err)
		}
		wrongHostKey := base64.StdEncoding.EncodeToString(wrongSigner.PublicKey().Marshal())

		host, port, _ := net.SplitHostPort(addr)
		var portNum int
		fmt.Sscanf(port, "%d", &portNum)

		b := &SFTPBackend{
			Host:     host,
			Port:     portNum,
			User:     "testuser",
			Path:     "/backups",
			Password: "testpass",
			HostKey:  wrongHostKey,
		}

		err = b.TestConnection()
		if err == nil {
			t.Fatal("TestConnection() expected error for mismatched host key, got nil")
		}
		if !contains(err.Error(), "SSH handshake failed") {
			t.Errorf("error = %q, want to contain 'SSH handshake failed'", err.Error())
		}
	})

	t.Run("accepts correct host key", func(t *testing.T) {
		addr, hostKey, cleanup := startTestSSHServer(t, "testuser", "testpass")
		defer cleanup()

		host, port, _ := net.SplitHostPort(addr)
		var portNum int
		fmt.Sscanf(port, "%d", &portNum)

		b := &SFTPBackend{
			Host:     host,
			Port:     portNum,
			User:     "testuser",
			Path:     "/backups",
			Password: "testpass",
			HostKey:  hostKey,
		}

		err := b.TestConnection()
		if err != nil {
			t.Errorf("TestConnection() error = %v, want nil", err)
		}
	})

	t.Run("rejects nonexistent known_hosts file", func(t *testing.T) {
		b := &SFTPBackend{
			Host:           "localhost",
			User:           "user",
			Path:           "/backups",
			Password:       "pass",
			KnownHostsFile: "/nonexistent/known_hosts",
		}

		err := b.TestConnection()
		if err == nil {
			t.Fatal("TestConnection() expected error, got nil")
		}
		if !contains(err.Error(), "known_hosts file not found") {
			t.Errorf("error = %q, want to contain 'known_hosts file not found'", err.Error())
		}
	})

	t.Run("uses known_hosts file for verification", func(t *testing.T) {
		addr, _, cleanup := startTestSSHServer(t, "testuser", "testpass")
		defer cleanup()

		host, port, _ := net.SplitHostPort(addr)
		var portNum int
		fmt.Sscanf(port, "%d", &portNum)

		// Fetch the host key and write a known_hosts file
		hostKeyB64, err := FetchHostKey(host, portNum)
		if err != nil {
			t.Fatalf("FetchHostKey() error = %v", err)
		}

		hostKeyBytes, err := base64.StdEncoding.DecodeString(hostKeyB64)
		if err != nil {
			t.Fatalf("failed to decode host key: %v", err)
		}
		pubKey, err := ssh.ParsePublicKey(hostKeyBytes)
		if err != nil {
			t.Fatalf("failed to parse public key: %v", err)
		}

		knownHostsPath := filepath.Join(t.TempDir(), "known_hosts")
		line := fmt.Sprintf("[%s]:%s %s %s\n", host, port, pubKey.Type(), base64.StdEncoding.EncodeToString(pubKey.Marshal()))
		if err := os.WriteFile(knownHostsPath, []byte(line), 0600); err != nil {
			t.Fatalf("failed to write known_hosts: %v", err)
		}

		b := &SFTPBackend{
			Host:           host,
			Port:           portNum,
			User:           "testuser",
			Path:           "/backups",
			Password:       "testpass",
			KnownHostsFile: knownHostsPath,
		}

		err = b.TestConnection()
		if err != nil {
			t.Errorf("TestConnection() with known_hosts error = %v, want nil", err)
		}
	})

	t.Run("host key takes priority over known_hosts", func(t *testing.T) {
		addr, hostKey, cleanup := startTestSSHServer(t, "testuser", "testpass")
		defer cleanup()

		host, port, _ := net.SplitHostPort(addr)
		var portNum int
		fmt.Sscanf(port, "%d", &portNum)

		b := &SFTPBackend{
			Host:           host,
			Port:           portNum,
			User:           "testuser",
			Path:           "/backups",
			Password:       "testpass",
			HostKey:        hostKey,
			KnownHostsFile: "/nonexistent/known_hosts", // should be ignored
		}

		err := b.TestConnection()
		if err != nil {
			t.Errorf("TestConnection() error = %v, want nil (HostKey should take priority)", err)
		}
	})
}

func TestFetchHostKey(t *testing.T) {
	t.Run("fetches host key from server", func(t *testing.T) {
		addr, expectedKey, cleanup := startTestSSHServer(t, "user", "pass")
		defer cleanup()

		host, port, _ := net.SplitHostPort(addr)
		var portNum int
		fmt.Sscanf(port, "%d", &portNum)

		got, err := FetchHostKey(host, portNum)
		if err != nil {
			t.Fatalf("FetchHostKey() error = %v", err)
		}
		if got != expectedKey {
			t.Errorf("FetchHostKey() = %q, want %q", got, expectedKey)
		}
	})

	t.Run("returns error for empty host", func(t *testing.T) {
		_, err := FetchHostKey("", 22)
		if err == nil {
			t.Fatal("FetchHostKey() expected error for empty host, got nil")
		}
		if !contains(err.Error(), "host is required") {
			t.Errorf("error = %q, want to contain 'host is required'", err.Error())
		}
	})

	t.Run("returns error for unreachable host", func(t *testing.T) {
		// Use a port that's not listening
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("failed to listen: %v", err)
		}
		addr := listener.Addr().String()
		listener.Close()

		host, port, _ := net.SplitHostPort(addr)
		var portNum int
		fmt.Sscanf(port, "%d", &portNum)

		_, err = FetchHostKey(host, portNum)
		if err == nil {
			t.Fatal("FetchHostKey() expected error for unreachable host, got nil")
		}
		if !contains(err.Error(), "failed to connect") {
			t.Errorf("error = %q, want to contain 'failed to connect'", err.Error())
		}
	})

	t.Run("defaults port to 22", func(t *testing.T) {
		// We can't easily test port 22 in CI, but we can verify
		// the function doesn't panic with port 0
		_, err := FetchHostKey("192.0.2.1", 0) // TEST-NET, should timeout or fail
		if err == nil {
			t.Fatal("FetchHostKey() expected error for unreachable host, got nil")
		}
	})
}
