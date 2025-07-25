package datalayer

import (
	"context"
	"io"

	"github.com/glizzus/sound-off/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type PutOptions struct {
	Size        int64
	ContentType string
}

type BlobStorage interface {
	Put(ctx context.Context, key string, data io.Reader, opts PutOptions) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
}

type MinioStorage struct {
	client *minio.Client
	bucket string
}

func NewMinioStorageFromEnv() (*MinioStorage, error) {
	cfg, err := config.NewMinioConfigFromEnv()
	if err != nil {
		return nil, err
	}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.Username, cfg.Password, ""),
		Secure: false,
	})
	if err != nil {
		return nil, err
	}

	return &MinioStorage{
		client: client,
		bucket: cfg.Bucket,
	}, nil
}

func (s *MinioStorage) EnsureBucket(ctx context.Context) error {
	err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{})
	// If the bucket is already owned, succeed
	if err != nil {
		if minio.ToErrorResponse(err).Code != "BucketAlreadyOwnedByYou" {
			return err
		}
	}
	policyJSON := `
	{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": "*",
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::` + s.bucket + `/*"]

			}
		]
	}
	`

	err = s.client.SetBucketPolicy(ctx, s.bucket, policyJSON)
	if err != nil {
		return err
	}

	return nil
}

var _ BlobStorage = (*MinioStorage)(nil)

func (s *MinioStorage) Put(ctx context.Context, key string, data io.Reader, opts PutOptions) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, data, opts.Size, minio.PutObjectOptions{
		ContentType: opts.ContentType,
	})
	return err
}

func (s *MinioStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return obj, nil
}
