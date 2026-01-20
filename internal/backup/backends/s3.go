package backends

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Backend represents an S3-compatible storage backend.
// Supports AWS S3, MinIO, Wasabi, and other S3-compatible services.
type S3Backend struct {
	Endpoint        string `json:"endpoint,omitempty"`
	Bucket          string `json:"bucket"`
	Prefix          string `json:"prefix,omitempty"`
	Region          string `json:"region,omitempty"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	UseSSL          bool   `json:"use_ssl"`
}

// Type returns the repository type.
func (b *S3Backend) Type() models.RepositoryType {
	return models.RepositoryTypeS3
}

// ToResticConfig converts the backend to a ResticConfig.
func (b *S3Backend) ToResticConfig(password string) ResticConfig {
	// Build the repository URL
	var repository string
	if b.Endpoint != "" {
		// Custom endpoint (MinIO, Wasabi, etc.)
		scheme := "http"
		if b.UseSSL {
			scheme = "https"
		}
		endpoint := b.Endpoint
		// Parse and rebuild to ensure proper formatting
		if u, err := url.Parse(b.Endpoint); err == nil && u.Host != "" {
			endpoint = u.Host
		}
		repository = fmt.Sprintf("s3:%s://%s/%s", scheme, endpoint, b.Bucket)
	} else {
		// AWS S3
		repository = fmt.Sprintf("s3:s3.amazonaws.com/%s", b.Bucket)
	}

	if b.Prefix != "" {
		repository = repository + "/" + b.Prefix
	}

	env := map[string]string{
		"AWS_ACCESS_KEY_ID":     b.AccessKeyID,
		"AWS_SECRET_ACCESS_KEY": b.SecretAccessKey,
	}

	if b.Region != "" {
		env["AWS_DEFAULT_REGION"] = b.Region
	}

	return ResticConfig{
		Repository: repository,
		Password:   password,
		Env:        env,
	}
}

// Validate checks if the configuration is valid.
func (b *S3Backend) Validate() error {
	if b.Bucket == "" {
		return errors.New("s3 backend: bucket is required")
	}
	if b.AccessKeyID == "" {
		return errors.New("s3 backend: access_key_id is required")
	}
	if b.SecretAccessKey == "" {
		return errors.New("s3 backend: secret_access_key is required")
	}
	return nil
}

// TestConnection tests the S3 backend connection by attempting to list the bucket.
func (b *S3Backend) TestConnection() error {
	if err := b.Validate(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build AWS config
	region := b.Region
	if region == "" {
		region = "us-east-1"
	}

	opts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			b.AccessKeyID,
			b.SecretAccessKey,
			"",
		)),
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return fmt.Errorf("s3 backend: failed to load config: %w", err)
	}

	// Create S3 client
	clientOpts := []func(*s3.Options){}

	if b.Endpoint != "" {
		scheme := "http"
		if b.UseSSL {
			scheme = "https"
		}
		endpoint := b.Endpoint
		if u, err := url.Parse(b.Endpoint); err == nil && u.Host != "" {
			endpoint = u.Host
		}
		endpointURL := fmt.Sprintf("%s://%s", scheme, endpoint)
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpointURL)
			o.UsePathStyle = true
		})
	}

	client := s3.NewFromConfig(cfg, clientOpts...)

	// Try to head the bucket to verify access
	_, err = client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(b.Bucket),
	})
	if err != nil {
		return fmt.Errorf("s3 backend: failed to access bucket: %w", err)
	}

	return nil
}
