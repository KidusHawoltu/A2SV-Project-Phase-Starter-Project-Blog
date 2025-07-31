package routers

import (
	"A2SV_Starter_Project_Blog/Delivery/controllers"
	infrastructure "A2SV_Starter_Project_Blog/Infrastructure"

	"github.com/gin-gonic/gin"
)

func SetupRouter(
	userController *controllers.UserController,
	jwtService infrastructure.JWTService,
) *gin.Engine {

	router := gin.Default()
	
	apiV1 := router.Group("/api/v1")
	{
		// Public routes
		auth := apiV1.Group("/auth")
		auth.POST("/register", userController.Register)
		auth.POST("/login", userController.Login)
		
		// Protected routes
		profile := apiV1.Group("/profile")
		profile.Use(infrastructure.AuthMiddleware(jwtService)) // Apply middleware
		profile.GET("", userController.GetProfile)
	}

	return router
}