package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang-jwt/jwt/v5"
	"github.com/nithinK-142/notification-console/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	testJWTSecret  = "test-jwt-secret-that-is-long-enough-0001"
	testAdminEmail = "demo-admin@example.com"
	testAdminHash  = "$2a$12$EMc44EaEIr/.mFnQyhBWHOKhmwBdNL0LkFDsgHPRuyd2qrD1VYj1m"
)

func testHandler(t *testing.T) (http.Handler, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		JWTSecret:      testJWTSecret,
		AllowedOrigins: []string{"http://localhost:5173"},
		CookieSecure:   true,
		APIAddr:        ":8080",
	}
	return New(db, cfg), mock
}

func adminRows(active bool) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "email", "password_hash", "active", "created_at"}).
		AddRow(int64(1), testAdminEmail, testAdminHash, active, time.Now())
}

func expectAdminByEmail(mock sqlmock.Sqlmock, active bool) {
	query := regexp.QuoteMeta(`SELECT * FROM "admins" WHERE lower(email) = $1 ORDER BY "admins"."id" LIMIT $2`)
	mock.ExpectQuery(query).WithArgs(testAdminEmail, 1).WillReturnRows(adminRows(active))
}

func TestAdminSessionRequired(t *testing.T) {
	handler, mock := testHandler(t)
	request := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLoginRequiresJSONContentType(t *testing.T) {
	handler, mock := testHandler(t)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"email":"demo-admin@example.com","password":"DemoAdmin123!"}`))
	request.Header.Set("Content-Type", "text/plain")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnsupportedMediaType)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLoginRejectsOversizedBody(t *testing.T) {
	handler, mock := testHandler(t)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(strings.Repeat(" ", maxJSONBodySize+1)))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLoginAndCurrentAdmin(t *testing.T) {
	handler, mock := testHandler(t)
	expectAdminByEmail(mock, true)

	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"email":" DEMO-ADMIN@example.com ","password":"DemoAdmin123!"}`))
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("login response must not be cached")
	}
	var loginBody map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &loginBody); err != nil {
		t.Fatal(err)
	}
	if loginBody["email"] != testAdminEmail || len(loginBody) != 1 {
		t.Fatalf("unexpected login response: %#v", loginBody)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookie count = %d, want 1", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != sessionCookieName || cookie.Value == "" || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode || cookie.MaxAge <= 0 {
		t.Fatalf("unexpected session cookie: %#v", cookie)
	}

	query := regexp.QuoteMeta(`SELECT * FROM "admins" WHERE id = $1 AND active = $2 ORDER BY "admins"."id" LIMIT $3`)
	mock.ExpectQuery(query).WithArgs(int64(1), true, 1).WillReturnRows(adminRows(true))
	meRequest := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meRequest.AddCookie(cookie)
	meResponse := httptest.NewRecorder()
	handler.ServeHTTP(meResponse, meRequest)
	if meResponse.Code != http.StatusOK {
		t.Fatalf("current admin status = %d, want %d; body=%s", meResponse.Code, http.StatusOK, meResponse.Body.String())
	}
	var meBody map[string]string
	if err := json.Unmarshal(meResponse.Body.Bytes(), &meBody); err != nil {
		t.Fatal(err)
	}
	if meBody["email"] != testAdminEmail || len(meBody) != 1 {
		t.Fatalf("unexpected current admin response: %#v", meBody)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLoginRejectsInactiveAdmin(t *testing.T) {
	handler, mock := testHandler(t)
	expectAdminByEmail(mock, false)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"email":"demo-admin@example.com","password":"DemoAdmin123!"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if strings.Contains(response.Body.String(), "inactive") || response.Header().Get("Set-Cookie") != "" {
		t.Fatalf("inactive admin response leaks state or sets a session: %s", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLogoutClearsSessionCookie(t *testing.T) {
	handler, mock := testHandler(t)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != sessionCookieName || cookies[0].Value != "" || cookies[0].MaxAge >= 0 || !cookies[0].HttpOnly || !cookies[0].Secure {
		t.Fatalf("unexpected logout cookie: %#v", cookies)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminSessionRejectsWrongSubject(t *testing.T) {
	handler, mock := testHandler(t)
	now := time.Now()
	claims := sessionClaims{
		AdminID:   1,
		Subject:   "user",
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(sessionLifetime)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: signed})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCORSRejectsUnknownOrigin(t *testing.T) {
	handler, mock := testHandler(t)
	request := httptest.NewRequest(http.MethodOptions, "/api/auth/login", nil)
	request.Header.Set("Origin", "https://evil.example")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("unexpected Access-Control-Allow-Origin for unknown origin")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCORSRejectsUnknownOriginOnActualRequest(t *testing.T) {
	handler, mock := testHandler(t)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	request.Header.Set("Origin", "https://evil.example")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
	if response.Header().Get("Set-Cookie") != "" {
		t.Fatal("disallowed origin must not trigger logout")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
