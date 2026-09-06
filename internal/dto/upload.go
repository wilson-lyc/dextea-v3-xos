package dto

// UploadResp 上传接口响应体。
type UploadResp struct {
	Bucket    string `json:"bucket"`
	ObjectKey string `json:"object_key"`
	Size      int64  `json:"size"`
	ETag      string `json:"etag"`
}
