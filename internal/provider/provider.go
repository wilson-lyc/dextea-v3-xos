package provider

import (
	"context"
	"io"

	"gin-quickstart/internal/config"
)

// UploadInput 上传请求参数。
type UploadInput struct {
	FileName    string // 原始文件名，未指定 objectKey 时用于生成存储路径
	ObjectKey   string // 对象完整路径，如 2026/09/06/xxx.png
	Reader      io.Reader
	Size        int64
	ContentType string
}

// UploadResult 上传结果。
type UploadResult struct {
	Bucket   string
	ObjectKey string
	Size     int64
	ETag     string
}

// ObjectProvider 对象存储统一抽象，后续若需支持非 S3 协议厂商，新增实现即可。
type ObjectProvider interface {
	// Upload 上传对象到指定 bucket（bucket 为空时使用该源的默认 bucket）。
	Upload(ctx context.Context, bucket string, in UploadInput) (*UploadResult, error)
}

// Factory 根据存储源配置构造 provider，按协议注册，支持多厂商扩展。
type Factory func(spec config.SourceSpec) (ObjectProvider, error)

var registry = map[string]Factory{}

// Register 注册协议工厂，key 为协议名（如 "s3"）。
func Register(protocol string, f Factory) {
	registry[protocol] = f
}

// New 根据存储源配置创建 provider。
func New(protocol string, spec config.SourceSpec) (ObjectProvider, error) {
	f, ok := registry[protocol]
	if !ok {
		return nil, errUnsupportedProtocol(protocol)
	}
	return f(spec)
}

type UnsupportedProtocolError string

func (e UnsupportedProtocolError) Error() string {
	return "unsupported storage protocol: " + string(e)
}

func errUnsupportedProtocol(p string) error { return UnsupportedProtocolError(p) }
