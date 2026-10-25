package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/nithinK-142/notification-console/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fatal(err.Error())
	}
	if err := cfg.ValidateDatabase(); err != nil {
		fatal(err.Error())
	}
	if _, err := exec.LookPath("atlas"); err != nil {
		fatal("Atlas CLI is required; install it from https://atlasgo.io/getting-started")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "atlas", "migrate", "apply",
		"--dir", "file://migrations",
		"--url", cfg.DatabaseURL,
		"--revisions-schema", "public",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fatal(fmt.Sprintf("apply migrations: %v", err))
	}
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, "migrate:", message)
	os.Exit(1)
}
