// Package backup provides cloud restore functionality for uploading restored files to cloud storage.
package backup

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/MacJediWizard/keldris/internal/backup/backends"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog"
)

// CloudRestoreTargetType represents the type of cloud storage target.
type CloudRestoreTargetType string

const (
	// CloudRestoreTargetS3 represents an S3-compatible storage target.
	CloudRestoreTargetS3 CloudRestoreTargetType = "s3"
	// CloudRestoreTargetB2 represents a Backblaze B2 storage target.
	CloudRestoreTargetB2 CloudRestoreTargetType = "b2"
	// CloudRestoreTargetRestic represents another Restic repository as a target.
	CloudRestoreTargetRestic CloudRestoreTargetType = "restic"
)

// CloudRestoreTarget represents the target cloud storage for a restore operation.
type CloudRestoreTarget struct {
	Type CloudRestoreTargetType `json:"type"`
	// S3/B2 configuration
	Bucket          string `json:"bucket,omitempty"`
	Prefix          string `json:"prefix,omitempty"`
	Region          string `json:"region,omitempty"`
	Endpoint        string `json:"endpoint,omitempty"`
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
	UseSSL          bool   `json:"use_ssl,omitempty"`
	// B2 specific
	AccountID      string `json:"account_id,omitempty"`
	ApplicationKey string `json:"application_key,omitempty"`
	// Restic repository configuration
	Repository         string `json:"repository,omitempty"`
	RepositoryPassword string `json:"repository_password,omitempty"`
}

// CloudRestoreProgress represents the progress of a cloud restore upload operation.
type CloudRestoreProgress struct {
	mu               sync.RWMutex
	TotalFiles       int64     `json:"total_files"`
	TotalBytes       int64     `json:"total_bytes"`
	UploadedFiles    int64     `json:"uploaded_files"`
	UploadedBytes    int64     `json:"uploaded_bytes"`
	CurrentFile      string    `json:"current_file"`
	Status           string    `json:"status"` // "restoring", "uploading", "verifying", "completed", "failed"
	StartedAt        time.Time `json:"started_at"`
	ErrorMessage     string    `json:"error_message,omitempty"`
	VerifiedChecksum bool      `json:"verified_checksum"`
}

// Update updates the progress with thread safety.
func (p *CloudRestoreProgress) Update(fn func(*CloudRestoreProgress)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	fn(p)
}

// Get returns a copy of the progress with thread safety.
func (p *CloudRestoreProgress) Get() CloudRestoreProgress {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return CloudRestoreProgress{
		TotalFiles:       p.TotalFiles,
		TotalBytes:       p.TotalBytes,
		UploadedFiles:    p.UploadedFiles,
		UploadedBytes:    p.UploadedBytes,
		CurrentFile:      p.CurrentFile,
		Status:           p.Status,
		StartedAt:        p.StartedAt,
		ErrorMessage:     p.ErrorMessage,
		VerifiedChecksum: p.VerifiedChecksum,
	}
}

// PercentComplete returns the upload completion percentage.
func (p *CloudRestoreProgress) PercentComplete() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.TotalBytes == 0 {
		return 0
	}
	return float64(p.UploadedBytes) / float64(p.TotalBytes) * 100
}

// CloudRestoreOptions configures a cloud restore operation.
type CloudRestoreOptions struct {
	SnapshotID   string              `json:"snapshot_id"`
	Include      []string            `json:"include,omitempty"`
	Exclude      []string            `json:"exclude,omitempty"`
	Target       CloudRestoreTarget  `json:"target"`
	TempDir      string              `json:"temp_dir,omitempty"` // Defaults to os.TempDir()
	VerifyUpload bool                `json:"verify_upload"`      // Verify upload integrity with checksums
	Concurrency  int                 `json:"concurrency"`        // Number of concurrent uploads (default: 5)
	ProgressChan chan<- CloudRestoreProgress `json:"-"` // Optional channel to receive progress updates
}

// CloudRestoreResult contains the result of a cloud restore operation.
type CloudRestoreResult struct {
	UploadedFiles  int64         `json:"uploaded_files"`
	UploadedBytes  int64         `json:"uploaded_bytes"`
	Duration       time.Duration `json:"duration"`
	TargetLocation string        `json:"target_location"`
	Verified       bool          `json:"verified"`
}

// CloudRestore wraps cloud restore operations.
type CloudRestore struct {
	restic *Restic
	logger zerolog.Logger
}

