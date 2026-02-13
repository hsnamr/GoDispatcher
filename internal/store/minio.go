package store

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/halamri/go-dispatcher/internal/config"
)

// MinIO wraps S3-compatible MinIO client for object storage.
type MinIO struct {
	Client *minio.Client
	cfg    *config.Config
}

// NewMinIO creates a MinIO client from config.
func NewMinIO(cfg *config.Config) (*MinIO, error) {
	if cfg.MinIOEndpoint == "" || cfg.MinIOAccessKey == "" || cfg.MinIOSecretKey == "" {
		return nil, fmt.Errorf("minio config incomplete")
	}
	client, err := minio.New(cfg.MinIOEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIOAccessKey, cfg.MinIOSecretKey, ""),
		Secure: cfg.MinIOUseSSL,
		Region: cfg.MinIORegion,
	})
	if err != nil {
		return nil, err
	}
	return &MinIO{Client: client, cfg: cfg}, nil
}

// GetObject returns an object from the given bucket and key.
func (m *MinIO) GetObject(ctx context.Context, bucket, key string) (*minio.Object, error) {
	return m.Client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
}

// PutObject uploads an object (e.g. for exports).
func (m *MinIO) PutObject(ctx context.Context, bucket, key string, reader io.Reader, size int64, contentType string) (minio.UploadInfo, error) {
	return m.Client.PutObject(ctx, bucket, key, reader, size, minio.PutObjectOptions{ContentType: contentType})
}

// WhatsAppBucket returns the configured WhatsApp media bucket name.
func (m *MinIO) WhatsAppBucket() string {
	return m.cfg.MinIOWhatsAppBucket
}

// EngagementBucket returns the configured engagement bucket name.
func (m *MinIO) EngagementBucket() string {
	return m.cfg.MinIOEngagementBucket
}
