## inis

一款基于go语言开发的强大的通用api系统
基于 [Gin](https://gin-gonic.com/zh-cn/) 二次开发，数据库基于 [Gorm](https://gorm.io/) ，设计风格参考了 [ThinkPHP 6](https://www.kancloud.cn/manual/thinkphp6_0/1037479)

## 特点和功能
- 毫秒级响应
- 内置大量基础功能

## 配置文件
> 配置文件在 `config` 目录下

## 运行
> 运行前请先安装 [Go](https://golang.org/dl/) ，然后在项目根目录下执行 `go run main.go` 即可   
> 如需后台运行，可以使用 `go build -ldflags -H=windowsgui` 命令编译，[bee](https://github.com/beego/bee) 工具也可以使用 `bee pack -ba="-ldflags -H=windowsgui"` 命令打包

## 打包
> 部署前请先安装 [Go](https://golang.org/dl/) ，然后在项目根目录下执行 `go build -ldflags "-s -w" -o inis main.go` 即可，编译完成后会生成一个的可执行文件，将其放到服务器上即可

## 未来规划
inis 正在持续扩展中，计划新增以下关键功能：
### 通信协议扩展
- WebSocket 实时通信模块
- GraphQL 接口支持（基于 gqlgen）
- 多数据格式适配（XML、Protobuf 等）
### 核心能力强化
- 动态数据验证系统（支持自定义规则与多语言提示）
- 消息通知中心（整合短信、邮件等渠道）
- 高级文件管理（分片上传、断点续传、CDN 集成）
### 场景化模块
- 电商解决方案（订单、支付、库存等）
- 内容管理增强（审核、推荐算法）
- IoT 设备接入（MQTT 协议支持）

## 更新日志
[1.0.0]更新内容如下【开发中】
- 重构：默认前后台主题
- 新增：阿里云号码验证短信方式
- 新增：媒体管理
- 优化：去除富文本编辑器
- 优化：去除inis社区账号绑定
- 优化：后台提供接口，可自定义签到，经验，等级