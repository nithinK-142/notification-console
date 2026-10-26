package config

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/joho/godotenv"
)

func TestEnvExampleMatchesConfigVariables(t *testing.T) {
	path := testEnvExamplePath(t)
	values, err := godotenv.Read(path)
	if err != nil {
		t.Fatalf("read test env example %q: %v", path, err)
	}

	var actual []string
	for key := range values {
		actual = append(actual, key)
	}
	expected := append([]string(nil), variables...)
	sort.Strings(actual)
	sort.Strings(expected)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("%s variables %v do not match config variables %v", path, actual, expected)
	}

	source, err := os.ReadFile("config.go")
	if err != nil {
		t.Fatal(err)
	}
	keyPattern := regexp.MustCompile(`(?:os\.(?:Getenv|LookupEnv)|env)\("([A-Z][A-Z0-9_]*)"`)
	seen := make(map[string]bool)
	for _, match := range keyPattern.FindAllStringSubmatch(string(source), -1) {
		seen[match[1]] = true
	}
	var readKeys []string
	for key := range seen {
		readKeys = append(readKeys, key)
	}
	sort.Strings(readKeys)
	if !reflect.DeepEqual(readKeys, expected) {
		t.Fatalf("config reads %v but %s declares %v", readKeys, path, expected)
	}
}

func TestLoadReadsTestEnvironmentAndValidatesAPI(t *testing.T) {
	values := loadTestEnvironment(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if err := cfg.ValidateAPI(); err != nil {
		t.Fatalf("ValidateAPI() error = %v", err)
	}
	if cfg.APIAddr != values["API_ADDR"] {
		t.Errorf("APIAddr = %q, want %q", cfg.APIAddr, values["API_ADDR"])
	}
	wantCookieSecure, err := strconv.ParseBool(values["COOKIE_SECURE"])
	if err != nil {
		t.Fatalf("invalid COOKIE_SECURE in .env.test: %v", err)
	}
	if cfg.CookieSecure != wantCookieSecure {
		t.Errorf("CookieSecure = %t, want %t", cfg.CookieSecure, wantCookieSecure)
	}
	wantOrigins := splitCSV(values["ALLOWED_ORIGINS"])
	if !reflect.DeepEqual(cfg.AllowedOrigins, wantOrigins) {
		t.Errorf("AllowedOrigins = %v, want %v", cfg.AllowedOrigins, wantOrigins)
	}
}

func TestValidateAPIRejectsShortSecret(t *testing.T) {
	cfg := validAPIConfig(t)
	cfg.JWTSecret = "too-short"
	if err := cfg.ValidateAPI(); err == nil || !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("ValidateAPI() error = %v, want JWT_SECRET validation error", err)
	}
}

func TestValidateDatabaseRejectsNonPostgresURL(t *testing.T) {
	cfg := validAPIConfig(t)
	cfg.DatabaseURL = "mysql://user:password@localhost:3306/notification_console"
	if err := cfg.ValidateDatabase(); err == nil {
		t.Fatal("ValidateDatabase() accepted a MySQL URL")
	}
}

func TestValidateAPIRejectsInvalidCookieSetting(t *testing.T) {
	cfg := validAPIConfig(t)
	cfg.cookieSecureRaw = "sometimes"
	if err := cfg.ValidateAPI(); err == nil || !strings.Contains(err.Error(), "COOKIE_SECURE") {
		t.Fatalf("ValidateAPI() error = %v, want COOKIE_SECURE validation error", err)
	}
}

func loadTestEnvironment(t *testing.T) map[string]string {
	t.Helper()

	path := testEnvPath(t)
	values, err := godotenv.Read(path)
	if err != nil {
		t.Fatalf("read test env file %q: %v", path, err)
	}
	for key, value := range values {
		t.Setenv(key, value)
	}
	for _, key := range variables {
		if strings.TrimSpace(values[key]) == "" {
			t.Fatalf("%s is missing or empty in %s", key, path)
		}
	}
	return values
}

func testEnvPath(t *testing.T) string {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller could not locate config_test.go")
	}
	return filepath.Join(filepath.Dir(sourceFile), "..", "..", ".env.test")
}

func testEnvExamplePath(t *testing.T) string {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller could not locate config_test.go")
	}
	return filepath.Join(filepath.Dir(sourceFile), "..", "..", ".env.test.example")
}

func validAPIConfig(t *testing.T) Config {
	t.Helper()
	loadTestEnvironment(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() from .env.test: %v", err)
	}
	return cfg
}