// NewCloudRestore creates a new CloudRestore instance.
func NewCloudRestore(restic *Restic, logger zerolog.Logger) *CloudRestore {
	return &CloudRestore{
		restic: restic,
		logger: logger.With().Str("component", "cloud_restore").Logger(),
	}
}

// RestoreToCloud restores a snapshot and uploads it to cloud storage.
func (cr *CloudRestore) RestoreToCloud(ctx context.Context, cfg backends.ResticConfig, opts CloudRestoreOptions) (*CloudRestoreResult, error) {
	start := time.Now()
	progress := &CloudRestoreProgress{
		Status:    "restoring",
		StartedAt: start,
	}

	// Set defaults
	if opts.TempDir == "" {
		opts.TempDir = os.TempDir()
	}
	if opts.Concurrency <= 0 {
		opts.Concurrency = 5
	}

	// Create a unique temp directory for this restore
	tempRestoreDir, err := os.MkdirTemp(opts.TempDir, "cloud-restore-*")
	if err != nil {
		return nil, fmt.Errorf("create temp directory: %w", err)
	}
	defer func() {
		cr.logger.Info().Str("temp_dir", tempRestoreDir).Msg("cleaning up temp directory")
		if removeErr := os.RemoveAll(tempRestoreDir); removeErr != nil {
			cr.logger.Warn().Err(removeErr).Str("temp_dir", tempRestoreDir).Msg("failed to clean up temp directory")
		}
	}()

	cr.logger.Info().
		Str("snapshot_id", opts.SnapshotID).
		Str("target_type", string(opts.Target.Type)).
		Str("temp_dir", tempRestoreDir).
		Msg("starting cloud restore")

	// Send progress update
	cr.sendProgress(opts.ProgressChan, progress)

	// Step 1: Restore to temp location
	restoreOpts := RestoreOptions{
		TargetPath: tempRestoreDir,
		Include:    opts.Include,
		Exclude:    opts.Exclude,
	}

	if err := cr.restic.Restore(ctx, cfg, opts.SnapshotID, restoreOpts); err != nil {
		progress.Update(func(p *CloudRestoreProgress) {
			p.Status = "failed"
			p.ErrorMessage = fmt.Sprintf("restore failed: %v", err)
		})
		cr.sendProgress(opts.ProgressChan, progress)
		return nil, fmt.Errorf("restore to temp: %w", err)
	}

	cr.logger.Info().Str("temp_dir", tempRestoreDir).Msg("restore to temp completed")

	// Step 2: Calculate total files and size
	var totalFiles, totalBytes int64
	err = filepath.WalkDir(tempRestoreDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			totalFiles++
			if info, infoErr := d.Info(); infoErr == nil {
				totalBytes += info.Size()
			}
		}
		return nil
	})
	if err != nil {
		progress.Update(func(p *CloudRestoreProgress) {
			p.Status = "failed"
			p.ErrorMessage = fmt.Sprintf("scan files failed: %v", err)
		})
		cr.sendProgress(opts.ProgressChan, progress)
		return nil, fmt.Errorf("scan restored files: %w", err)
	}

	progress.Update(func(p *CloudRestoreProgress) {
		p.Status = "uploading"
		p.TotalFiles = totalFiles
		p.TotalBytes = totalBytes
	})
	cr.sendProgress(opts.ProgressChan, progress)

	cr.logger.Info().
		Int64("total_files", totalFiles).
		Int64("total_bytes", totalBytes).
		Msg("starting upload to cloud")

	// Step 3: Upload to cloud target
	var result *CloudRestoreResult
	switch opts.Target.Type {
	case CloudRestoreTargetS3:
		result, err = cr.uploadToS3(ctx, tempRestoreDir, opts, progress)
	case CloudRestoreTargetB2:
		result, err = cr.uploadToB2(ctx, tempRestoreDir, opts, progress)
	case CloudRestoreTargetRestic:
		result, err = cr.uploadToRestic(ctx, tempRestoreDir, opts, progress, cfg)
	default:
		err = fmt.Errorf("unsupported cloud target type: %s", opts.Target.Type)
	}

	if err != nil {
		progress.Update(func(p *CloudRestoreProgress) {
			p.Status = "failed"
			p.ErrorMessage = fmt.Sprintf("upload failed: %v", err)
		})
		cr.sendProgress(opts.ProgressChan, progress)
		return nil, err
	}

	// Step 4: Verify upload if requested
	if opts.VerifyUpload {
		progress.Update(func(p *CloudRestoreProgress) {
			p.Status = "verifying"
		})
		cr.sendProgress(opts.ProgressChan, progress)

		verified, verifyErr := cr.verifyUpload(ctx, opts, result)
		if verifyErr != nil {
			cr.logger.Warn().Err(verifyErr).Msg("upload verification failed")
			result.Verified = false
		} else {
			result.Verified = verified
		}

		progress.Update(func(p *CloudRestoreProgress) {
			p.VerifiedChecksum = result.Verified
		})
	}

	result.Duration = time.Since(start)

	progress.Update(func(p *CloudRestoreProgress) {
		p.Status = "completed"
	})
	cr.sendProgress(opts.ProgressChan, progress)

	cr.logger.Info().
		Int64("uploaded_files", result.UploadedFiles).
		Int64("uploaded_bytes", result.UploadedBytes).
		Dur("duration", result.Duration).
		Bool("verified", result.Verified).
		Msg("cloud restore completed")

	return result, nil
}

