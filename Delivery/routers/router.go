package routers

import (
	"A2SV_Starter_Project_Blog/Delivery/controllers"

	"github.com/gin-gonic/gin"
)

func SetupBlogRouter(group *gin.Engine, blogController *controllers.BlogController) {
	blogRoutes := group.Group("/blogs")
	{
		blogRoutes.POST("", blogController.Create)
		blogRoutes.GET("", blogController.Fetch)
		blogRoutes.GET("/:id", blogController.GetByID)
		blogRoutes.PUT("/:id", blogController.Update)
		blogRoutes.DELETE("/:id", blogController.Delete)
	}
}
