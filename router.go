package main

import "github.com/gin-gonic/gin"

func initRouter() *gin.Engine {
	router := gin.New()

	router.Use(GinMiddleware("http://localhost:8081"))

	return router
}
