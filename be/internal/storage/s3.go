// Package storage — фото фиксаций в S3 (RustFS/MinIO/AWS; клиент minio-go).
package storage

import (
	"bytes"
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3 — бакет с фото.
type S3 struct {
	client *minio.Client
	bucket string
}

// NewS3: endpoint — "host:port" без схемы.
func NewS3(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*S3, error) {
	c, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
		Region: "us-east-1",
	})
	if err != nil {
		return nil, fmt.Errorf("s3: %w", err)
	}
	return &S3{client: c, bucket: bucket}, nil
}

// Put кладёт объект в бакет.
func (s *S3) Put(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("s3 put %s: %w", key, err)
	}
	return nil
}

// Ping — проверка готовности: бакет доступен (HeadBucket).
func (s *S3) Ping(ctx context.Context) error {
	ok, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("s3 head bucket %s: %w", s.bucket, err)
	}
	if !ok {
		return fmt.Errorf("s3: бакет %s не найден", s.bucket)
	}
	return nil
}
