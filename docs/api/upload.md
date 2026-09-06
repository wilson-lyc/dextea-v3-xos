# 图片上传

将图片上传到指定的对象存储源。仅允许上传图片文件（jpeg / png / gif / webp / bmp），单文件大小默认上限 20MB。

## 请求

```
POST /api/v1/storage/{source}/objects
Content-Type: multipart/form-data
```

### 路径参数

| 参数 | 位置 | 必填 | 说明 |
| --- | --- | --- | --- |
| `source` | path | 是 | 存储源名称。决定了文件存到哪个对象存储（如 `minio-dev`、`aliyun-oss-prod`）。可用存储源列表请向平台管理员获取。 |

### 表单字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `file` | file | 是 | 要上传的文件 |
| `bucket` | string | 否 | 目标桶。不传时使用该存储源配置的默认桶 |
| `objectKey` | string | 否 | 文件的存储路径（如 `img/2026/banner.png`）。不传时由系统自动生成，格式为 `日期/时间戳_原文件名`，例如 `2026/09/06/1788659034105780934_banner.png` |

## 响应

### 成功（200）

```json
{
  "bucket": "gin-quickstart-dev",
  "objectKey": "2026/09/06/1788659034105780934_go.mod",
  "size": 1568,
  "etag": "d41d8cd98f00b204e9800998ecf8427e"
}
```

| 字段 | 说明 |
| --- | --- |
| `bucket` | 文件实际落入的桶 |
| `objectKey` | 文件在桶中的完整路径。后续下载、删除接口均以 `bucket + objectKey` 定位文件，请妥善保存 |
| `size` | 文件大小（字节） |
| `etag` | 文件内容指纹（MD5），可用于校验上传完整性 |

### 错误

| 状态码 | code | 场景 |
| --- | --- | --- |
| 400 | 40000 | 未携带 `file` 字段 |
| 400 | 40011 | 文件超过大小上限（默认 20MB，由配置 `storage.max-upload-size` 控制） |
| 400 | 40012 | 文件类型不是允许的图片格式（支持 jpeg / png / gif / webp / bmp，基于文件内容检测，伪造扩展名无效） |
| 404 | 40010 | 存储源名称不存在 |
| 500 | 50010 | 存储服务连接失败 / 上传失败 |

## 调用示例

### cURL

```bash
curl -X POST http://localhost:8080/api/v1/storage/minio-dev/objects \
  -F "file=@/path/to/banner.png"
```

指定桶和存储路径：

```bash
curl -X POST http://localhost:8080/api/v1/storage/minio-dev/objects \
  -F "file=@/path/to/banner.png" \
  -F "bucket=my-bucket" \
  -F "objectKey=img/2026/banner.png"
```

## 注意事项

1. **`source` 必须是平台已配置的存储源名称**，拼写错误会返回 40400 并提示 storage source not found。
2. 不传 `objectKey` 时系统生成的路径包含纳秒级时间戳，**重复上传同名文件不会被覆盖**；如需覆盖旧文件，请显式传入固定的 `objectKey`。
3. `objectKey` 中可使用 `/` 分隔多级目录，但请勿以 `/` 开头，也不要包含 `..`。
4. 文件类型通过文件头内容检测，伪造扩展名或 Content-Type 无法绕过白名单。
5. 排查问题时请携带响应头 `X-Request-ID` 联系平台管理员。
