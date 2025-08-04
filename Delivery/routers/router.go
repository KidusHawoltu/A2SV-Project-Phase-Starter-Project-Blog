package routers

import (
	"A2SV_Starter_Project_Blog/Delivery/controllers"
	infrastructure "A2SV_Starter_Project_Blog/Infrastructure"

	"github.com/gin-gonic/gin"
)

// SetupRouter sets up the main router with both auth/profile and blog routes
func SetupRouter(
	userController *controllers.UserController,
	blogController *controllers.BlogController,
	jwtService infrastructure.JWTService,
) *gin.Engine {

	router := gin.Default()
	apiV1 := router.Group("/api/v1")

	// ---------------------
	// Auth Routes (Public)
	// ---------------------
	auth := apiV1.Group("/auth")
	auth.POST("/register", userController.Register)
	auth.POST("/login", userController.Login)

	// ------------------------
	// Profile Routes (Private)
	// ------------------------
	profile := apiV1.Group("/profile")
	profile.Use(infrastructure.AuthMiddleware(jwtService))
	profile.GET("", userController.GetProfile)

	// ------------------------
	// Blog Routes (Protected)
	// ------------------------
	publicBlogs := apiV1.Group("/blogs")
	{
		publicBlogs.GET("", blogController.SearchAndFilter)
		publicBlogs.GET("/:id", blogController.GetByID)
	}

	protectedBlogs := publicBlogs.Group("")
	protectedBlogs.Use(infrastructure.AuthMiddleware(jwtService))
	{
		protectedBlogs.POST("", blogController.Create)
		protectedBlogs.PUT("/:id", blogController.Update)
		protectedBlogs.DELETE("/:id", blogController.Delete)
	}

	return router
}
