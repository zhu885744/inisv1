# 友链状态检测接口使用指南
## 三个接口统一用法
### 通用参数 status
参数 类型 默认值 说明 status bool false 是否检测友链在线状态和响应速度

## 1. /api/links/one - 获取单个友链
基础用法（不检测状态）：

```
curl "http://localhost:8080/api/links/one?id=1"
```
检测状态用法：

```
curl "http://localhost:8080/api/links/one?id=1&status=true"
```
响应示例（带状态检测）：

```
{
  "code": 200,
  "msg": "数据请求成功！",
  "data": {
    "id": 1,
    "nickname": "朱某的生活印记",
    "url": "https://zhuxu.asia",
    "description": "没有销声匿迹，我在热
    爱生活",
    "online": true,
    "responseTime": 156,
    "response_time": 156
  }
}
```
## 2. /api/links/all - 获取友链列表
基础用法（不检测状态）：

```
curl "http://localhost:8080/api/links/all?page=1&limit=10"
```
检测状态用法：

```
curl "http://localhost:8080/api/links/all?page=1&limit=10&status=true"
```
响应示例（带状态检测）：

```
{
  "code": 200,
  "msg": "数据请求成功！",
  "data": {
    "data": [
      {
        "id": 1,
        "nickname": "示例网站1",
        "url": "https://example1.com",
        "online": true,
        "responseTime": 123,
        "response_time": 123
      },
      {
        "id": 2,
        "nickname": "示例网站2",
        "url": "https://example2.com",
        "online": false,
        "responseTime": 8000,
        "response_time": 8000
      }
    ],
    "count": 2,
    "page": 1
  }
}
```
## 3. /api/links/rand - 随机获取友链
基础用法（不检测状态）：

```
curl "http://localhost:8080/api/links/rand?limit=5"
```
检测状态用法：

```
curl "http://localhost:8080/api/links/rand?limit=5&status=true"
```
响应示例（带状态检测）：

```
{
  "code": 200,
  "msg": "好的！",
  "data": [
    {
      "id": 3,
      "nickname": "随机网站",
      "url": "https://random.com",
      "online": true,
      "responseTime": 456,
      "response_time": 456
    }
  ]
}
```
## 响应字段说明
字段 类型 说明 online bool 是否在线 responseTime int 响应时间（毫秒）- 兼容旧代码 response_time int 响应时间（毫秒）- 规范命名

## 状态判断规则
状态码/情况 判定结果 200 ≤ code < 500 ✅ 在线（包括 403、404） 500+ ❌ 不在线 超时（8秒） ❌ 不在线 无法连接 ❌ 不在线

## 性能特性
特性 说明 最大并发 10 个 goroutine 同时检测 单链接超时 8 秒 整体超时 30 秒 URL 清洗 自动去除空格、引号、反引号

## 使用场景建议
```
# 首页展示友链（快速响应，不检测状态）
curl "http://localhost:8080/api/links/all"

# 友链管理页面（需要状态检测）
curl "http://localhost:8080/api/links/all?status=true"

# 侧边栏随机展示（检测状态，只显示在线的）
curl "http://localhost:8080/api/links/rand?limit=3&status=true"
```