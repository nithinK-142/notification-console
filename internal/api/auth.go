package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/nithinK-142/notification-console/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var request loginRequest
	if err := decodeJSON(w, r, &request); err != nil {
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
		return
	}
	request.Email = strings.ToLower(strings.TrimSpace(request.Email))
	if request.Email == "" || len(request.Email) > 320 || request.Password == "" || len(request.Password) > 1024 {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	var admin models.Admin
	err := s.db.WithContext(r.Context()).Where("lower(email) = ?", request.Email).First(&admin).Error
	passwordHash := s.dummyHash
	active := false
	if err == nil {
		passwordHash = []byte(admin.PasswordHash)
		active = admin.Active
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(request.Password))
		writeError(w, http.StatusServiceUnavailable, "authentication service unavailable")
		return
	}

	passwordErr := bcrypt.CompareHashAndPassword(passwordHash, []byte(request.Password))
	if passwordErr != nil || err != nil || !active {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	now := time.Now()
	expires := now.Add(sessionLifetime)
	claims := sessionClaims{
		AdminID:   admin.ID,
		Subject:   "admin",
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expires),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, signErr := token.SignedString([]byte(s.cfg.JWTSecret))
	if signErr != nil {
		writeError(w, http.StatusInternalServerError, "could not create session")
		return
	}

	s.setSessionCookie(w, signed, expires)
	writeJSON(w, http.StatusOK, map[string]string{"email": admin.Email})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	s.clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "signed out"})
}
