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
	blogs := apiV1.Group("/blogs")
	blogs.Use(infrastructure.AuthMiddleware(jwtService))
	blogs.POST("", blogController.Create)
	blogs.GET("", blogController.Fetch)
	blogs.GET("/:id", blogController.GetByID)
	blogs.PUT("/:id", blogController.Update)
	blogs.DELETE("/:id", blogController.Delete)

	return router
}
