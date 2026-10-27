package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	firebase "firebase.google.com/go/v4"
	"github.com/nithinK-142/notification-console/internal/config"
	"github.com/nithinK-142/notification-console/internal/store"
	"github.com/nithinK-142/notification-console/internal/worker"
	"google.golang.org/api/option"
)

func main() {
	once := flag.Bool("once", false, "process all currently claimable jobs and exit")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fatal(err.Error())
	}
	if err := cfg.ValidateWorker(); err != nil {
		fatal(err.Error())
	}
	workerID, err := NewWorkerID()
	if err != nil {
		fatal("create worker ID: " + err.Error())
	}

	openCtx, cancelOpen := context.WithTimeout(context.Background(), 10*time.Second)
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

	firebaseCtx, cancelFirebase := context.WithTimeout(context.Background(), 10*time.Second)
	app, err := firebase.NewApp(firebaseCtx, &firebase.Config{}, option.WithCredentialsFile(config.FirebaseKeyPath()))
	if err != nil {
		cancelFirebase()
		fatal("initialize Firebase Admin SDK: " + err.Error())
	}
	messagingClient, err := app.Messaging(firebaseCtx)
	cancelFirebase()
	if err != nil {
		fatal("initialize Firebase Messaging: " + err.Error())
	}

	service := worker.New(db, worker.NewFirebaseSender(messagingClient), workerID)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log.Printf("worker %s started", workerID)
	if err := service.Run(ctx, *once); err != nil {
		fatal(err.Error())
	}
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, "worker:", message)
	os.Exit(1)
}

func NewWorkerID() (string, error) {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "worker"
	}
	var suffix [4]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%d-%s", hostname, os.Getpid(), hex.EncodeToString(suffix[:])), nil
}
