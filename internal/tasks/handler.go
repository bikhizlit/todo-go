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

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")

	api.GET("/tasks", h.List)
	api.POST("/tasks", h.Create)
	api.GET("/tasks/:id", h.Get)
	api.PATCH("/tasks/:id", h.Update)
	api.DELETE("/tasks/:id", h.Delete)
}

func (h *Handler) List(c *gin.Context) {
	result, err := h.repo.List(c.Request.Context())
	if err != nil {
		log.Printf("list tasks: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to load tasks",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) Create(c *gin.Context) {
	var input CreateTaskInput

	if err := decodeJSON(c, &input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	task, err := h.repo.Create(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, ErrInvalidTitle) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		log.Printf("create task: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create task",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": task})
}

func (h *Handler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	task, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err, "get task")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": task})
}

func (h *Handler) Update(c *gin.Context) {
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

	task, err := h.repo.Update(c.Request.Context(), id, input)
	if err != nil {
		h.respondError(c, err, "update task")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": task})
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		h.respondError(c, err, "delete task")
		return
	}

	c.Status(http.StatusNoContent)
}

func parseID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid task ID",
		})
		return 0, false
	}

	validID, err := ValidateID(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid task ID",
		})
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
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request must contain one JSON value")
		}
		return err
	}

	return nil
}

func (h *Handler) respondError(
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
