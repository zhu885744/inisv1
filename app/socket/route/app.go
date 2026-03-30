package route

import (
	apimiddleware "inis/app/api/middleware"
	"inis/app/socket/controller"
	socketmiddleware "inis/app/socket/middleware"

	"github.com/gin-gonic/gin"
)

func Route(engine *gin.Engine) {
	socket := engine.Group("/socket", apimiddleware.Jwt(), socketmiddleware.App)
	{
		class := controller.Index{}
		socket.GET("", class.Connect)
		socket.GET("/", class.Connect)
	}
}
