package controller

import (
	"database/sql"
	"fmt"
	"inis/app/facade"
	"inis/app/model"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Install struct {
	base
}

const (
	defaultHostPort     = 3306
	defaultCharset      = "utf8mb4"
	defaultHostName     = "localhost"
	databaseConfigFile  = "config/database.toml"
	installLockFile     = "install.lock"
	defaultAdminAccount = "admin"
	defaultAdminEmail   = "admin@admin.com"
	defaultAdminPassword = "admin123456"
	defaultAdminNickname = "系统管理员"
)

// IGET - GET请求本体
func (this *Install) IGET(ctx *gin.Context) {
	allow := map[string]any{
		"check": this.check,
	}
	this.handleHTTPMethod(ctx, allow)
}

// IPOST - POST请求本体
func (this *Install) IPOST(ctx *gin.Context) {
	allow := map[string]any{
		"lock":       this.lock,
		"init-db":    this.initDB,
		"connect-db": this.connectDB,
	}
	this.handleHTTPMethod(ctx, allow)
}

// IPUT - PUT请求本体
func (this *Install) IPUT(ctx *gin.Context) {
	this.handleHTTPMethod(ctx, map[string]any{})
}

// IDEL - DELETE请求本体
func (this *Install) IDEL(ctx *gin.Context) {
	this.handleHTTPMethod(ctx, map[string]any{})
}

// INDEX - GET请求本体
func (this *Install) INDEX(ctx *gin.Context) {
	this.json(ctx, map[string]any{
		"system": this.getSystemInfo(ctx),
	}, facade.Lang(ctx, defaultResponseMsg), DefaultSuccessCode)
}

// connectDB - 连接数据库
func (this *Install) connectDB(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{
		"hostport": defaultHostPort,
		"charset":  defaultCharset,
		"hostname": defaultHostName,
	})

	// 验证必填参数
	params, ok := this.validateRequiredParams(ctx, "username", "database", "password")
	if !ok {
		return
	}

	charset := cast.ToString(params["charset"])
	hostname := cast.ToString(params["hostname"])
	hostport := cast.ToString(params["hostport"])
	username := cast.ToString(params["username"])
	database := cast.ToString(params["database"])
	password := cast.ToString(params["password"])

	// 构建 DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=Local",
		username, password, hostname, hostport, database, charset)

	// 连接数据库
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		this.json(ctx, nil, fmt.Sprintf("数据库连接失败：%v", err.Error()), DefaultErrorCode)
		return
	}

	// 测试连接并关闭
	sqlDB, err := db.DB()
	if err != nil {
		this.json(ctx, nil, fmt.Sprintf("数据库连接失败：%v", err.Error()), DefaultErrorCode)
		return
	}
	defer func(sqlDB *sql.DB) {
		_ = sqlDB.Close()
	}(sqlDB)

	// 创建配置文件
	utils.File().Save(strings.NewReader(utils.Replace(facade.TempDatabase, map[string]any{
		"${mysql.hostname}": hostname,
		"${mysql.hostport}": hostport,
		"${mysql.username}": username,
		"${mysql.database}": database,
		"${mysql.password}": password,
		"${mysql.charset}":  charset,
		"${mysql.migrate}":  "true",
	})), databaseConfigFile)

	this.json(ctx, nil, facade.Lang(ctx, defaultResponseMsg), DefaultSuccessCode)
}

// initDB - 初始化数据库
func (this *Install) initDB(ctx *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			this.json(ctx, nil, fmt.Sprintf("数据库初始化失败：%v", err), DefaultInternalServerErrorCode)
			return
		}
	}()

	// 初始化数据库
	facade.WatchDB(false)

	// 初始化数据表
	model.InitTable()

	// 自动创建内置管理员账号
	this.createDefaultAdmin()

	this.json(ctx, nil, facade.Lang(ctx, defaultResponseMsg), DefaultSuccessCode)
}

// createDefaultAdmin - 创建默认管理员账号
func (this *Install) createDefaultAdmin() {
	defer func() {
		if err := recover(); err != nil {
			facade.Log.Error(map[string]any{
				"error": err,
			}, "创建默认管理员账号失败")
		}
	}()

	table := model.Users{}
	exist, _ := facade.DB.Model(&table).Where("account", defaultAdminAccount).Exist()
	if exist {
		return
	}

	table.Account = defaultAdminAccount
	table.Email = defaultAdminEmail
	table.Password = utils.Password.Create(defaultAdminPassword)
	table.Nickname = defaultAdminNickname

	_, err := facade.DB.Model(&table).Create(&table)
	if err != nil {
		facade.Log.Error(map[string]any{
			"error": err,
		}, "创建管理员用户失败")
	}
}

// lock - 上锁（安装锁）
func (this *Install) lock(ctx *gin.Context) {
	if ok := utils.File().Exist(databaseConfigFile); !ok {
		this.json(ctx, nil, facade.Lang(ctx, "请先完成数据库配置！"), DefaultErrorCode)
		return
	}

	// 删除安装锁
	item := utils.File().Path(installLockFile).Remove()
	if item.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "解除安装锁失败：%v", item.Error.Error()), DefaultErrorCode)
		return
	}

	this.json(ctx, nil, facade.Lang(ctx, defaultResponseMsg), DefaultSuccessCode)
}

// check - 安装状态
//
// data 为 true 表示已完成安装（口径见 facade.Installed：安装锁已解除且数据库配置已生成）。
func (this *Install) check(ctx *gin.Context) {
	this.json(ctx, facade.Installed(), facade.Lang(ctx, defaultResponseMsg), DefaultSuccessCode)
}