// uploadToS3 uploads restored files to S3-compatible storage.
func (cr *CloudRestore) uploadToS3(ctx context.Context, sourceDir string, opts CloudRestoreOptions, progress *CloudRestoreProgress) (*CloudRestoreResult, error) {
	target := opts.Target

	// Validate S3 configuration
	if target.Bucket == "" {
		return nil, fmt.Errorf("s3 bucket is required")
	}
	if target.AccessKeyID == "" || target.SecretAccessKey == "" {
		return nil, fmt.Errorf("s3 credentials are required")
	}

	// Set up AWS config
	region := target.Region
	if region == "" {
		region = "us-east-1"
	}

	awsOpts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			target.AccessKeyID,
			target.SecretAccessKey,
			"",
		)),
	}

	cfg, err := config.LoadDefaultConfig(ctx, awsOpts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	// Create S3 client
	clientOpts := []func(*s3.Options){}
	if target.Endpoint != "" {
		scheme := "http"
		if target.UseSSL {
			scheme = "https"
		}
		endpoint := target.Endpoint
		// Remove scheme if present
		endpoint = strings.TrimPrefix(endpoint, "http://")
		endpoint = strings.TrimPrefix(endpoint, "https://")
		endpointURL := fmt.Sprintf("%s://%s", scheme, endpoint)
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpointURL)
			o.UsePathStyle = true
		})
	}

	client := s3.NewFromConfig(cfg, clientOpts...)
	uploader := manager.NewUploader(client, func(u *manager.Uploader) {
		u.Concurrency = opts.Concurrency
	})

	var uploadedFiles, uploadedBytes int64
	targetLocation := fmt.Sprintf("s3://%s/%s", target.Bucket, target.Prefix)

	err = filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		// Calculate the relative path for the S3 key
		relPath, relErr := filepath.Rel(sourceDir, path)
		if relErr != nil {
			return fmt.Errorf("get relative path: %w", relErr)
		}

		// Build S3 key with prefix
		key := relPath
		if target.Prefix != "" {
			key = strings.TrimSuffix(target.Prefix, "/") + "/" + relPath
		}

		progress.Update(func(p *CloudRestoreProgress) {
			p.CurrentFile = relPath
		})
		cr.sendProgress(opts.ProgressChan, progress)

		// Open file for upload
		file, openErr := os.Open(path)
		if openErr != nil {
			return fmt.Errorf("open file %s: %w", path, openErr)
		}
		defer file.Close()

		info, _ := file.Stat()
		fileSize := info.Size()

		// Upload to S3
		_, uploadErr := uploader.Upload(ctx, &s3.PutObjectInput{
			Bucket: aws.String(target.Bucket),
			Key:    aws.String(key),
			Body:   file,
		})
		if uploadErr != nil {
			return fmt.Errorf("upload %s: %w", key, uploadErr)
		}

		uploadedFiles++
		uploadedBytes += fileSize

		progress.Update(func(p *CloudRestoreProgress) {
			p.UploadedFiles = uploadedFiles
			p.UploadedBytes = uploadedBytes
		})
		cr.sendProgress(opts.ProgressChan, progress)

		cr.logger.Debug().
			Str("key", key).
			Int64("size", fileSize).
			Msg("uploaded file to S3")

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &CloudRestoreResult{
		UploadedFiles:  uploadedFiles,
		UploadedBytes:  uploadedBytes,
		TargetLocation: targetLocation,
	}, nil
}

