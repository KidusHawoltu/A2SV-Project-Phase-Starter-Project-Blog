package routers

import (
	"A2SV_Starter_Project_Blog/Delivery/controllers"
	infrastructure "A2SV_Starter_Project_Blog/Infrastructure"

	"github.com/gin-gonic/gin"
)

// SetupRouter sets up all API routes for the blog platform
func SetupRouter(
	userController *controllers.UserController,
	blogController *controllers.BlogController,
	aiController *controllers.AIController,
	jwtService infrastructure.JWTService,
) *gin.Engine {

	router := gin.Default()
	apiV1 := router.Group("/api/v1")

	// ---------------------
	// Auth Routes (Public)
	// ---------------------
	auth := apiV1.Group("/auth")
	{
		auth.POST("/register", userController.Register)
		auth.GET("/activate", userController.ActivateAccount)
		auth.POST("/login", userController.Login)
		auth.POST("/refresh", userController.RefreshToken)
		auth.POST("/logout", userController.Logout)
	}

	// -------------------------
	// Password Routes (Public)
	// -------------------------
	password := apiV1.Group("/password")
	{
		password.POST("/forget", userController.ForgetPassword)
		password.POST("/reset", userController.ResetPassword)
	}

	// ------------------------
	// Profile Routes (Private)
	// ------------------------
	profile := apiV1.Group("/profile")
	profile.Use(infrastructure.AuthMiddleware(jwtService))
	{
		profile.GET("", userController.GetProfile)
		profile.PUT("", userController.UpdateProfile)
	}

	// ------------------------
	// Blog Routes (Mixed)
	// ------------------------
	publicBlogs := apiV1.Group("/blogs")
	{
		publicBlogs.GET("", blogController.SearchAndFilter)
		publicBlogs.GET("/:id", blogController.GetByID)

		protectedBlogs := publicBlogs.Group("")
		protectedBlogs.Use(infrastructure.AuthMiddleware(jwtService))
		{
			protectedBlogs.POST("", blogController.Create)
			protectedBlogs.PUT("/:id", blogController.Update)
			protectedBlogs.DELETE("/:id", blogController.Delete)
			protectedBlogs.POST("/:id/interact", blogController.InteractWithBlog)
		}
	}

	// ------------------------
	// AI Routes (Protected)
	// ------------------------
	ai := apiV1.Group("/ai")
	ai.Use(infrastructure.AuthMiddleware(jwtService))
	{
		ai.POST("/suggest", aiController.Suggest)
	}

	return router
}
