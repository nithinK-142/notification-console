package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/nithinK-142/notification-console/internal/config"
	"github.com/nithinK-142/notification-console/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const maxJSONBodySize = 64 << 10

var errUnsupportedContentType = errors.New("Content-Type must be application/json")

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return errUnsupportedContentType
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodySize)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}

	var trailing any
	err = decoder.Decode(&trailing)
	if err == nil {
		return errors.New("request must contain one JSON value")
	}
	if !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if status != http.StatusNoContent {
		_ = json.NewEncoder(w).Encode(value)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeDecodeError(w http.ResponseWriter, err error) {
	if errors.Is(err, errUnsupportedContentType) {
		writeError(w, http.StatusUnsupportedMediaType, err.Error())
		return
	}
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
		return
	}
	writeError(w, http.StatusBadRequest, "invalid JSON request")
}

const (
	sessionCookieName = "notification_console_session"
	sessionLifetime   = 8 * time.Hour
)

type sessionClaims struct {
	AdminID int64 `json:"admin_id"`
	jwt.RegisteredClaims
}

type adminContextKey struct{}

type adminSession struct {
	ID    int64
	Email string
}

type Server struct {
	db        *gorm.DB
	cfg       config.Config
	origins   map[string]struct{}
	dummyHash []byte
}

func New(db *gorm.DB, cfg config.Config) http.Handler {
	dummyHash, _ := bcrypt.GenerateFromPassword([]byte("notification-console-dummy-password"), bcrypt.DefaultCost)
	s := &Server{
		db:        db,
		cfg:       cfg,
		origins:   make(map[string]struct{}, len(cfg.AllowedOrigins)),
		dummyHash: dummyHash,
	}
	for _, origin := range cfg.AllowedOrigins {
		s.origins[origin] = struct{}{}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.HandleFunc("POST /api/auth/logout", s.logout)
	mux.Handle("GET /api/auth/me", s.requireAdmin(http.HandlerFunc(s.me)))
	mux.Handle("GET /api/notifications", s.requireAdmin(http.HandlerFunc(s.listNotifications)))
	mux.Handle("POST /api/notifications", s.requireAdmin(http.HandlerFunc(s.createNotification)))
	mux.Handle("GET /api/notifications/{id}", s.requireAdmin(http.HandlerFunc(s.getNotification)))
	mux.Handle("GET /api/notifications/{id}/batches", s.requireAdmin(http.HandlerFunc(s.listBatches)))
	mux.Handle("DELETE /api/notifications/{id}", s.requireAdmin(http.HandlerFunc(s.deleteNotification)))
	return s.cors(mux)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	sqlDB, err := s.db.DB()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	session, ok := r.Context().Value(adminContextKey{}).(adminSession)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"email": session.Email})
}

func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		claims := new(sessionClaims)
		token, err := jwt.ParseWithClaims(cookie.Value, claims, func(_ *jwt.Token) (any, error) {
			return []byte(s.cfg.JWTSecret), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
		if err != nil || !token.Valid || claims.AdminID < 1 || claims.Subject != "admin" || claims.IssuedAt == nil {
			s.clearSessionCookie(w)
			writeError(w, http.StatusUnauthorized, "session expired or invalid")
			return
		}

		var admin models.Admin
		err = s.db.WithContext(r.Context()).Where("id = ? AND active = ?", claims.AdminID, true).First(&admin).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.clearSessionCookie(w)
			writeError(w, http.StatusUnauthorized, "session expired or invalid")
			return
		}
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "authentication service unavailable")
			return
		}

		ctx := context.WithValue(r.Context(), adminContextKey{}, adminSession{ID: admin.ID, Email: admin.Email})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		_, allowed := s.origins[origin]
		if origin != "" && !allowed {
			writeError(w, http.StatusForbidden, "origin not allowed")
			return
		}
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Max-Age", "600")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) setSessionCookie(w http.ResponseWriter, value string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     "/",
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure,
		SameSite: s.sessionSameSite(),
	})
}

func (s *Server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure,
		SameSite: s.sessionSameSite(),
	})
}

func (s *Server) sessionSameSite() http.SameSite {
	if s.cfg.CookieSecure {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}

func parsePositiveID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("invalid ID")
	}
	return id, nil
}
