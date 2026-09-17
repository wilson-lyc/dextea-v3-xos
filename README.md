# Dextea XOS

图片存储与图库 gRPC 服务，默认监听 `0.0.0.0:9091`。协议位于共享仓库 `dextea-proto/proto/xos/v1/xos.proto`，完整服务名为 `xos.v1.XOSService`。

## 本地运行

当前新协议尚未发布，`go.mod` 使用相邻目录 `../dextea-proto`，因此构建时两个仓库必须同时存在；独立 CI 也需检出共享协议到该位置。协议发布后可改为正式模块版本并移除 replace。

复制 `configs/config.example.yaml` 为 `configs/config.yaml`，配置 MySQL、Redis 和 S3 存储源；可复制 `.env.example` 为 `.env` 配置环境变量。启动前需准备已有的 gallery 数据表。

```sh
go run ./cmd/server
go test ./...
```

## RPC

- `Upload`：source、file_name 和 content 必填，bucket、object_key 可选；content 为原始图片字节。默认最大 20 MiB，服务继续按文件头验证图片类型，不信任文件扩展名。
- `ListPage`：page、page_size 均须大于零，返回图库列表与分页信息。
- `Delete`：正整数 id，删除对象及图库记录。
- `ValidateID`：正整数 id，返回是否存在。
- `GetURLs`：1–100 个正整数 ids，返回 id 到图片地址的映射；忽略不存在的记录。
- 健康检查使用标准 `grpc.health.v1.Health`，支持空服务名和 `xos.v1.XOSService`。

上传使用 unary RPC。接收消息上限为文件上限加 64 KiB 协议开销，文件上限另行精确校验；调用端如设置了发送限制，也需按上传大小调整。RPC 错误采用标准 gRPC status，业务错误码保存在 `google.rpc.ErrorInfo` 的 reason 字段中。

链路追踪使用 OTLP/gRPC，默认端口 4317；生产环境按部署配置 TLS。进程收到 SIGINT/SIGTERM 后停止健康状态并优雅关闭，最多等待 5 秒。

原 Gin 服务、REST 路由、multipart 上传、HTTP 响应封装和 HTTP 链路导出已移除。S3 SDK 的底层通信和返回给图片消费者的访问 URL 属于对象存储协议，继续保留。
