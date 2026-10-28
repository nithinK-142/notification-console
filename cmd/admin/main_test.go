package main

import "testing"

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
	password := "correct horse battery staple"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == password || hash == "" {
		t.Fatal("HashPassword() returned an invalid hash")
	}
	if err := VerifyPassword(password, hash); err != nil {
		t.Fatalf("VerifyPassword() rejected the correct password: %v", err)
	}
	if err := VerifyPassword("wrong password", hash); err == nil {
		t.Fatal("VerifyPassword() accepted an incorrect password")
	}
}
