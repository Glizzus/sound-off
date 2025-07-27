package config

import (
	"context"

	"github.com/sethvargo/go-envconfig"
)

type MinioConfig struct {
	Endpoint string `env:"MINIO_ENDPOINT, required"`
	Username string `env:"MINIO_USERNAME, required"`
	Password string `env:"MINIO_PASSWORD, required"`
	Bucket   string `env:"MINIO_BUCKET, default=soundoff"`
}

func NewMinioConfigFromEnv() (*MinioConfig, error) {
	var cfg MinioConfig
	if err := envconfig.Process(context.Background(), &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
