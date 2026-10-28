package main

import (
	"strings"
	"testing"
)

func TestRunCLIRequiresCommand(t *testing.T) {
	err := runCLI(nil)
	if err == nil || !strings.Contains(err.Error(), "Usage:") {
		t.Fatalf("runCLI(nil) error = %v, want usage error", err)
	}
}

func TestRunCLIRejectsUnknownCommand(t *testing.T) {
	err := runCLI([]string{"unknown"})
	if err == nil || !strings.Contains(err.Error(), `unknown command "unknown"`) {
		t.Fatalf("runCLI(unknown) error = %v, want unknown-command error", err)
	}
}

func TestRunAdminRequiresSubcommand(t *testing.T) {
	err := runAdmin(nil)
	if err == nil || !strings.Contains(err.Error(), "admin <create|set-active>") {
		t.Fatalf("runAdmin(nil) error = %v, want usage error", err)
	}
}

func TestRunMigrateRequiresUp(t *testing.T) {
	err := runMigrate(nil)
	if err == nil || !strings.Contains(err.Error(), "cli migrate up") {
		t.Fatalf("runMigrate(nil) error = %v, want usage error", err)
	}
}
