# inis

inis 是一款基于 Go 语言开发的高性能内容管理系统（CMS），基于 Gin 框架二次开发，采用 Gorm 作为数据库 ORM 工具，设计风格参考 ThinkPHP 6 的简洁架构理念。系统以 "轻量核心、高效响应、灵活扩展" 为核心定位，致力于为开发者提供易上手、具备良好扩展基础的 CMS 解决方案，同时满足企业级应用的性能与安全需求。

## 核心特性

- 🚀 **高性能**：基于 Go 语言和 Gin 框架，提供毫秒级响应能力
- 🔒 **安全可靠**：多层安全防护机制，包括安装锁、API 签名、QPS 限流等
- 📦 **轻量灵活**：简洁的架构设计，易于理解和扩展
- 🌍 **国际化**：内置多语言支持，方便全球化部署
- 💾 **高效缓存**：智能缓存策略，提升数据查询效率

## 快速开始
后端主程序开源仓库：[inisv1](https://github.com/zhu885744/inisv1)<br>
默认主题Github开源仓库：[xiao-inisv1-vue](https://github.com/zhu885744/xiao-inisv1-vue)

### 开发环境运行
#### 步骤 1：安装依赖
1. 安装 [Go](https://golang.org/dl/) 1.25.0+ 版本
2. 克隆项目代码：
   ```bash
   git clone https://github.com/zhu885744/inisv1.git
   cd inisv1
   ```
3. 安装项目依赖：
   ```bash
   go mod tidy
   ```

#### 步骤 2：配置数据库
1. 创建数据库（MySQL 8.0+ 或 PostgreSQL 14+）
2. 复制配置文件模板并修改：
   ```bash
   cp config/app.toml.example config/app.toml
   ```
3. 编辑 `config/app.toml` 文件，配置数据库连接信息

#### 步骤 3：运行项目
```bash
go run main.go
```

> 访问地址：http://localhost:8080
> 默认管理员账号：admin
> 默认管理员密码：123456

### 打包教程

#### 使用 build.bat 脚本（推荐）
1. 在项目根目录下双击 `build.bat` 文件
2. 根据提示选择编译平台（Windows/Linux）
3. 等待编译完成，生成的可执行文件会放在 `output` 目录

#### 手动打包

##### Windows 平台
```bash
# 编译为可执行文件
go build -o inis.exe main.go

# 后台运行版本（无控制台窗口）
go build -ldflags -H=windowsgui -o inis.exe main.go
```

##### Linux 平台
```bash
# 编译为可执行文件
go build -o inis main.go

# 设置可执行权限
chmod +x inis
```

##### 使用 bee 工具打包
```bash
# 安装 bee 工具
go get github.com/beego/bee

# 打包为 Windows 后台运行版本
bee pack -ba="-ldflags -H=windowsgui"

# 打包为 Linux 版本
bee pack -ba="-ldflags -s -w"
```

### 部署教程

#### 1. 服务器准备
- **操作系统**：Debian 12 / Ubuntu Server 22.04
- **CPU**：2 核及以上
- **内存**：2GB 及以上
- **存储**：10GB SSD
- **网络**：5Mbps 及以上带宽

#### 2. 软件安装
```bash
# 更新系统
apt update && apt upgrade -y

# 安装必要软件
apt install -y nginx mysql-server redis-server

# 安装 Go
download_url=$(curl -s https://golang.org/dl/ | grep -oP 'https://dl.google.com/go/go\d+\.\d+\.\d+\.linux-amd64\.tar.gz' | head -1)
wget $download_url
tar -C /usr/local -xzf go*.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

#### 3. 数据库配置
```bash
# 登录 MySQL
mysql -u root -p

# 创建数据库
CREATE DATABASE inis CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

# 创建用户并授权
CREATE USER 'inis'@'localhost' IDENTIFIED BY 'your_password';
GRANT ALL PRIVILEGES ON inis.* TO 'inis'@'localhost';
FLUSH PRIVILEGES;
EXIT;
```

#### 4. 部署应用
1. 上传编译好的可执行文件到服务器
2. 创建配置文件：
   ```bash
   mkdir -p /opt/inis/config
   # 上传 config/app.toml 文件到 /opt/inis/config/
   ```
3. 创建启动脚本：
   ```bash
   cat > /opt/inis/start.sh << 'EOF'
   #!/bin/bash
   cd /opt/inis
   ./inis
   EOF
   chmod +x /opt/inis/start.sh
   ```
4. 创建系统服务：
   ```bash
   cat > /etc/systemd/system/inis.service << 'EOF'
   [Unit]
   Description=inis CMS
   After=network.target mysql.service

   [Service]
   Type=simple
   WorkingDirectory=/opt/inis
   ExecStart=/opt/inis/start.sh
   Restart=on-failure

   [Install]
   WantedBy=multi-user.target
   EOF

   systemctl daemon-reload
   systemctl enable inis
   systemctl start inis
   ```

#### 5. Nginx 配置
```bash
cat > /etc/nginx/sites-available/inis << 'EOF'
server {
    listen 80;
    server_name example.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # 静态文件处理
    location /static/ {
        root /opt/inis/public;
        expires 30d;
    }
}
EOF

ln -s /etc/nginx/sites-available/inis /etc/nginx/sites-enabled/
nginx -t
systemctl reload nginx
```

#### 6. 部署主题
1. 克隆主题代码：
   ```bash
   git clone https://github.com/zhu885744/xiao-inisv1-vue.git /opt/inis/public
   ```
2. 配置主题：
   ```bash
   # 编辑主题配置文件
   nano /opt/inis/public/config.js
   ```

### 部署后维护

#### 1. 日志管理
- 应用日志：`/opt/inis/logs/`
- Nginx 日志：`/var/log/nginx/`

#### 2. 定期备份
```bash
# 创建备份脚本
cat > /opt/inis/backup.sh << 'EOF'
#!/bin/bash
DATE=$(date +%Y%m%d)

# 备份数据库
mysqldump -u inis -p'your_password' inis > /opt/inis/backup/inis_${DATE}.sql

# 备份配置文件
cp -r /opt/inis/config /opt/inis/backup/config_${DATE}

# 清理30天前的备份
find /opt/inis/backup -name "*" -mtime +30 -delete
EOF

chmod +x /opt/inis/backup.sh
mkdir -p /opt/inis/backup

# 添加到定时任务
crontab -e
# 添加以下行（每天凌晨2点执行备份）
0 2 * * * /opt/inis/backup.sh
```

#### 3. 监控
- 系统监控：可以使用 Prometheus + Grafana
- 应用监控：可以使用 `systemctl status inis` 查看应用状态

### 常见部署问题

| 问题 | 原因 | 解决方案 |
|------|------|----------|
| 端口被占用 | 8080 端口已被其他服务占用 | 修改 `config/app.toml` 中的端口配置 |
| 数据库连接失败 | 数据库配置错误 | 检查数据库连接信息和权限 |
| 404 错误 | 主题文件未部署 | 确保主题文件已正确部署到 `public` 目录 |
| 502 错误 | 应用未运行或端口错误 | 检查应用运行状态和 Nginx 配置 |

> 系统默认提供一个默认主题，内置完整的管理后台
> Github开源仓库：[xiao-inisv1-vue](https://github.com/zhu885744/xiao-inisv1-vue)

## 系统架构

### 技术栈
- **底层框架**：基于 Gin 实现的高性能 HTTP 服务，毫秒级响应能力满足高并发场景
- **数据交互**：集成 Gorm 实现数据库操作抽象，支持多种关系型数据库（MySQL、PostgreSQL 等）
- **缓存系统**：支持文件缓存、内存缓存和 Redis 缓存，灵活的缓存策略
- **模板引擎**：使用 Go 原生模板引擎，支持服务端渲染
- **权限控制**：基于 RBAC 模型的权限管理系统

### 架构分层
```
┌─────────────────────────────────────────
│          表现层 (Presentation)          
│  - 路由层 (Route)                       
│  - 控制器层 (Controller)                
│  - 中间件层 (Middleware)                
└─────────────────┬───────────────────────
                  │
┌─────────────────▼───────────────────────
│          业务层 (Business)              
│  - 门面层 (Facade)                      
│  - 服务层 (Service)                     
│  - 验证器 (Validator)                   
└─────────────────┬───────────────────────
                  │
┌─────────────────▼───────────────────────
│          数据层 (Data)                  
│  - 模型层 (Model)                       
│  - ORM (Gorm)                          
│  - 数据库 (MySQL/PostgreSQL)            
│  - 缓存 (Redis/Memory/file)             
└─────────────────────────────────────────
```

### 核心功能
- **配置管理系统**：支持动态配置存储与缓存，灵活管理系统参数
- **多语言国际化**：内置中、英、日、韩、俄等语言包，支持自定义扩展
- **安全防护机制**：包含安装锁（install.lock）、API 签名验证、请求限流（QPS 控制）等基础防护
- **媒体资源处理**：支持图片动态压缩、格式转换及多种存储模式（本地存储为基础，预留云存储扩展接口）
- **高效缓存策略**：实现内存缓存机制，支持按标签批量清理缓存，提升数据查询效率
- **用户权限系统**：基于 RBAC 模型的用户权限管理，支持角色和权限组管理
- **文章管理系统**：支持文章创建、编辑、审核、发布、分类、标签等完整功能
- **评论系统**：支持文章评论、评论审核、评论回复等功能
- **社交登录**：支持邮箱、手机号验证码登录，以及第三方社交登录

### 功能模块

#### 1. 用户模块
- 用户注册/登录
- 用户信息管理
- 密码找回
- 社交登录（邮箱/手机号）
- 用户等级系统
- 经验值管理

#### 2. 内容模块
- 文章管理（CRUD）
- 文章分类（支持多级分类）
- 标签管理
- 文章审核
- 文章置顶
- 浏览量统计

#### 3. 权限模块
- 角色管理
- 权限分组
- 权限规则
- 用户组管理

#### 4. 系统模块
- 系统配置
- API 密钥管理
- 友情链接
- 页面管理
- 轮播管理

#### 5. 互动模块
- 评论系统
- 点赞/收藏/分享

## 配置说明

### 配置文件
配置文件位于 `config` 目录下，主要包括：
- `app.go`：应用配置核心逻辑（启动服务等）
- `i18n/`：国际化语言配置目录，包含各语言的翻译文件

### 版本管理
后端版本号定义在 `app/facade/const.go` 文件中，可根据需要修改。

### API 接口文档
本文档详细标注了如何在开发主题中使用自定义接口。

登录后，可通过访问 swagger 访问地址：https://{host}/swagger/index.html 查看所有 API。

## 目录结构

```
inisv1/
├── .gitignore          # Git 忽略文件配置
├── LICENSE             # 项目许可证
├── README.md           # 项目说明文档（功能、运行、规划等）
├── build.bat           # 编译脚本（生成可执行文件）
├── go.mod              # Go 模块依赖配置
├── go.sum              # 依赖校验文件
├── inis.sh             # linux 安装脚本
├── install.lock        # 安装锁文件（标记是否完成初始化）
├── main.go             # 程序入口文件
├── config/             # 配置文件目录
│   ├── .gitignore      # 配置目录的 Git 忽略规则
│   ├── app.go          # 应用配置核心逻辑（启动服务等）
│   └── i18n/           # 国际化语言配置
│       ├── en-us.json   # 英语语言包
│       ├── ja-jp.json   # 日语语言包
│       ├── ko-kr.json   # 韩语语言包
│       ├── ru-ru.json   # 俄语语言包
│       └── zh-cn.json   # 中文语言包
├── docs/               # API 文档目录
│   ├── docs.go         # Swagger 文档生成
│   ├── swagger.json    # Swagger JSON 文件
│   └── swagger.yaml    # Swagger YAML 文件
└── app/                # 核心业务代码目录
    ├── api/            # API 接口相关（控制器、路由）
    │   ├── controller/ # API 控制器
    │   ├── middleware/ # API 中间件
    │   └── route/      # API 路由
    ├── dev/            # 开发相关功能（系统信息、调试等）
    │   ├── controller/ # 开发控制器
    │   └── route/      # 开发路由
    ├── facade/         # 门面层（封装核心服务、工具）
    ├── index/          # 首页相关路由/控制器
    │   ├── controller/ # 首页控制器
    │   └── route/      # 首页路由
    ├── middleware/     # 全局中间件（CORS、权限校验等）
    ├── model/          # 数据模型（与数据库交互）
    ├── socket/         # WebSocket 相关（实时通信）
    │   ├── controller/ # WebSocket 控制器
    │   ├── middleware/ # WebSocket 中间件
    │   └── route/      # WebSocket 路由
    ├── timer/          # 定时任务（日志清理等）
    └── validator/      # 数据验证器
```

## 部署环境

### 推荐配置
- **操作系统**：Debian 12 / Ubuntu Server 22.04
- **CPU**：2 核及以上
- **内存**：2GB 及以上
- **存储**：10GB SSD
- **网络**：5Mbps 及以上带宽

### 软件要求
- **Go**：1.25.0+
- **数据库**：MySQL 8.0+ / PostgreSQL 14+
- **缓存**：Redis 7.0+（可选）
- **Web 服务器**：Nginx 1.24+（反向代理）

## 开发指南

### 代码规范
- 遵循 Go 语言官方代码规范
- 使用 `gofmt` 格式化代码
- 保持函数简洁，单一职责原则
- 合理使用注释说明复杂逻辑

### 添加新功能
1. 在 `app/model/` 创建数据模型
2. 在 `app/api/controller/` 创建控制器
3. 在 `app/api/route/` 注册路由
4. 在 `app/validator/` 添加验证器（如需要）
5. 编写单元测试

### 数据库迁移
系统使用 Gorm 的 AutoMigrate 功能自动管理数据库结构，确保模型定义正确即可。

## 常见问题

### Q: 如何修改默认端口？
A: 在 `config/app.toml` 中修改端口配置。

### Q: 如何切换数据库？
A: 修改数据库配置文件，并确保安装了对应的数据库驱动。

### Q: 如何启用缓存？
A: 在配置文件中设置缓存相关参数，支持文件缓存、内存缓存和 Redis 缓存。

## 贡献指南

欢迎提交 Issue 和 Pull Request 来帮助改进项目！

## 许可证

本项目采用 [Apache-2.0 license](LICENSE) 许可证。

## 联系方式

如有问题或建议，请通过以下方式联系：
- GitHub Issues
- 交流群：119300889
- 邮箱：xz@zhuxu.asia

## 致谢
原作者「陈兔子」：[https://github.com/racns](https://github.com/racns)<br>
原开源仓库「已停更」：[https://github.com/inis-io/inis](https://github.com/inis-io/inis)<br>
感谢所有为开源社区做出贡献的开发者！