// uploadToB2 uploads restored files to Backblaze B2 storage.
// B2 is S3-compatible, so we use the S3 upload mechanism with B2's S3 endpoint.
func (cr *CloudRestore) uploadToB2(ctx context.Context, sourceDir string, opts CloudRestoreOptions, progress *CloudRestoreProgress) (*CloudRestoreResult, error) {
	target := opts.Target

	// Validate B2 configuration
	if target.Bucket == "" {
		return nil, fmt.Errorf("b2 bucket is required")
	}
	if target.AccountID == "" || target.ApplicationKey == "" {
		return nil, fmt.Errorf("b2 credentials are required")
	}

	// B2 S3-compatible endpoint format
	// Region can be us-west-001, us-west-002, eu-central-003, etc.
	region := target.Region
	if region == "" {
		region = "us-west-002"
	}
	endpoint := fmt.Sprintf("s3.%s.backblazeb2.com", region)

	// Create S3-compatible upload using B2's S3 interface
	s3Target := CloudRestoreTarget{
		Type:            CloudRestoreTargetS3,
		Bucket:          target.Bucket,
		Prefix:          target.Prefix,
		Region:          region,
		Endpoint:        endpoint,
		AccessKeyID:     target.AccountID,
		SecretAccessKey: target.ApplicationKey,
		UseSSL:          true,
	}

	s3Opts := CloudRestoreOptions{
		SnapshotID:   opts.SnapshotID,
		Include:      opts.Include,
		Exclude:      opts.Exclude,
		Target:       s3Target,
		TempDir:      opts.TempDir,
		VerifyUpload: opts.VerifyUpload,
		Concurrency:  opts.Concurrency,
		ProgressChan: opts.ProgressChan,
	}

	result, err := cr.uploadToS3(ctx, sourceDir, s3Opts, progress)
	if err != nil {
		return nil, err
	}

	// Update the target location to reflect B2
	result.TargetLocation = fmt.Sprintf("b2://%s/%s", target.Bucket, target.Prefix)

	return result, nil
}

// uploadToRestic backs up the restored files to another Restic repository.
func (cr *CloudRestore) uploadToRestic(ctx context.Context, sourceDir string, opts CloudRestoreOptions, progress *CloudRestoreProgress, sourceCfg backends.ResticConfig) (*CloudRestoreResult, error) {
	target := opts.Target

	// Validate Restic target configuration
	if target.Repository == "" {
		return nil, fmt.Errorf("target restic repository is required")
	}
	if target.RepositoryPassword == "" {
		return nil, fmt.Errorf("target restic repository password is required")
	}

	targetCfg := backends.ResticConfig{
		Repository: target.Repository,
		Password:   target.RepositoryPassword,
		Env:        make(map[string]string),
	}

	// Initialize target repository if needed
	if err := cr.restic.Init(ctx, targetCfg); err != nil {
		cr.logger.Debug().Err(err).Msg("target repository init (may already exist)")
	}

	// Count files for progress
	var totalFiles, totalBytes int64
	err := filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			totalFiles++
			if info, infoErr := d.Info(); infoErr == nil {
				totalBytes += info.Size()
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan files: %w", err)
	}

	progress.Update(func(p *CloudRestoreProgress) {
		p.TotalFiles = totalFiles
		p.TotalBytes = totalBytes
		p.CurrentFile = "backing up to target repository"
	})
	cr.sendProgress(opts.ProgressChan, progress)

	// Backup the restored files to the target repository
	backupStats, err := cr.restic.Backup(ctx, targetCfg, []string{sourceDir}, nil, []string{"cloud-restore"})
	if err != nil {
		return nil, fmt.Errorf("backup to target repository: %w", err)
	}

	progress.Update(func(p *CloudRestoreProgress) {
		p.UploadedFiles = totalFiles
		p.UploadedBytes = totalBytes
	})
	cr.sendProgress(opts.ProgressChan, progress)

	return &CloudRestoreResult{
		UploadedFiles:  totalFiles,
		UploadedBytes:  totalBytes,
		TargetLocation: target.Repository,
		Verified:       backupStats.SnapshotID != "",
	}, nil
}

