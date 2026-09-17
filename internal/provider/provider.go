package provider

import (
	"context"
	"fmt"
	"io"

	"github.com/wilson-lyc/dextea-v3-xos/internal/config"
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
	Bucket    string
	ObjectKey string
	Size      int64
	ETag      string
}

// ObjectProvider 对象存储统一抽象，后续若需支持非 S3 协议厂商，新增实现即可。
type ObjectProvider interface {
	// Upload 上传对象到指定 bucket（bucket 为空时使用该源的默认 bucket）。
	Upload(ctx context.Context, bucket string, in UploadInput) (*UploadResult, error)
	// Delete 删除指定 bucket 中的对象（bucket 为空时使用该源的默认 bucket）。
	Delete(ctx context.Context, bucket, objectKey string) error
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

// Manager 持有所有存储源的 provider 与配置，供上传、图库删除等业务共用。
type Manager struct {
	providers map[string]ObjectProvider // key 为存储源名称
	specs     map[string]config.SourceSpec
}

// NewManager 按配置初始化所有存储源的 provider。
// 协议固定为 s3；后续接入非 S3 协议厂商时，可依据 spec.Vendor/新增协议字段路由到不同 provider。
func NewManager(sources map[string]config.SourceSpec) (*Manager, error) {
	providers := make(map[string]ObjectProvider, len(sources))
	for name, spec := range sources {
		// 当前所有源统一走 S3 协议；扩展点：按 spec 声明的协议选择 provider
		p, err := New("s3", spec)
		if err != nil {
			return nil, fmt.Errorf("init storage source %q: %w", name, err)
		}
		providers[name] = p
	}
	return &Manager{providers: providers, specs: sources}, nil
}

// Get 返回指定存储源的 provider 与配置。
func (m *Manager) Get(source string) (ObjectProvider, config.SourceSpec, bool) {
	p, ok := m.providers[source]
	return p, m.specs[source], ok
}

type UnsupportedProtocolError string

func (e UnsupportedProtocolError) Error() string {
	return "unsupported storage protocol: " + string(e)
}

func errUnsupportedProtocol(p string) error { return UnsupportedProtocolError(p) }
