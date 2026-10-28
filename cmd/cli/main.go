package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/nithinK-142/notification-console/internal/config"
	"github.com/nithinK-142/notification-console/internal/store"
	"gorm.io/gorm"
)

const cliUsage = `Usage:
  cli admin create --email admin@example.com
  cli admin set-active --email admin@example.com --active=true|false
  cli migrate up
`

func main() {
	if err := runCLI(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "cli:", err)
		os.Exit(1)
	}
}

func runCLI(args []string) error {
	if len(args) == 0 {
		return errors.New(cliUsage)
	}

	switch args[0] {
	case "admin":
		return runAdmin(args[1:])
	case "migrate":
		return runMigrate(args[1:])
	case "help", "-h", "--help":
		fmt.Print(cliUsage)
		return nil
	default:
		return fmt.Errorf("unknown command %q\n%s", args[0], cliUsage)
	}
}

func loadDatabaseURL() (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}
	if err := cfg.ValidateDatabase(); err != nil {
		return "", err
	}
	return cfg.DatabaseURL, nil
}

func openDatabase() (*gorm.DB, func(), error) {
	databaseURL, err := loadDatabaseURL()
	if err != nil {
		return nil, nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := store.OpenPostgres(ctx, databaseURL)
	if err != nil {
		return nil, nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("get PostgreSQL connection pool: %w", err)
	}
	return db, func() { _ = sqlDB.Close() }, nil
}
