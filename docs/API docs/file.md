# File API 文档

## 接口概述

File 控制器负责文件管理，提供文件上传、随机图片获取、图片转Base64等功能。

**接口类型**：业务接口

---

## 状态码规范

| 状态码 | 使用场景 | 开发注意事项 |
|--------|----------|--------------|
| 200 | 接口调用成功且数据返回正常 | 所有正常响应必须返回200 |
| 400 | 请求参数错误或文件校验失败 | 文件大小超限、类型不允许等 |
| 405 | 请求方法不允许 | 调用了未定义的method |
| 500 | 服务器异常 | 存储操作失败等 |

---

## 接口列表

### 1. 文件上传

**请求方式**：POST  
**请求路径**：`/api/file/upload`

**Content-Type**：`multipart/form-data`

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| file | file | 是 | 要上传的文件 |

**允许的文件类型**：

| 类别 | 扩展名 |
|------|--------|
| 图片 | .jpg, .jpeg, .png, .gif, .bmp, .webp, .svg |
| 文档 | .pdf, .doc, .docx, .xls, .xlsx, .ppt, .pptx, .txt |
| 压缩包 | .zip, .rar, .7z, .tar, .gz |
| 音视频 | .mp3, .mp4, .wav, .avi, .mov, .flv |

**限制说明**：
- 文件大小最大：10MB
- 文件名会进行安全处理，防止路径遍历攻击

**响应示例**：

```json
{
  "code": 200,
  "msg": "上传成功！",
  "data": {
    "path": "https://example.com/storage/images/xxx.jpg"
  }
}
```

### 2. 随机图片

**请求方式**：GET  
**请求路径**：`/api/file/rand`

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| name | string | 否 | 指定目录或文件名 |
| json | bool | 否 | 是否返回JSON格式 |
| redirect | bool | 否 | 是否重定向到图片 |
| size | string | 否 | 图片尺寸，格式如 `200x200` |
| mode | string | 否 | 图片处理模式（fill/resize/fit） |

**使用说明**：

1. **获取随机图片URL（JSON格式）**：
   ```
   GET /api/file/rand?json=true
   ```
   返回：`{"code":200,"msg":"随机图！","data":"https://example.com/rand/xxx.jpg"}`

2. **直接输出图片**：
   ```
   GET /api/file/rand
   ```
   返回：图片二进制流

3. **重定向到图片**：
   ```
   GET /api/file/rand?redirect=true
   ```
   返回：302重定向

4. **指定目录**：
   ```
   GET /api/file/rand?name=nature
   ```
   从 `nature` 目录中随机选择

5. **指定尺寸**：
   ```
   GET /api/file/rand?size=300x200&mode=fill
   ```
   返回300x200的图片

**处理模式说明**：

| 模式 | 说明 |
|------|------|
| fill | 填充模式，按指定尺寸裁剪 |
| resize | 强制缩放，可能变形 |
| fit | 等比例缩放（默认） |

### 3. 图片转Base64

**请求方式**：GET  
**请求路径**：`/api/file/to-base64`

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| url | string | 是 | 远程图片URL |

**响应示例**：

```json
{
  "code": 200,
  "msg": "成功！",
  "data": "data:image/jpeg;base64,/9j/4AAQSkZJRg..."
}
```

---

## 特殊说明

### 文件存储

- 文件按扩展名分类存储到不同目录
- 支持的存储驱动可配置

### 安全处理

- 文件名安全过滤（移除 `..`, `/`, `\`）
- 文件类型白名单校验
- 大小限制检查

### 图片处理

- 使用 `imaging` 库进行图片压缩和尺寸调整
- 支持 JPEG、PNG、GIF、BMP、TIFF 等格式
- GIF图片压缩后保持动画效果