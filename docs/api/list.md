# 分页查询图库

分页返回图库表 `gallery` 的记录，按 id 倒序（最新上传在前）。

## 请求

```
GET /api/v1/gallery?page=1&pageSize=10
```

### Query 参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `page` | int | 是 | 页码，从 1 开始 |
| `pageSize` | int | 是 | 每页条数，正整数 |

无其他入参。

## 响应

### 成功（200）

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 12,
        "source": "minio-dev",
        "url": "http://minio-dev.example.com/gin-quickstart-dev/img/banner.png",
        "objectKey": "img/banner.png",
        "name": "banner.png",
        "createdAt": "2026-09-06T10:30:00+08:00"
      }
    ],
    "total": 42,
    "page": 1,
    "pageSize": 10,
    "totalPages": 5
  }
}
```

| 字段 | 说明 |
| --- | --- |
| `list` | 当前页记录列表，可能为空数组 |
| `list[].id` | 记录主键，删除、校验接口使用该值 |
| `list[].source` | 存储源名称，对应配置中 `storage.sources` 的 key |
| `list[].url` | 文件访问地址 |
| `list[].objectKey` | 文件在桶中的完整路径 |
| `list[].name` | 原始文件名 |
| `list[].createdAt` | 上传时间（RFC3339） |
| `total` | 记录总条数 |
| `page` | 当前页码 |
| `pageSize` | 当前页大小 |
| `totalPages` | 总页数 |

### 错误

| 状态码 | code | 场景 |
| --- | --- | --- |
| 400 | 40000 | `page` 或 `pageSize` 缺失、非数字或小于 1 |

## 调用示例

```bash
curl "http://localhost:8080/api/v1/gallery?page=1&pageSize=10"
```