// verifyUpload verifies the integrity of uploaded files.
func (cr *CloudRestore) verifyUpload(ctx context.Context, opts CloudRestoreOptions, result *CloudRestoreResult) (bool, error) {
	cr.logger.Info().
		Str("target_type", string(opts.Target.Type)).
		Str("target_location", result.TargetLocation).
		Msg("verifying upload integrity")

	switch opts.Target.Type {
	case CloudRestoreTargetS3:
		return cr.verifyS3Upload(ctx, opts, result)
	case CloudRestoreTargetB2:
		return cr.verifyB2Upload(ctx, opts, result)
	case CloudRestoreTargetRestic:
		// Restic verifies automatically during backup
		return true, nil
	default:
		return false, fmt.Errorf("verification not supported for target type: %s", opts.Target.Type)
	}
}

// verifyS3Upload verifies that files were correctly uploaded to S3.
func (cr *CloudRestore) verifyS3Upload(ctx context.Context, opts CloudRestoreOptions, result *CloudRestoreResult) (bool, error) {
	target := opts.Target

	region := target.Region
	if region == "" {
		region = "us-east-1"
	}

	awsOpts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			target.AccessKeyID,
			target.SecretAccessKey,
			"",
		)),
	}

	cfg, err := config.LoadDefaultConfig(ctx, awsOpts...)
	if err != nil {
		return false, fmt.Errorf("load aws config: %w", err)
	}

	clientOpts := []func(*s3.Options){}
	if target.Endpoint != "" {
		scheme := "http"
		if target.UseSSL {
			scheme = "https"
		}
		endpoint := target.Endpoint
		endpoint = strings.TrimPrefix(endpoint, "http://")
		endpoint = strings.TrimPrefix(endpoint, "https://")
		endpointURL := fmt.Sprintf("%s://%s", scheme, endpoint)
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpointURL)
			o.UsePathStyle = true
		})
	}

	client := s3.NewFromConfig(cfg, clientOpts...)

	// List objects with the prefix and count them
	prefix := target.Prefix
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	var objectCount int64
	var totalSize int64
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(target.Bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, pageErr := paginator.NextPage(ctx)
		if pageErr != nil {
			return false, fmt.Errorf("list objects: %w", pageErr)
		}
		for _, obj := range page.Contents {
			objectCount++
			if obj.Size != nil {
				totalSize += *obj.Size
			}
		}
	}

	// Verify counts match
	if objectCount != result.UploadedFiles {
		cr.logger.Warn().
			Int64("expected_files", result.UploadedFiles).
			Int64("found_files", objectCount).
			Msg("file count mismatch")
		return false, nil
	}

	if totalSize != result.UploadedBytes {
		cr.logger.Warn().
			Int64("expected_bytes", result.UploadedBytes).
			Int64("found_bytes", totalSize).
			Msg("size mismatch")
		return false, nil
	}

	cr.logger.Info().
		Int64("verified_files", objectCount).
		Int64("verified_bytes", totalSize).
		Msg("upload verification passed")

	return true, nil
}

// verifyB2Upload verifies that files were correctly uploaded to B2.
func (cr *CloudRestore) verifyB2Upload(ctx context.Context, opts CloudRestoreOptions, result *CloudRestoreResult) (bool, error) {
	// B2 verification uses S3-compatible API
	target := opts.Target
	region := target.Region
	if region == "" {
		region = "us-west-002"
	}

	s3Target := CloudRestoreTarget{
		Type:            CloudRestoreTargetS3,
		Bucket:          target.Bucket,
		Prefix:          target.Prefix,
		Region:          region,
		Endpoint:        fmt.Sprintf("s3.%s.backblazeb2.com", region),
		AccessKeyID:     target.AccountID,
		SecretAccessKey: target.ApplicationKey,
		UseSSL:          true,
	}

	s3Opts := CloudRestoreOptions{
		Target: s3Target,
	}

	return cr.verifyS3Upload(ctx, s3Opts, result)
}

// sendProgress sends a progress update if a channel is configured.
func (cr *CloudRestore) sendProgress(ch chan<- CloudRestoreProgress, progress *CloudRestoreProgress) {
	if ch == nil {
		return
	}
	select {
	case ch <- progress.Get():
	default:
		// Don't block if channel is full
	}
}

// ProgressReader wraps an io.Reader to track read progress.
type ProgressReader struct {
	reader    io.Reader
	bytesRead int64
	onRead    func(bytesRead int64)
}

// NewProgressReader creates a new ProgressReader.
func NewProgressReader(reader io.Reader, onRead func(bytesRead int64)) *ProgressReader {
	return &ProgressReader{
		reader: reader,
		onRead: onRead,
	}
}

// Read implements io.Reader.
func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.bytesRead += int64(n)
	if pr.onRead != nil {
		pr.onRead(pr.bytesRead)
	}
	return n, err
}
