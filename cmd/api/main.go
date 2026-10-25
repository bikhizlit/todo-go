package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/bikhizlit/todo-go/internal/auth"
	"github.com/bikhizlit/todo-go/internal/config"
	"github.com/bikhizlit/todo-go/internal/database"
	"github.com/bikhizlit/todo-go/internal/tasks"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	startupCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)

	db, err := database.Connect(startupCtx, cfg)
	cancel()

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			2*time.Second,
		)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":   "unhealthy",
				"database": "disconnected",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   "ok",
			"database": "connected",
		})
	})

	// Connect repository -> service -> controller -> routes.
	taskRepo := tasks.NewTaskRepository(db)
	taskService := tasks.NewTaskService(taskRepo)
	taskController := tasks.NewTaskController(taskService)

	tasks.RegisterTaskRoutes(router, taskController)
	tokenManager, err := auth.NewTokenManager(
		[]byte(cfg.JWTSecret),
		cfg.JWTIssuer,
		cfg.JWTAudience,
		15*time.Minute,
	)
	if err != nil {
		log.Fatalf("initialize JWT token manager: %v", err)
	}
	// Auth module
	authRepo := auth.NewUserRepository(db)
	authService := auth.NewService(authRepo, tokenManager)
	authHandler := auth.NewHandler(authService)

	auth.RegisterUserRoutes(router, authHandler, tokenManager)

	server := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("API listening on http://localhost:%s", cfg.AppPort)
		serverErrors <- server.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server: %v", err)
		}
	case <-stop:
		log.Println("Shutting down server...")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Graceful shutdown failed: %v", err)

		if closeErr := server.Close(); closeErr != nil {
			log.Printf("Close server: %v", closeErr)
		}
	}
}
