package dto

// UploadResp 上传接口响应体。
type UploadResp struct {
	GalleryID int64  `json:"galleryId"`
	Bucket    string `json:"bucket"`
	ObjectKey string `json:"objectKey"`
	Size      int64  `json:"size"`
	ETag      string `json:"etag"`
	URL       string `json:"url"`
	Name      string `json:"name"`
}
