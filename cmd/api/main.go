package main

import (
	"context"
	"fmt"
	"log"
	"time"
	"github.com/bikhizlit/todo-go/internal/config"
	"github.com/bikhizlit/todo-go/internal/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var databaseName string
	err = db.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&databaseName)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Connected successfully to database: %s\n", databaseName)
}
