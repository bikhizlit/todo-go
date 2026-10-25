package tasks

import (
	
	"github.com/gin-gonic/gin"
)

func RegisterTaskRoutes(
	router *gin.Engine,
	taskController *TaskController,
) {
	api := router.Group("/api/v1")

	api.GET("/tasks", taskController.List)
	api.POST("/tasks", taskController.Create)
	api.GET("/tasks/:id", taskController.Get)
	api.PATCH("/tasks/:id", taskController.Update)
	api.DELETE("/tasks/:id", taskController.Delete)
}
