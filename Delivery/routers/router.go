package routers

import (
	"A2SV_Starter_Project_Blog/Delivery/controllers"
	infrastructure "A2SV_Starter_Project_Blog/Infrastructure"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// SetupRouter sets up all API routes for the blog platform
func SetupRouter(
	userController *controllers.UserController,
	blogController *controllers.BlogController,
	aiController *controllers.AIController,
	commentController *controllers.CommentController,
	oauthController *controllers.OAuthController,
	jwtService infrastructure.JWTService,
	rateLimiter *infrastructure.RateLimiter,
) *gin.Engine {

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// --- Rate Limiting ---
	// General api limit for all routes
	generalAPILimiter := rateLimiter.LimiterMiddleware(200, 1*time.Minute, "userID")
	// Strict limit for sensitive routes
	strictAPILimiter := rateLimiter.LimiterMiddleware(10, 1*time.Minute, "userID")
	// Highest limit for expeinsive routes
	aiAPILimiter := rateLimiter.LimiterMiddleware(10, 1*time.Hour, "userID")

	apiV1 := router.Group("/api/v1")

	// ---------------------
	// Auth Routes (Public)
	// ---------------------
	auth := apiV1.Group("/auth")
	auth.Use(strictAPILimiter)
	{
		auth.POST("/register", userController.Register)
		auth.GET("/activate", userController.ActivateAccount)
		auth.POST("/login", userController.Login)
		auth.POST("/refresh", userController.RefreshToken)
		auth.POST("/logout", userController.Logout)

		google := auth.Group("/google")
		{
			google.POST("/callback", oauthController.HandleGoogleCallback)
		}
	}

	// -------------------------
	// Password Routes (Public)
	// -------------------------
	password := apiV1.Group("/password")
	password.Use(strictAPILimiter)
	{
		password.POST("/forget", userController.ForgetPassword)
		password.POST("/reset", userController.ResetPassword)
	}

	// ------------------------
	// Profile Routes (Private)
	// ------------------------
	profile := apiV1.Group("/profile")
	profile.Use(infrastructure.AuthMiddleware(jwtService), generalAPILimiter)
	{
		profile.GET("", userController.GetProfile)
		profile.PUT("", userController.UpdateProfile)
	}

	// ------------------------
	// Admin Routes
	// ------------------------
	admin := apiV1.Group("/admin")
	admin.Use(infrastructure.AuthMiddleware(jwtService), infrastructure.AdminOnlyMiddleware(), generalAPILimiter)
	{
		admin.GET("/users", userController.SearchAndFilter)
		admin.PATCH("/users/:userID/role", userController.SetUserRole)
	}

	// ------------------------
	// Blog Routes (Mixed)
	// ------------------------
	publicBlogs := apiV1.Group("/blogs")
	publicBlogs.Use(generalAPILimiter)
	{
		publicBlogs.GET("", blogController.SearchAndFilter)
		publicBlogs.GET("/:blogID", blogController.GetByID)
		publicBlogs.GET("/:blogID/comments", commentController.GetCommentsForBlog)
	}

	protectedBlogs := apiV1.Group("/blogs")
	protectedBlogs.Use(infrastructure.AuthMiddleware(jwtService), strictAPILimiter)
	{
		protectedBlogs.POST("", blogController.Create)
		protectedBlogs.PUT("/:blogID", blogController.Update)
		protectedBlogs.DELETE("/:blogID", blogController.Delete)
		protectedBlogs.POST("/:blogID/interact", blogController.InteractWithBlog)
		// If it is a top level comment, parent Id will be null
		protectedBlogs.POST("/:blogID/comments", commentController.CreateComment)
	}

	// ------------------------
	// AI Routes (Protected)
	// ------------------------
	ai := apiV1.Group("/ai")
	ai.Use(infrastructure.AuthMiddleware(jwtService), aiAPILimiter)
	{
		ai.POST("/suggest", aiController.Suggest)
	}

	// ------------------------
	// Comment Routes
	// ------------------------
	comments := apiV1.Group("/comments")
	comments.Use(generalAPILimiter)
	{
		comments.GET("/:commentID/replies", commentController.GetRepliesForComment)
	}
	protectedComments := apiV1.Group("/comments")
	protectedComments.Use(infrastructure.AuthMiddleware(jwtService), strictAPILimiter)
	{
		protectedComments.PUT("/:commentID", commentController.UpdateComment)
		protectedComments.DELETE("/:commentID", commentController.DeleteComment)
	}

	return router
}
