package storage

import (
	"context"
	"io"

	"github.com/minio/minio-go/v7"
)

// MinioClient wraps a minio.Client with bucket-level helpers.
type MinioClient struct {
	Client     *minio.Client
	BucketName string
}

// NewMinioClient creates a new wrapper instance.
func NewMinioClient(client *minio.Client, bucketName string) *MinioClient {
	return &MinioClient{
		Client:     client,
		BucketName: bucketName,
	}
}

// UploadObject uploads a reader to the configured bucket.
func (m *MinioClient) UploadObject(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error {
	_, err := m.Client.PutObject(ctx, m.BucketName, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}
