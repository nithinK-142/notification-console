package main

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestNormalizeEmail(t *testing.T) {
	got, err := normalizeEmail("  Admin@Example.com ")
	if err != nil {
		t.Fatalf("normalizeEmail() error = %v", err)
	}
	if got != "admin@example.com" {
		t.Fatalf("normalizeEmail() = %q, want %q", got, "admin@example.com")
	}
}

func TestNormalizeEmailRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"", "admin", "Admin <admin@example.com>", "admin@example.com other", "a b@example.com"} {
		t.Run(value, func(t *testing.T) {
			if _, err := normalizeEmail(value); err == nil {
				t.Fatalf("normalizeEmail(%q) unexpectedly succeeded", value)
			}
		})
	}
}

func TestHashPassword(t *testing.T) {
	password := "a sufficiently long admin password"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == "" || hash == password {
		t.Fatal("HashPassword() returned an invalid hash")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		t.Fatalf("generated hash rejected the correct password: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("incorrect password")); err == nil {
		t.Fatal("generated hash accepted an incorrect password")
	}
}
