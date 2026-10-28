package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

func runMigrate(args []string) error {
	if len(args) != 1 || args[0] != "up" {
		return errors.New("usage: cli migrate up")
	}

	databaseURL, err := loadDatabaseURL()
	if err != nil {
		return err
	}
	if _, err := exec.LookPath("atlas"); err != nil {
		return errors.New("Atlas CLI is required; install it from https://atlasgo.io/getting-started")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "atlas", "migrate", "apply",
		"--dir", "file://migrations",
		"--url", databaseURL,
		"--revisions-schema", "public",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return errors.New("apply migrations: timed out after 5 minutes")
		}
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
