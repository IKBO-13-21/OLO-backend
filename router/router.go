package router

import "github.com/gin-gonic/gin"

var r *gin.Engine

func initRouter(userHander *aut) {
	r = gin.Default()

}
