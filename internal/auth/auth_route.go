package auth

import "github.com/gin-gonic/gin"

func RegisterUserRoutes(
	router *gin.Engine,
	handler *Handler,
	tokens *AccessTokenManager,
) {
	api := router.Group("/api/v1/auth")

	api.POST("/register", handler.Register)
	api.POST("/login", handler.Login)

	protected := api.Group("")
	protected.Use(Authenticate(tokens))

	protected.GET("/me", handler.Me)
}
