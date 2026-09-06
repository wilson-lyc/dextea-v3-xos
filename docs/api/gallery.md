# 图库

图库表 `gallery` 的查询与维护接口。每条记录对应一次成功上传的文件（存储源、URL、存储路径、文件名等）。

## 目录

- [分页查询图库](#分页查询图库)
- [删除图库记录](#删除图库记录)
- [校验 id 是否合法](#校验-id-是否合法)

---

## 分页查询图库

分页返回图库记录，按 id 倒序（最新上传在前）。

### 请求

```
GET /api/v1/gallery?page=1&page_size=10
```

### Query 参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `page` | int | 是 | 页码，从 1 开始 |
| `page_size` | int | 是 | 每页条数，正整数。无其他入参 |

### 成功响应（200）

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "ID": 12,
        "Source": "minio-dev",
        "URL": "http://minio-dev.example.com/gin-quickstart-dev/img/banner.png",
        "ObjectKey": "img/banner.png",
        "Name": "banner.png",
        "CreatedAt": "2026-09-06T10:30:00+08:00"
      }
    ],
    "total": 42,
    "page": 1,
    "page_size": 10,
    "total_pages": 5
  }
}
```

| 字段 | 说明 |
| --- | --- |
| `list` | 当前页记录列表，可能为空数组 |
| `list[].ID` | 记录主键，删除、校验接口使用该值 |
| `list[].Source` | 存储源名称，对应配置中 `storage.sources` 的 key |
| `list[].URL` | 文件访问地址 |
| `list[].ObjectKey` | 文件在桶中的完整路径 |
| `list[].Name` | 原始文件名 |
| `list[].CreatedAt` | 上传时间（RFC3339） |
| `total` | 记录总条数 |
| `total_pages` | 总页数 |

### 错误

| 状态码 | code | 场景 |
| --- | --- | --- |
| 400 | 40000 | `page` 或 `page_size` 缺失、非数字或小于 1 |

---

## 删除图库记录

按 id 删除图库记录。**仅支持单删**，不提供批量删除。

### 请求

```
DELETE /api/v1/gallery/{id}
```

### 路径参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `id` | int64 | 是 | 图库记录主键，可从分页查询接口获取 |

### 成功响应（200）

```json
{
  "code": 0,
  "message": "success"
}
```

### 错误

| 状态码 | code | 场景 |
| --- | --- | --- |
| 400 | 40000 | `id` 非数字或小于 1 |
| 404 | 40400 | id 对应的记录不存在 |

---

## 校验 id 是否合法

校验某个 id 对应的图库记录是否存在。校验不通过属于正常业务结果（`valid: false`），不返回 404。

### 请求

```
GET /api/v1/gallery/{id}/valid
```

### 路径参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `id` | int64 | 是 | 待校验的图库记录主键，无其他入参 |

### 成功响应（200）

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 12,
    "valid": true
  }
}
```

| 字段 | 说明 |
| --- | --- |
| `id` | 回显被校验的 id |
| `valid` | `true` 表示记录存在；`false` 表示不存在 |

### 错误

| 状态码 | code | 场景 |
| --- | --- | --- |
| 400 | 40000 | `id` 非数字或小于 1 |

---

## 调用示例

### cURL

```bash
# 分页查询第 1 页、每页 10 条
curl "http://localhost:8080/api/v1/gallery?page=1&page_size=10"

# 删除 id 为 12 的记录
curl -X DELETE http://localhost:8080/api/v1/gallery/12

# 校验 id 为 12 的记录是否存在
curl http://localhost:8080/api/v1/gallery/12/valid
```
