
package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/bikhizlit/todo-go/internal/controllers"
)

func RegisterTaskRoutes(
	router *gin.Engine,
	taskController *controllers.TaskController,
) {
	api := router.Group("/api/v1")

	api.GET("/tasks", taskController.List)
	api.POST("/tasks", taskController.Create)
	api.GET("/tasks/:id", taskController.Get)
	api.PATCH("/tasks/:id", taskController.Update)
	api.DELETE("/tasks/:id", taskController.Delete)
}
