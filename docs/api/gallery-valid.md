# 校验图库 id 是否合法

校验某个 id 对应的图库记录是否存在。校验不通过属于正常业务结果（`valid: false`），不返回 404。

## 请求

```
GET /api/v1/gallery/{id}/valid
```

### 路径参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `id` | int64 | 是 | 待校验的图库记录主键 |

无其他入参。

## 响应

### 成功（200）

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

## 调用示例

```bash
curl http://localhost:8080/api/v1/gallery/12/valid
```
