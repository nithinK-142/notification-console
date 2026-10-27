package api

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/nithinK-142/notification-console/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type tokenRequest struct {
	ExternalID string `json:"external_id"`
	Token      string `json:"token"`
	Platform   string `json:"platform"`
}

func (s *Server) registerToken(w http.ResponseWriter, r *http.Request) {
	var request tokenRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeDecodeError(w, err)
		return
	}
	request.ExternalID = strings.TrimSpace(request.ExternalID)
	request.Token = strings.TrimSpace(request.Token)
	request.Platform = strings.ToLower(strings.TrimSpace(request.Platform))
	if err := validateTokenRequest(request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	now := time.Now()
	err := s.db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		user := models.User{ExternalID: request.ExternalID}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "external_id"}},
			DoNothing: true,
		}).Create(&user).Error; err != nil {
			return err
		}
		if err := tx.Where("external_id = ?", request.ExternalID).First(&user).Error; err != nil {
			return err
		}
		deviceToken := models.DeviceToken{
			UserID: user.ID, Token: request.Token, Platform: request.Platform,
			Active: true, UpdatedAt: now,
		}
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "token"}},
			DoUpdates: clause.AssignmentColumns([]string{"user_id", "platform", "active", "updated_at"}),
		}).Create(&deviceToken).Error
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not register token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "registered"})
}

func (s *Server) deactivateToken(w http.ResponseWriter, r *http.Request) {
	var request tokenRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeDecodeError(w, err)
		return
	}
	request.ExternalID = strings.TrimSpace(request.ExternalID)
	request.Token = strings.TrimSpace(request.Token)
	if request.ExternalID == "" || utf8.RuneCountInString(request.ExternalID) > 255 || request.Token == "" || utf8.RuneCountInString(request.Token) > 4096 {
		writeError(w, http.StatusBadRequest, "external_id and token are required")
		return
	}

	result := s.db.WithContext(r.Context()).Model(&models.DeviceToken{}).
		Where("token = ? AND user_id IN (?) AND active = ?", request.Token,
			s.db.Model(&models.User{}).Select("id").Where("external_id = ?", request.ExternalID), true).
		Updates(map[string]any{"active": false, "updated_at": time.Now()})
	if result.Error != nil {
		writeError(w, http.StatusInternalServerError, "could not deactivate token")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validateTokenRequest(request tokenRequest) error {
	if request.ExternalID == "" || utf8.RuneCountInString(request.ExternalID) > 255 {
		return errors.New("external_id must contain 1 to 255 characters")
	}
	if request.Token == "" || utf8.RuneCountInString(request.Token) > 4096 {
		return errors.New("token must contain 1 to 4096 characters")
	}
	switch request.Platform {
	case "ios", "android", "web":
		return nil
	default:
		return errors.New("platform must be ios, android, or web")
	}
}
