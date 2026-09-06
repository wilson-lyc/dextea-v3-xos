package service

import (
	"fmt"

	"gin-quickstart/internal/ecode"
)

// 业务错误统一使用 ecode 中定义的 *BizError，handler 据此映射
// HTTP 状态码与业务错误码。包装底层错误时用 WithCause 保留原因。
func ErrUnknownSource(source string) error {
	return ecode.SourceNotFound.WithCause(
		fmt.Errorf("unknown storage source %q (check configs/config.yaml)", source))
}

func ErrUploadFailed(source string, err error) error {
	return ecode.UploadFailed.WithCause(fmt.Errorf("upload to source %q: %w", source, err))
}
