package middleware

import (
	"inis/app/facade"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

// Install常量
const (
	// installPath 安装向导的前端路由（Mellow 主题里的一个页面，见 Mellow/src/views/install/Index.vue）
	installPath = "/install"
	// legacyInstallPath 老版独立安装页路径，保留跳转兼容（public/install.html 已移除）
	legacyInstallPath = "/install.html"
	devInstallPath    = "/dev/install"
	apiPathPrefix     = "/api/"
	// installGateCode / installGateName 安装引导的响应标记
	//
	// 412 在本项目里被复用了两种含义：登录失效（见 app/api/controller/comm.go）与安装引导未完成。
	// 前端据此区分（Mellow/src/api/request.js），否则未安装的站点会把「未装完」当成登录异常反复弹窗。
	installGateCode = 412
	installGateName = "install"
)

// Install - 安装引导中间件
//
// 两种状态（判定口径统一在 facade.Installed：安装锁已解除且数据库配置已生成）：
//   - 未安装：首页与老安装页路径 302 到前端安装向导 /install，
//     同时拦住 /api（此时还没有站点数据可用），安装接口 /dev/install/* 保持可访问；
//   - 已安装：安装向导页与 /dev/install/* 都不可用（302 回首页 / 412）。
//
// 用 302 而不是 301：301 会被浏览器长期缓存，装完之后用户仍会被带去安装页。
// /install 自身不做跳转（否则会自己跳自己，形成死循环）。
func Install() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		urlPath := ctx.Request.URL.Path
		method := strings.ToUpper(ctx.Request.Method)

		if !facade.Installed() {
			// 未安装：/api 一律拦住（此时还没有站点数据可用），
			// 带上 gate 标记，避免前端把它当成「登录失效」处理
			if strings.HasPrefix(urlPath, apiPathPrefix) {
				ctx.JSON(http.StatusOK, gin.H{
					"code": installGateCode,
					"msg":  "安装引导未完成，禁止访问！",
					"data": nil,
					"gate": installGateName,
				})
				ctx.Abort()
				return
			}

			// 未安装：所有「页面」访问统一引导到安装向导。
			// 以前只拦首页，导致 SPA 深链（/auth/login、/admin…）照样能打开外壳，
			// 外壳一挂载就连发好几个 /api 请求，每个都被拦成 412 —— 表现为「一直弹登录状态异常」。
			if method == "GET" && (urlPath == legacyInstallPath || isPagePath(urlPath)) {
				ctx.Redirect(http.StatusFound, installPath)
				ctx.Abort()
				return
			}
		} else {
			// 已安装：安装接口与安装向导都不再可用
			if strings.HasPrefix(urlPath, devInstallPath) {
				ctx.JSON(http.StatusOK, map[string]any{"code": 412, "msg": "程序已完成安装，禁止访问！", "data": nil})
				ctx.Abort()
				return
			}

			if urlPath == installPath || urlPath == legacyInstallPath {
				ctx.Redirect(http.StatusFound, "/")
				ctx.Abort()
				return
			}
		}

		ctx.Next()
	}
}

// isPagePath - 判断是否是「页面」请求（而不是静态资源）
//
// 用「路径末段有没有扩展名」区分：/auth/login、/user/profile 是页面（要引导到安装向导），
// /assets/index-abc123.js、/favicon.ico 是资源（必须放行，否则安装向导自己都加载不出来）。
// /install 自身不跳转（否则自己跳自己形成死循环）；/dev/install/* 与 /api/* 已在上面分流。
func isPagePath(urlPath string) bool {
	if urlPath == installPath || strings.HasPrefix(urlPath, devInstallPath) {
		return false
	}

	return path.Ext(urlPath) == ""
}
