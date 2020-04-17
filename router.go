package main

import "github.com/gin-gonic/gin"

func initRouter() *gin.Engine {
	router := gin.New()

	router.Use(GinMiddleware("http://localhost:5000"))

	return router
}
