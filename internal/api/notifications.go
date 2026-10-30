package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/nithinK-142/notification-console/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type notificationListItem struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	Sent      int64     `json:"sent"`
	Total     int64     `json:"total"`
	Failed    int64     `json:"failed"`
	CreatedAt time.Time `json:"created_at"`
}

type notificationJobView struct {
	Status        string     `json:"status"`
	HighWatermark int64      `json:"high_watermark"`
	Cursor        int64      `json:"cursor"`
	Total         int64      `json:"total"`
	Sent          int64      `json:"sent"`
	Failed        int64      `json:"failed"`
	Invalid       int64      `json:"invalid"`
	Attempts      int        `json:"attempts"`
	NextAttemptAt time.Time  `json:"next_attempt_at"`
	LeaseUntil    *time.Time `json:"lease_until,omitempty"`
	LastError     *string    `json:"last_error,omitempty"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

type notificationDetails struct {
	ID        int64               `json:"id"`
	Title     string              `json:"title"`
	Body      string              `json:"body"`
	CreatedBy int64               `json:"created_by"`
	CreatedAt time.Time           `json:"created_at"`
	Job       notificationJobView `json:"job"`
}

type createNotificationRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func (s *Server) listNotifications(w http.ResponseWriter, r *http.Request) {
	limit, err := queryInt(r, "limit", 50, 1, 100)
	if err != nil {
		writeError(w, http.StatusBadRequest, "limit must be between 1 and 100")
		return
	}
	offset, err := queryInt(r, "offset", 0, 0, int(^uint(0)>>1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "offset must be a non-negative integer")
		return
	}

	sortDirection, err := parseNotificationSort(r.URL.Query().Get("sort"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	base := s.db.WithContext(r.Context()).Model(&models.Notification{}).
		Joins("JOIN notification_jobs ON notification_jobs.notification_id = notifications.id")
	base, err = filterNotifications(base, r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not list notifications")
		return
	}

	items := make([]notificationListItem, 0, limit)
	order := "notifications.created_at " + sortDirection + ", notifications.id " + sortDirection
	err = base.Select("notifications.id, notifications.title, notifications.created_at, notification_jobs.status, notification_jobs.sent, notification_jobs.total, notification_jobs.failed").
		Order(order).Limit(limit).Offset(offset).Scan(&items).Error
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list notifications")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total})
}

func (s *Server) createNotification(w http.ResponseWriter, r *http.Request) {
	var request createNotificationRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeDecodeError(w, err)
		return
	}
	request.Title = strings.TrimSpace(request.Title)
	request.Body = strings.TrimSpace(request.Body)
	if err := validateNotificationInput(request.Title, request.Body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	session, ok := r.Context().Value(adminContextKey{}).(adminSession)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	now := time.Now()
	notification := models.Notification{Title: request.Title, Body: request.Body, CreatedBy: session.ID, CreatedAt: now}
	job := models.NotificationJob{Status: "pending", NextAttemptAt: now}
	err := s.db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&notification).Error; err != nil {
			return err
		}
		job.NotificationID = notification.ID
		return tx.Create(&job).Error
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create notification")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": notification.ID, "status": job.Status})
}

func (s *Server) getNotification(w http.ResponseWriter, r *http.Request) {
	id, err := parsePositiveID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid notification ID")
		return
	}
	var notification models.Notification
	if err := s.db.WithContext(r.Context()).First(&notification, id).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		writeError(w, http.StatusNotFound, "notification not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read notification")
		return
	}
	var job models.NotificationJob
	if err := s.db.WithContext(r.Context()).Where("notification_id = ?", id).First(&job).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "notification job unavailable")
		return
	}
	writeJSON(w, http.StatusOK, notificationDetails{
		ID: notification.ID, Title: notification.Title, Body: notification.Body,
		CreatedBy: notification.CreatedBy, CreatedAt: notification.CreatedAt,
		Job: notificationJobView{
			Status: job.Status, HighWatermark: job.HighWatermark, Cursor: job.Cursor,
			Total: job.Total, Sent: job.Sent, Failed: job.Failed, Invalid: job.Invalid,
			Attempts: job.Attempts, NextAttemptAt: job.NextAttemptAt, LeaseUntil: job.LeaseUntil,
			LastError: job.LastError, StartedAt: job.StartedAt, CompletedAt: job.CompletedAt,
		},
	})
}

func (s *Server) listBatches(w http.ResponseWriter, r *http.Request) {
	id, err := parsePositiveID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid notification ID")
		return
	}
	limit, err := queryInt(r, "limit", 50, 1, 100)
	if err != nil {
		writeError(w, http.StatusBadRequest, "limit must be between 1 and 100")
		return
	}
	var before int64
	if raw := strings.TrimSpace(r.URL.Query().Get("before")); raw != "" {
		before, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || before < 1 {
			writeError(w, http.StatusBadRequest, "before must be a positive batch number")
			return
		}
	}

	var exists int64
	if err := s.db.WithContext(r.Context()).Model(&models.Notification{}).Where("id = ?", id).Count(&exists).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not read notification")
		return
	}
	if exists == 0 {
		writeError(w, http.StatusNotFound, "notification not found")
		return
	}

	query := s.db.WithContext(r.Context()).Where("notification_id = ?", id)
	if before > 0 {
		query = query.Where("batch_no < ?", before)
	}
	var batches []models.NotificationBatch
	if err := query.Order("batch_no DESC").Limit(limit + 1).Find(&batches).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not list batches")
		return
	}
	hasMore := len(batches) > limit
	if hasMore {
		batches = batches[:limit]
	}
	var nextBefore *int64
	if hasMore && len(batches) > 0 {
		value := batches[len(batches)-1].BatchNo
		nextBefore = &value
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": batches, "next_before": nextBefore})
}

func (s *Server) deleteNotification(w http.ResponseWriter, r *http.Request) {
	id, err := parsePositiveID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid notification ID")
		return
	}
	processing := false
	notFound := false
	err = s.db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		var job models.NotificationJob
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("notification_id = ?", id).First(&job).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			notFound = true
			return nil
		}
		if err != nil {
			return err
		}
		if job.Status == "processing" {
			processing = true
			return nil
		}
		result := tx.Where("id = ?", id).Delete(&models.Notification{})
		if result.Error != nil {
			return result.Error
		}
		notFound = result.RowsAffected == 0
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete notification")
		return
	}
	if notFound {
		writeError(w, http.StatusNotFound, "notification not found")
		return
	}
	if processing {
		writeError(w, http.StatusConflict, "notification is processing")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validateNotificationInput(title, body string) error {
	if utf8.RuneCountInString(title) < 1 || utf8.RuneCountInString(title) > 120 {
		return errors.New("title must contain 1 to 120 characters")
	}
	if utf8.RuneCountInString(body) < 1 || utf8.RuneCountInString(body) > 2000 {
		return errors.New("body must contain 1 to 2000 characters")
	}
	payload, err := json.Marshal(map[string]string{"title": title, "body": body})
	if err != nil || len(payload) > 3800 {
		return errors.New("title and body are too large for FCM")
	}
	return nil
}

func filterNotifications(query *gorm.DB, r *http.Request) (*gorm.DB, error) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status != "" {
		switch status {
		case "pending", "processing", "done", "failed":
			query = query.Where("notification_jobs.status = ?", status)
		default:
			return nil, errors.New("status must be pending, processing, done, or failed")
		}
	}
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	if utf8.RuneCountInString(search) > 200 {
		return nil, errors.New("q must contain at most 200 characters")
	}
	if search != "" {
		pattern := "%" + search + "%"
		query = query.Where("(notifications.title ILIKE ? OR notifications.body ILIKE ?)", pattern, pattern)
	}
	return query, nil
}

func queryInt(r *http.Request, key string, fallback, minimum, maximum int) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum {
		return 0, errors.New("invalid integer query parameter")
	}
	return value, nil
}

func parseNotificationSort(value string) (string, error) {
	sortDirection := strings.ToUpper(strings.TrimSpace(value))
	if sortDirection == "" {
		return "DESC", nil
	}
	if sortDirection != "ASC" && sortDirection != "DESC" {
		return "", errors.New("sort must be asc or desc")
	}
	return sortDirection, nil
}
