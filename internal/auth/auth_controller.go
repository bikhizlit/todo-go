package auth

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func NewHandler(service *Service) *Handler {
	return &Handler{Service: service}
}

func withTimeout(c *gin.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.Request.Context(), 5*time.Second)
}

func (h *Handler) Register(c *gin.Context) {

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid JSON or required fields",
		})
		return
	}

	ctx, cancel := withTimeout(c)
	defer cancel()

	result, err := h.Service.Register(
		ctx,
		req.Name,
		req.Email,
		req.Password,
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailExists):
			c.JSON(http.StatusConflict, gin.H{
				"error": "email already registered",
			})

		case errors.Is(err, ErrInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid name, email or password",
			})

		default:
			  log.Printf("registration failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unale to register",
			})
		}
		return
	}
	c.JSON(http.StatusCreated, result)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid json or required fields",
		})
		return
	}

	ctx, cancel := withTimeout(c)
	defer cancel()

	result, err := h.Service.Login(
		ctx, req.Email, req.Password,
	)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid email or password",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "unable to login",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid JSON",
		})
		return
	}

	ctx, cancel := withTimeout(c)
	defer cancel()

	pair, err := h.Service.Refresh(ctx, req.RefreshToken)

	if err != nil {
		if errors.Is(err, ErrInvalidToken) ||
			errors.Is(err, ErrTokenReplay) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid refresh session",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "unable to refresh",
		})
		return
	}

	c.JSON(http.StatusOK, pair)
}

func (h *Handler) Logout(c *gin.Context) {
	var req RefreshRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid JSON",
		})
		return
	}

	ctx, cancel := withTimeout(c)
	defer cancel()

	err := h.Service.Logout(ctx, req.RefreshToken)

	if err != nil {
		if errors.Is(err, ErrInvalidToken) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid refresh session",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "unable to log out",
		})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	id, ok := CurrentUserID(c)

	if !ok {
		c.Status(http.StatusUnauthorized)
		return
	}

	ctx, cancel := withTimeout(c)
	defer cancel()

	user, err := h.Service.GetUser(ctx, id)

	if err != nil {
		if errors.Is(err, ErrInvalidToken) {
			c.Status(http.StatusUnauthorized)
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "unable to read profile",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

func Authenticate(tokens *AccessTokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatus(401)
			return
		}
		id, err := tokens.Validate(parts[1])
		if err != nil {
			c.AbortWithStatus(401)
			return
		}
		c.Set("userID", id)
		c.Next()
	}
}

func CurrentUserID(c *gin.Context) (uint64, bool) {
	raw, ok := c.Get("userID")
	if !ok {
		return 0, false
	}
	id, ok := raw.(uint64)
	return id, ok && id != 0
}
