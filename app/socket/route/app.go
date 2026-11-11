package route

import (
	apimiddleware "inis/app/api/middleware"
	"inis/app/socket/controller"
	socketmiddleware "inis/app/socket/middleware"

	"github.com/gin-gonic/gin"
)

func Route(engine *gin.Engine) {
	// IpBlack 必须在 Auth 之前：WebSocket 握手同样要受 IP 封禁约束，
	// 否则被封禁的 IP 只要改走 /socket 就能继续连（此前 /socket 完全没有
	// 走 api 分组的任何风控中间件）。
	socket := engine.Group("/socket", apimiddleware.IpBlack(), apimiddleware.Auth(), socketmiddleware.App)
	{
		class := controller.Index{}
		socket.GET("", class.Connect)
		socket.GET("/", class.Connect)
	}
}
