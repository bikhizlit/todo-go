package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	mysql "github.com/go-sql-driver/mysql"

	"github.com/bikhizlit/todo-go/internal/config"
)

func Connect(ctx context.Context, cfg config.Config) (*sql.DB, error) {
	driverConfig := mysql.NewConfig()
	driverConfig.User = cfg.DBUser
	driverConfig.Passwd = cfg.DBPassword
	driverConfig.Net = "tcp"
	driverConfig.Addr = cfg.DBHost + ":" + cfg.DBPort
	driverConfig.DBName = cfg.DBName
	driverConfig.ParseTime = true
	driverConfig.Params = map[string]string{
		"charset": "utf8mb4",
	}

	db, err := sql.Open("mysql", driverConfig.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("Open MYSQL connection: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(3 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to MYSQL: %w", err)
	}
	return db, nil
}
