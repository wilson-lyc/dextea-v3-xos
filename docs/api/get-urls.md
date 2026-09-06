# 批量根据 id 查询 url

提供 id 数组，返回 id 对应的 url 映射。不存在的 id 不在结果中（不属于错误）。

## 请求

```
POST /api/v1/gallery/urls
Content-Type: application/json
```

### 请求体

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `ids` | int64 数组 | 是 | 待查询的图库记录主键，1 ~ 100 个，每个 id 需大于 0 |

```json
{
  "ids": [1, 2, 12]
}
```

## 响应

### 成功（200）

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "urls": {
      "1": "https://cdn.example.com/a.png",
      "12": "https://cdn.example.com/b.png"
    }
  }
}
```

| 字段 | 说明 |
| --- | --- |
| `urls` | `id -> url` 对象，仅包含实际存在的记录；全部不存在时为 `{}` |

### 错误

| 状态码 | code | 场景 |
| --- | --- | --- |
| 400 | 40000 | 请求体非法、`ids` 为空、超过 100 个，或存在小于 1 的 id |

## 调用示例

```bash
curl -X POST http://localhost:8080/api/v1/gallery/urls \
  -H 'Content-Type: application/json' \
  -d '{"ids": [1, 2, 12]}'
```
