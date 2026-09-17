package ecode

import (
	"errors"
	"fmt"
	"google.golang.org/grpc/codes"
)

// BizError 业务错误，service 层抛出，handler/中间件统一转换成响应。
type BizError struct {
	RPCCode codes.Code // RPC 状态码
	Code    int        // 业务错误码
	Message string     // 对外错误信息
	cause   error      // 内部原因，用于日志与 errors.Is/As 链，不对外暴露
}

func (e *BizError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *BizError) Unwrap() error { return e.cause }

// WithCause 记录底层错误（如存储 SDK 返回的 error），仅用于日志排查。
func (e *BizError) WithCause(err error) *BizError {
	ne := *e
	ne.cause = err
	return &ne
}

// New 定义一个业务错误码。code 建议按模块分段规划（见文件底部错误码表）。
func New(rpcCode codes.Code, code int, message string) *BizError {
	return &BizError{RPCCode: rpcCode, Code: code, Message: message}
}

// From 将任意 error 转为 *BizError：已是 BizError 则原样返回，
// 否则归一化为 Internal（避免内部细节泄露给调用方）。
func From(err error) *BizError {
	var be *BizError
	if errors.As(err, &be) {
		return be
	}
	return Internal.WithCause(err)
}

// 通用错误码（0 表示成功）。
var (
	InvalidParam = New(codes.InvalidArgument, 40000, "invalid parameter")
	Unauthorized = New(codes.Unauthenticated, 40100, "unauthorized")
	Forbidden    = New(codes.PermissionDenied, 40300, "forbidden")
	NotFound     = New(codes.NotFound, 40400, "resource not found")
	Internal     = New(codes.Internal, 50000, "internal server error")
)

// 业务模块错误码：存储上传模块（4001x / 5001x）。
var (
	SourceNotFound = New(codes.NotFound, 40010, "storage source not found")
	UploadFailed   = New(codes.Internal, 50010, "upload to storage failed")
	FileTooLarge   = New(codes.ResourceExhausted, 40011, "file size exceeds limit")
	FileTypeDenied = New(codes.InvalidArgument, 40012, "only image files are allowed")
	DeleteFailed   = New(codes.Internal, 50011, "delete from storage failed")
)
