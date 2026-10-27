package config

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL     string
	JWTSecret       string
	AppAPIKey       string
	AllowedOrigins  []string
	CookieSecure    bool
	cookieSecureRaw string
	APIAddr         string
}

type serviceAccount struct {
	Type         string `json:"type"`
	ProjectID    string `json:"project_id"`
	PrivateKeyID string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
	ClientEmail  string `json:"client_email"`
	ClientID     string `json:"client_id"`
	TokenURI     string `json:"token_uri"`
}

var variables = []string{
	"DATABASE_URL", "JWT_SECRET", "APP_API_KEY", "ALLOWED_ORIGINS",
	"COOKIE_SECURE", "API_ADDR",
}

func env(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return strings.TrimSpace(value)
	}
	return fallback
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}

func Load() (Config, error) {
	if err := loadEnvironment(); err != nil {
		return Config{}, err
	}
	secureRaw := env("COOKIE_SECURE", "false")
	secure, _ := strconv.ParseBool(secureRaw)
	return Config{
		DatabaseURL:     strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		AppAPIKey:       os.Getenv("APP_API_KEY"),
		AllowedOrigins:  splitCSV(env("ALLOWED_ORIGINS", "http://localhost:5173")),
		CookieSecure:    secure,
		cookieSecureRaw: secureRaw,
		APIAddr:         env("API_ADDR", ":8080"),
	}, nil
}

func loadEnvironment() error {
	for _, path := range []string{".env"} {
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		if err := godotenv.Load(path); err != nil {
			return fmt.Errorf("load %s: %w", path, err)
		}
		return nil
	}
	return nil
}

func (c Config) ValidateDatabase() error {
	parsed, err := url.Parse(c.DatabaseURL)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") {
		return errors.New("DATABASE_URL must be a valid postgres:// or postgresql:// URL")
	}
	if parsed.Hostname() == "" || parsed.User == nil || strings.TrimSpace(parsed.User.Username()) == "" || strings.Trim(parsed.Path, "/") == "" {
		return errors.New("DATABASE_URL must include host, username, and database name")
	}
	password, ok := parsed.User.Password()
	if !ok || password == "" {
		return errors.New("DATABASE_URL must include a password")
	}
	if port := parsed.Port(); port != "" {
		value, err := strconv.Atoi(port)
		if err != nil || value < 1 || value > 65535 {
			return errors.New("DATABASE_URL contains an invalid port")
		}
	}
	return nil
}

func (c Config) ValidateAPI() error {
	if err := c.ValidateDatabase(); err != nil {
		return err
	}
	if _, err := strconv.ParseBool(c.cookieSecureRaw); err != nil {
		return errors.New("COOKIE_SECURE must be true or false")
	}
	if len(c.JWTSecret) < 32 {
		return errors.New("JWT_SECRET must contain at least 32 bytes")
	}
	if len(c.AppAPIKey) < 32 {
		return errors.New("APP_API_KEY must contain at least 32 bytes")
	}
	_, port, err := net.SplitHostPort(c.APIAddr)
	if err != nil || strings.TrimSpace(c.APIAddr) == "" {
		return errors.New("API_ADDR must be a valid host:port address, such as :8080")
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return errors.New("API_ADDR must contain a valid port")
	}
	if len(c.AllowedOrigins) == 0 {
		return errors.New("ALLOWED_ORIGINS must contain at least one origin")
	}
	for _, origin := range c.AllowedOrigins {
		parsed, err := url.Parse(origin)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
			return fmt.Errorf("ALLOWED_ORIGINS contains an invalid origin: %q", origin)
		}
	}
	return nil
}

func (c Config) ValidateWorker() error {
	if err := c.ValidateDatabase(); err != nil {
		return err
	}
	return validateFirebaseKey(FirebaseKeyPath())
}

func FirebaseKeyPath() string {
	for _, path := range []string{"notification-service-key.json"} {
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			return path
		}
	}
	return "notification-service-key.json"
}

func validateFirebaseKey(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("Firebase service-account file not found at %q; save the downloaded key there", path)
		}
		return fmt.Errorf("check Firebase service-account file %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("Firebase service-account path %q must be a regular file", path)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read Firebase service-account file %q: %w", path, err)
	}

	var account serviceAccount
	if err := json.Unmarshal(contents, &account); err != nil {
		return fmt.Errorf("Firebase service-account file %q is not valid JSON", path)
	}
	if account.Type != "service_account" {
		return errors.New("Firebase key type must be service_account")
	}

	fields := map[string]string{
		"project_id":     account.ProjectID,
		"private_key_id": account.PrivateKeyID,
		"private_key":    account.PrivateKey,
		"client_email":   account.ClientEmail,
		"client_id":      account.ClientID,
		"token_uri":      account.TokenURI,
	}
	for name, field := range fields {
		if strings.TrimSpace(field) == "" ||
			strings.Contains(strings.ToUpper(field), "REPLACE_WITH") {
			return fmt.Errorf("Firebase service-account field %s is empty or contains a placeholder", name)
		}
	}

	if !strings.Contains(account.ClientEmail, "@") {
		return errors.New("Firebase service-account client_email is invalid")
	}

	tokenURI, err := url.Parse(account.TokenURI)
	if err != nil || tokenURI.Scheme != "https" || tokenURI.Host == "" {
		return errors.New("Firebase service-account token_uri must be a valid HTTPS URL")
	}

	block, rest := pem.Decode([]byte(account.PrivateKey))
	if block == nil || len(strings.TrimSpace(string(rest))) != 0 {
		return errors.New("Firebase service-account private_key is not valid PEM")
	}
	if block.Type != "PRIVATE KEY" && block.Type != "RSA PRIVATE KEY" {
		return errors.New("Firebase service-account private_key must be a PKCS#8 or PKCS#1 RSA private key")
	}

	var parsedKey any
	if block.Type == "RSA PRIVATE KEY" {
		parsedKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	} else {
		parsedKey, err = x509.ParsePKCS8PrivateKey(block.Bytes)
	}
	if err != nil {
		return fmt.Errorf("Firebase service-account private_key cannot be parsed: %w", err)
	}
	if _, ok := parsedKey.(*rsa.PrivateKey); !ok {
		return errors.New("Firebase service-account private_key must be an RSA private key")
	}

	return nil
}
