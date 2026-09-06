# 删除图库记录

按 id 删除图库表 `gallery` 的记录。**仅支持单删**，不提供批量删除。

## 请求

```
DELETE /api/v1/gallery/{id}
```

### 路径参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `id` | int64 | 是 | 图库记录主键，可从[分页查询图库](./list-gallery.md)接口获取 |

## 响应

### 成功（200）

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

## 调用示例

```bash
curl -X DELETE http://localhost:8080/api/v1/gallery/12
```
