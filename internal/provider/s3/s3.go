// Package s3 基于 S3 协议的对象存储 provider。
// 借助 S3 兼容性，同一实现可对接 MinIO、AWS S3、阿里云 OSS、腾讯云 COS、七牛 Kodo 等厂商。
package s3

import (
	"context"
	"errors"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/wilson-lyc/dextea-v3-xos/internal/config"
	"github.com/wilson-lyc/dextea-v3-xos/internal/provider"
)

func init() {
	provider.Register("s3", New)
}

type Client struct {
	mc            *minio.Client
	defaultBucket string
}

func New(spec config.SourceSpec) (provider.ObjectProvider, error) {
	if spec.Endpoint == "" {
		return nil, errors.New("s3 provider: endpoint is required")
	}
	mc, err := minio.New(spec.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(spec.AccessKey, spec.SecretKey, ""),
		Secure: spec.UseSSL,
		Region: spec.Region,
	})
	if err != nil {
		return nil, err
	}
	return &Client{mc: mc, defaultBucket: spec.DefaultBucket}, nil
}

func (c *Client) Upload(ctx context.Context, bucket string, in provider.UploadInput) (*provider.UploadResult, error) {
	if bucket == "" {
		bucket = c.defaultBucket
	}
	if bucket == "" {
		return nil, errors.New("s3 provider: bucket is required")
	}

	contentType := in.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	info, err := c.mc.PutObject(ctx, bucket, in.ObjectKey, in.Reader, in.Size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return nil, err
	}
	return &provider.UploadResult{
		Bucket:    bucket,
		ObjectKey: in.ObjectKey,
		Size:      info.Size,
		ETag:      info.ETag,
	}, nil
}

// Delete 删除指定 bucket 中的对象，bucket 为空时使用默认 bucket。
func (c *Client) Delete(ctx context.Context, bucket, objectKey string) error {
	if bucket == "" {
		bucket = c.defaultBucket
	}
	if bucket == "" {
		return errors.New("s3 provider: bucket is required")
	}
	return c.mc.RemoveObject(ctx, bucket, objectKey, minio.RemoveObjectOptions{})
}

// EnsureBucket 存在性检查 + 建桶，供服务启动或按需调用。
func (c *Client) EnsureBucket(ctx context.Context, bucket string) error {
	exists, err := c.mc.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return c.mc.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
}
