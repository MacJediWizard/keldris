package backends

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
)

// AzureBackend represents an Azure Blob Storage backend.
// Supports Azure public cloud, Azure Government, Azure China, and other sovereign clouds
// via the Endpoint field.
type AzureBackend struct {
	AccountName   string `json:"account_name"`
	AccountKey    string `json:"account_key"`
	ContainerName string `json:"container_name"`
	Endpoint      string `json:"endpoint,omitempty"`
	Prefix        string `json:"prefix,omitempty"`
}

// Type returns the repository type.
func (b *AzureBackend) Type() models.RepositoryType {
	return models.RepositoryTypeAzure
}

// ToResticConfig converts the backend to a ResticConfig.
func (b *AzureBackend) ToResticConfig(password string) ResticConfig {
	// Restic azure format: azure:container-name:/prefix
	repository := fmt.Sprintf("azure:%s:/", b.ContainerName)
	if b.Prefix != "" {
		repository = fmt.Sprintf("azure:%s:/%s", b.ContainerName, b.Prefix)
	}

	env := map[string]string{
		"AZURE_ACCOUNT_NAME": b.AccountName,
		"AZURE_ACCOUNT_KEY":  b.AccountKey,
	}

	if b.Endpoint != "" {
		env["AZURE_ENDPOINT_SUFFIX"] = b.Endpoint
	}

	return ResticConfig{
		Repository: repository,
		Password:   password,
		Env:        env,
	}
}

// Validate checks if the configuration is valid.
func (b *AzureBackend) Validate() error {
	if b.AccountName == "" {
		return errors.New("azure backend: account_name is required")
	}
	if b.AccountKey == "" {
		return errors.New("azure backend: account_key is required")
	}
	if b.ContainerName == "" {
		return errors.New("azure backend: container_name is required")
	}

	// Validate that the account key is valid base64
	if _, err := base64.StdEncoding.DecodeString(b.AccountKey); err != nil {
		return fmt.Errorf("azure backend: account_key is not valid base64: %w", err)
	}

	return nil
}

// TestConnection tests the Azure Blob Storage backend connection by listing the container.
func (b *AzureBackend) TestConnection() error {
	if err := b.Validate(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build the endpoint URL
	host := fmt.Sprintf("%s.blob.core.windows.net", b.AccountName)
	if b.Endpoint != "" {
		host = fmt.Sprintf("%s.blob.%s", b.AccountName, b.Endpoint)
	}
	reqURL := fmt.Sprintf("https://%s/%s?restype=container&comp=list&maxresults=1", host, b.ContainerName)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return fmt.Errorf("azure backend: failed to create request: %w", err)
	}

	// Set required headers for Azure SharedKey authentication
	now := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("x-ms-date", now)
	req.Header.Set("x-ms-version", "2020-10-02")

	// Sign the request with SharedKey
	authHeader, err := b.signRequest(req, host)
	if err != nil {
		return fmt.Errorf("azure backend: failed to sign request: %w", err)
	}
	req.Header.Set("Authorization", authHeader)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("azure backend: failed to connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("azure backend: container %q not found", b.ContainerName)
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("azure backend: authentication failed (status %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("azure backend: unexpected status %d", resp.StatusCode)
	}

	return nil
}

// signRequest creates a SharedKey authorization header for an Azure Storage request.
func (b *AzureBackend) signRequest(req *http.Request, host string) (string, error) {
	// Decode the account key
	keyBytes, err := base64.StdEncoding.DecodeString(b.AccountKey)
	if err != nil {
		return "", fmt.Errorf("invalid account key: %w", err)
	}

	// Build the canonicalized headers
	canonicalizedHeaders := fmt.Sprintf("x-ms-date:%s\nx-ms-version:%s",
		req.Header.Get("x-ms-date"),
		req.Header.Get("x-ms-version"))

	// Build the canonicalized resource
	// Format: /{account}/{path}\n{query params in alphabetical order}
	canonicalizedResource := fmt.Sprintf("/%s/%s", b.AccountName, b.ContainerName)

	// Add sorted query parameters
	queryParams := req.URL.Query()
	if len(queryParams) > 0 {
		// Azure requires query params sorted alphabetically
		keys := make([]string, 0, len(queryParams))
		for k := range queryParams {
			keys = append(keys, k)
		}
		sortStrings(keys)
		for _, k := range keys {
			canonicalizedResource += fmt.Sprintf("\n%s:%s", k, queryParams.Get(k))
		}
	}

	// Build the string to sign
	// https://learn.microsoft.com/en-us/rest/api/storageservices/authorize-with-shared-key
	stringToSign := strings.Join([]string{
		req.Method,                          // HTTP verb
		req.Header.Get("Content-Encoding"),  // Content-Encoding
		req.Header.Get("Content-Language"),   // Content-Language
		req.Header.Get("Content-Length"),     // Content-Length (empty string when 0)
		req.Header.Get("Content-MD5"),        // Content-MD5
		req.Header.Get("Content-Type"),       // Content-Type
		"",                                   // Date (empty when x-ms-date is used)
		req.Header.Get("If-Modified-Since"),  // If-Modified-Since
		req.Header.Get("If-Match"),           // If-Match
		req.Header.Get("If-None-Match"),      // If-None-Match
		req.Header.Get("If-Unmodified-Since"), // If-Unmodified-Since
		req.Header.Get("Range"),              // Range
		canonicalizedHeaders,
		canonicalizedResource,
	}, "\n")

	// HMAC-SHA256 sign
	mac := hmac.New(sha256.New, keyBytes)
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return fmt.Sprintf("SharedKey %s:%s", b.AccountName, signature), nil
}

// sortStrings sorts a slice of strings in place (simple insertion sort to avoid importing sort).
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
