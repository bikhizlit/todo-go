package tasks

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type TaskController struct {
	service *TaskService
}

func NewTaskController(service *TaskService) *TaskController {
	return &TaskController{service: service}
}

func (h *TaskController) List(c *gin.Context) {
	result, err := h.service.List(c.Request.Context())
	if err != nil {
		h.respondError(c, err, "list tasks")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *TaskController) Create(c *gin.Context) {
	var input CreateTaskInput

	if err := decodeJSON(c, &input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	task, err := h.service.Create(c.Request.Context(), input)
	if err != nil {
		h.respondError(c, err, "create task")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": task})
}

func (h *TaskController) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	task, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err, "get task")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": task})
}

func (h *TaskController) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var input UpdateTaskInput

	if err := decodeJSON(c, &input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	task, err := h.service.Update(c.Request.Context(), id, input)
	if err != nil {
		h.respondError(c, err, "update task")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": task})
}

func (h *TaskController) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		h.respondError(c, err, "delete task")
		return
	}

	c.Status(http.StatusNoContent)
}

func parseID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return 0, false
	}

	validID, err := ValidateID(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return 0, false
	}

	return validID, true
}

func decodeJSON(c *gin.Context, destination any) error {
	c.Request.Body = http.MaxBytesReader(
		c.Writer,
		c.Request.Body,
		1<<20,
	)

	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return err
	}

	var extra any
	if !errors.Is(decoder.Decode(&extra), io.EOF) {
		return errors.New("request must contain exactly one JSON value")
	}

	return nil
}

func (h *TaskController) respondError(
	c *gin.Context,
	err error,
	operation string,
) {
	switch {
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"error": "task not found",
		})

	case errors.Is(err, ErrInvalidTitle):
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})

	default:
		log.Printf("%s: %v", operation, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
	}
}
