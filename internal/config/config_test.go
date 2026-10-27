package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
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

func TestValidateFirebaseKeyRejectsPlaceholderFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "service-account.json")
	contents := `{"type":"service_account","project_id":"REPLACE_WITH_PROJECT_ID","private_key":"-----BEGIN PRIVATE KEY-----\\nREPLACE_WITH_PRIVATE_KEY\\n-----END PRIVATE KEY-----","client_email":"REPLACE_WITH_CLIENT_EMAIL"}`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateFirebaseKey(path); err == nil || !strings.Contains(err.Error(), "placeholder") {
		t.Fatalf("ValidateFirebaseKey() error = %v, want placeholder rejection", err)
	}
}

func TestValidateFirebaseKeyAcceptsValidRSAKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "service-account.json")
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	contents, err := json.Marshal(serviceAccount{
		Type:         "service_account",
		ProjectID:    "notification-console-test",
		PrivateKeyID: "test-key-id",
		PrivateKey:   string(privateKeyPEM),
		ClientEmail:  "firebase-admin@example.test",
		ClientID:     "123456789",
		TokenURI:     "https://oauth2.googleapis.com/token",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateFirebaseKey(path); err != nil {
		t.Fatalf("ValidateFirebaseKey() error = %v", err)
	}
}

func TestValidateFirebaseKeyRejectsMalformedPrivateKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "service-account.json")
	contents, err := json.Marshal(serviceAccount{
		Type:         "service_account",
		ProjectID:    "notification-console-test",
		PrivateKeyID: "test-key-id",
		PrivateKey:   "-----BEGIN PRIVATE KEY-----\\nnot-a-real-key\\n-----END PRIVATE KEY-----",
		ClientEmail:  "firebase-admin@example.test",
		ClientID:     "123456789",
		TokenURI:     "https://oauth2.googleapis.com/token",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateFirebaseKey(path); err == nil || !strings.Contains(err.Error(), "private_key") {
		t.Fatalf("ValidateFirebaseKey() error = %v, want private key parsing error", err)
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
