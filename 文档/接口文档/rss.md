## RSS 订阅 API

### 接口概述

RSS订阅控制器，提供站点内容的 RSS 订阅功能。

### 状态码规范

| 状态码 | 含义 | 使用场景 |
| :--- | :--- | :--- |
| 200 | 请求成功 | 成功返回RSS内容 |
| 405 | 方法不允许 | 调用了不存在的方法 |
| 500 | 服务器错误 | 服务器内部错误 |

### 接口列表

#### 1. 获取RSS订阅 [index]

**请求方式**: `GET`

**请求地址**: `/api/rss` 或 `/api/rss/index`

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| limit | int | 否 | 获取文章数量，默认20，最大100 |
| url | string | 否 | 站点URL，用于生成链接 |
| full | bool | 否 | 是否返回完整内容，默认false |

**响应格式**: `application/xml; charset=utf-8`

**响应示例**:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
  <channel>
    <title><![CDATA[inis]]></title>
    <link><![CDATA[https://example.com]]></link>
    <description><![CDATA[站点描述]]></description>
    <language>zh-cn</language>
    <lastBuildDate>Mon, 01 Jan 2024 00:00:00 +0000</lastBuildDate>
    <atom:link href="https://example.com/rss" rel="self" type="application/rss+xml"/>
    <item>
      <title><![CDATA[文章标题]]></title>
      <link><![CDATA[https://example.com/archives/1]]></link>
      <guid isPermaLink="true"><![CDATA[https://example.com/archives/1]]></guid>
      <description><![CDATA[文章摘要]]></description>
      <pubDate>Mon, 01 Jan 2024 00:00:00 +0000</pubDate>
    </item>
  </channel>
</rss>
```

### 特殊说明

1. **内容来源**: 仅返回审核通过的文章
2. **内容选择**: `full=true` 时返回文章完整内容，否则返回摘要
3. **站点信息**: URL优先使用参数传入的值，其次使用配置文件中的值，最后使用主机和端口生成