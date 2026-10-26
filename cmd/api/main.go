package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nithinK-142/notification-console/internal/api"
	"github.com/nithinK-142/notification-console/internal/config"
	"github.com/nithinK-142/notification-console/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fatal(err.Error())
	}
	if err := cfg.ValidateAPI(); err != nil {
		fatal(err.Error())
	}

	openCtx, cancelOpen := context.WithTimeout(context.Background(), 5*time.Second)
	db, err := store.OpenPostgres(openCtx, cfg.DatabaseURL)
	cancelOpen()
	if err != nil {
		fatal(err.Error())
	}
	sqlDB, err := db.DB()
	if err != nil {
		fatal("get PostgreSQL connection pool: " + err.Error())
	}
	defer sqlDB.Close()

	server := &http.Server{
		Addr:              cfg.APIAddr,
		Handler:           api.New(db, cfg),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serverErr := make(chan error, 1)
	go func() {
		log.Printf("API listening on %s", cfg.APIAddr)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			fatal("HTTP server: " + err.Error())
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			fatal("HTTP shutdown: " + err.Error())
		}
	}
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, "api:", message)
	os.Exit(1)
}
