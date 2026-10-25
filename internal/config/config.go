package config

import (
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
