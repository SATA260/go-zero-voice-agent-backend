package minioutil

import (
	"context"
	"fmt"
	"io"
	"time"

	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinioConfig describes the connection settings required to build a client instance.
type MinioConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

// MinioClient wraps minio.Client and exposes a narrow helper API for common actions.
type MinioClient struct {
	client *minio.Client
}

// NewMinioClient creates a Minio client based on the provided configuration.
func NewMinioClient(cfg MinioConfig) (*MinioClient, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("minio endpoint must not be empty")
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("minio credentials must not be empty")
	}

	cli, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	return &MinioClient{
		client: cli,
	}, nil
}

// EnsureBucket checks the existence of a bucket and creates it when needed.
func (m *MinioClient) EnsureBucket(ctx context.Context, bucketName string) error {
	exists, err := m.client.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("check bucket %s: %w", bucketName, err)
	}
	if exists {
		return nil
	}

	if err := m.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); err != nil {
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "BucketAlreadyOwnedByYou" || errResp.Code == "BucketAlreadyExists" {
			return nil
		}
		return fmt.Errorf("make bucket %s: %w", bucketName, err)
	}
	return nil
}

// Upload streams data to the bucket/object pair and returns the resulting upload info.
func (m *MinioClient) Upload(
	ctx context.Context,
	bucketName string,
	objectName string,
	reader io.Reader,
	size int64,
	contentType string,
	metadata map[string]string,
) (minio.UploadInfo, error) {
	if reader == nil {
		return minio.UploadInfo{}, fmt.Errorf("upload reader must not be nil")
	}

	opts := minio.PutObjectOptions{
		ContentType:  contentType,
		UserMetadata: metadata,
	}

	info, err := m.client.PutObject(ctx, bucketName, objectName, reader, size, opts)
	if err != nil {
		return minio.UploadInfo{}, fmt.Errorf("put object %s/%s: %w", bucketName, objectName, err)
	}
	return info, nil
}

// Download returns an object reader so callers can stream the contents.
func (m *MinioClient) Download(ctx context.Context, bucketName, objectName string) (*minio.Object, error) {
	obj, err := m.client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object %s/%s: %w", bucketName, objectName, err)
	}
	return obj, nil
}

// Remove deletes an object from the bucket.
func (m *MinioClient) Remove(ctx context.Context, bucketName, objectName string) error {
	err := m.client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("remove object %s/%s: %w", bucketName, objectName, err)
	}
	return nil
}

// PresignedGet generates a temporary GET URL for the specified object.
func (m *MinioClient) PresignedGet(
	ctx context.Context,
	bucketName string,
	objectName string,
	expires time.Duration,
) (string, error) {
	url, err := m.client.PresignedGetObject(ctx, bucketName, objectName, expires, nil)
	if err != nil {
		return "", fmt.Errorf("presign get %s/%s: %w", bucketName, objectName, err)
	}
	return url.String(), nil
}

// PresignedPut generates a temporary PUT URL for the specified object.
func (m *MinioClient) PresignedPut(
	ctx context.Context,
	bucketName string,
	objectName string,
	expires time.Duration,
) (string, error) {
	url, err := m.client.PresignedPutObject(ctx, bucketName, objectName, expires)
	if err != nil {
		return "", fmt.Errorf("presign put %s/%s: %w", bucketName, objectName, err)
	}
	return url.String(), nil
}

// Client exposes the wrapped *minio.Client for advanced scenarios.
func (m *MinioClient) Client() *minio.Client {
	return m.client
}
