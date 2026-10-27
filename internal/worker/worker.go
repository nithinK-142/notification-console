package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/nithinK-142/notification-console/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	pollInterval         = 2 * time.Second
	leaseDuration        = 2 * time.Minute
	heartbeatInterval    = 30 * time.Second
	fcmCallTimeout       = 20 * time.Second
	tokenBatchSize       = 500
	maxWholeCallAttempts = 5
	maxPerTokenRetries   = 3
)

var ErrLeaseLost = errors.New("worker lease lost")

type Sender interface {
	Send(context.Context, string, string, []string) ([]SendResult, error)
}

type SendResult struct {
	Success   bool
	Invalid   bool
	Transient bool
	Error     string
}

type Worker struct {
	db     *gorm.DB
	sender Sender
	id     string
	sleep  func(context.Context, time.Duration) error
}

type tokenRow struct {
	ID    int64
	Token string
}

type pageResult struct {
	lastID     int64
	sent       int64
	failed     int64
	invalid    int64
	invalidIDs []int64
}

type leaseState struct {
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
	lost   atomic.Bool
}

func New(db *gorm.DB, sender Sender, workerID string) *Worker {
	return &Worker{db: db, sender: sender, id: workerID, sleep: sleepContext}
}

// Run polls until stopped, or drains currently claimable jobs when once is true.
func (w *Worker) Run(ctx context.Context, once bool) error {
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		job, err := w.claim(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("claim notification job: %w", err)
		}
		if job == nil {
			if once {
				return nil
			}
			if err := sleepContext(ctx, pollInterval); err != nil {
				return nil
			}
			continue
		}
		if err := w.processJob(ctx, *job); err != nil {
			if errors.Is(err, ErrLeaseLost) {
				log.Printf("worker %s abandoned notification %d: lease lost", w.id, job.NotificationID)
				continue
			}
			return fmt.Errorf("process notification %d: %w", job.NotificationID, err)
		}
	}
}

func (w *Worker) claim(ctx context.Context) (*models.NotificationJob, error) {
	var claimed models.NotificationJob
	found := false
	err := w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("(status = ? AND next_attempt_at <= clock_timestamp()) OR (status = ? AND lease_until < clock_timestamp())", "pending", "processing").
			Order("next_attempt_at ASC, notification_id ASC").First(&claimed).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		found = true
		expiredLease := claimed.Status == "processing"
		updates := map[string]any{
			"status": "processing", "lease_owner": w.id,
			"lease_until": gorm.Expr("clock_timestamp() + (? * INTERVAL '1 second')", int64(leaseDuration.Seconds())),
		}
		if claimed.StartedAt == nil {
			var highWatermark int64
			if err := tx.Model(&models.DeviceToken{}).Select("COALESCE(MAX(id), 0)").Scan(&highWatermark).Error; err != nil {
				return err
			}
			var total int64
			if err := tx.Model(&models.DeviceToken{}).Where("active = ? AND id <= ?", true, highWatermark).Count(&total).Error; err != nil {
				return err
			}
			updates["high_watermark"] = highWatermark
			updates["total"] = total
			updates["started_at"] = gorm.Expr("clock_timestamp()")
		}
		result := tx.Model(&models.NotificationJob{}).Where("notification_id = ?", claimed.NotificationID).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrLeaseLost
		}
		if err := tx.Where("notification_id = ?", claimed.NotificationID).First(&claimed).Error; err != nil {
			return err
		}
		if expiredLease {
			owner := tx.Model(&models.NotificationJob{}).Select("notification_id").
				Where("notification_id = ? AND status = ? AND lease_owner = ?", claimed.NotificationID, "processing", w.id)
			updates := map[string]any{
				"status": "failed", "error": "worker lost lease before batch commit", "completed_at": gorm.Expr("clock_timestamp()"),
			}
			if err := tx.Model(&models.NotificationBatch{}).
				Where("notification_id = ? AND status = ? AND notification_id IN (?)", claimed.NotificationID, "sending", owner).
				Updates(updates).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return &claimed, nil
}

func (w *Worker) processJob(rootCtx context.Context, job models.NotificationJob) error {
	if rootCtx.Err() != nil {
		return w.releaseLease(job.NotificationID)
	}
	state := w.startHeartbeat(job.NotificationID)
	defer func() {
		state.cancel()
		<-state.done
	}()

	var notification models.Notification
	if err := w.db.WithContext(state.ctx).First(&notification, job.NotificationID).Error; err != nil {
		return fmt.Errorf("load notification content: %w", err)
	}
	for {
		if state.lost.Load() {
			return ErrLeaseLost
		}
		if rootCtx.Err() != nil {
			return w.releaseLease(job.NotificationID)
		}
		tokens, err := w.nextTokens(state.ctx, job.NotificationID, job.Cursor, job.HighWatermark)
		if err != nil {
			if state.lost.Load() {
				return ErrLeaseLost
			}
			return fmt.Errorf("read token page: %w", err)
		}
		if rootCtx.Err() != nil {
			return w.releaseLease(job.NotificationID)
		}
		if len(tokens) == 0 {
			if err := w.completeJob(state.ctx, job.NotificationID, job.HighWatermark); err != nil {
				if state.lost.Load() {
					return ErrLeaseLost
				}
				return err
			}
			return nil
		}

		result, err := w.sendWithRetries(state.ctx, rootCtx, job.NotificationID, notification, tokens)
		if err != nil {
			if state.lost.Load() || errors.Is(err, ErrLeaseLost) {
				return ErrLeaseLost
			}
			if rootCtx.Err() != nil {
				return w.releaseLease(job.NotificationID)
			}
			return w.scheduleWholeCallRetry(state.ctx, job.NotificationID, err)
		}
		if state.lost.Load() {
			return ErrLeaseLost
		}
		if err := w.commitPage(state.ctx, job.NotificationID, result); err != nil {
			if state.lost.Load() || errors.Is(err, ErrLeaseLost) {
				return ErrLeaseLost
			}
			return fmt.Errorf("commit token page: %w", err)
		}
		job.Cursor = result.lastID
		job.Attempts = 0
	}
}

func (w *Worker) nextTokens(ctx context.Context, notificationID, cursor, highWatermark int64) ([]tokenRow, error) {
	var tokens []tokenRow
	err := w.db.WithContext(ctx).Model(&models.DeviceToken{}).
		Select("id, token").Where("active = ? AND id > ? AND id <= ?", true, cursor, highWatermark).
		Order("id ASC").Limit(tokenBatchSize).Find(&tokens).Error
	return tokens, err
}

func (w *Worker) sendWithRetries(ctx, rootCtx context.Context, notificationID int64, notification models.Notification, tokens []tokenRow) (pageResult, error) {
	remaining := append([]tokenRow(nil), tokens...)
	final := make(map[int64]SendResult, len(tokens))
	delays := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}
	for attempt := 0; len(remaining) > 0; attempt++ {
		if attempt > 0 {
			if rootCtx.Err() != nil {
				break
			}
			if err := w.sleep(ctx, delays[attempt-1]); err != nil {
				return pageResult{}, err
			}
			if rootCtx.Err() != nil {
				return pageResult{}, rootCtx.Err()
			}
		}
		batchTokens := make([]string, len(remaining))
		for i, token := range remaining {
			batchTokens[i] = token.Token
		}
		results, err := w.sendCall(ctx, notificationID, notification.Title, notification.Body, batchTokens)
		if err != nil {
			return pageResult{}, err
		}
		next := make([]tokenRow, 0, len(remaining))
		for i, result := range results {
			token := remaining[i]
			if result.Transient && attempt < maxPerTokenRetries && rootCtx.Err() == nil {
				next = append(next, token)
				continue
			}
			final[token.ID] = result
		}
		remaining = next
		if rootCtx.Err() != nil {
			for _, token := range remaining {
				final[token.ID] = SendResult{Error: "worker stopping before transient retry"}
			}
			break
		}
	}

	if len(final) != len(tokens) {
		return pageResult{}, errors.New("sender returned an incomplete result set")
	}
	result := pageResult{lastID: tokens[len(tokens)-1].ID, invalidIDs: make([]int64, 0)}
	for _, token := range tokens {
		outcome := final[token.ID]
		if outcome.Success {
			result.sent++
			continue
		}
		result.failed++
		if outcome.Invalid {
			result.invalid++
			result.invalidIDs = append(result.invalidIDs, token.ID)
		}
	}
	return result, nil
}

func (w *Worker) sendCall(ctx context.Context, notificationID int64, title, body string, tokens []string) ([]SendResult, error) {
	batchNo, err := w.createBatchCall(ctx, notificationID, len(tokens))
	if err != nil {
		return nil, err
	}
	callCtx, cancel := context.WithTimeout(ctx, fcmCallTimeout)
	results, sendErr := w.sender.Send(callCtx, title, body, tokens)
	cancel()
	if sendErr == nil && len(results) != len(tokens) {
		sendErr = fmt.Errorf("FCM returned %d results for %d tokens", len(results), len(tokens))
	}
	if err := w.finishBatchCall(ctx, notificationID, batchNo, results, sendErr, len(tokens)); err != nil {
		return nil, err
	}
	if sendErr != nil {
		return nil, sendErr
	}
	return results, nil
}

func (w *Worker) createBatchCall(ctx context.Context, notificationID int64, tokenCount int) (int64, error) {
	var batchNo int64
	err := w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := w.ensureLease(tx, notificationID); err != nil {
			return err
		}
		if err := tx.Model(&models.NotificationBatch{}).Where("notification_id = ?", notificationID).
			Select("COALESCE(MAX(batch_no), 0)").Scan(&batchNo).Error; err != nil {
			return err
		}
		batchNo++
		batch := models.NotificationBatch{
			NotificationID: notificationID, BatchNo: batchNo, TokenCount: tokenCount,
			Status: "sending",
		}
		return tx.Create(&batch).Error
	})
	return batchNo, err
}

func (w *Worker) finishBatchCall(ctx context.Context, notificationID, batchNo int64, results []SendResult, callErr error, tokenCount int) error {
	return w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := w.ensureLease(tx, notificationID); err != nil {
			return err
		}
		updates := map[string]any{"completed_at": gorm.Expr("clock_timestamp()")}
		if callErr != nil {
			updates["status"] = "failed"
			updates["failure"] = tokenCount
			updates["error"] = shortError(callErr.Error())
		} else {
			var successes, failures, invalid int
			var firstError string
			for _, result := range results {
				if result.Success {
					successes++
				} else {
					failures++
					if result.Invalid {
						invalid++
					}
					if firstError == "" && result.Error != "" {
						firstError = result.Error
					}
				}
			}
			updates["status"] = "done"
			updates["success"] = successes
			updates["failure"] = failures
			updates["invalid"] = invalid
			if firstError != "" {
				updates["error"] = shortError(firstError)
			}
		}
		result := tx.Model(&models.NotificationBatch{}).
			Where("notification_id = ? AND batch_no = ? AND status = ?", notificationID, batchNo, "sending").
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrLeaseLost
		}
		return nil
	})
}

func (w *Worker) commitPage(ctx context.Context, notificationID int64, page pageResult) error {
	return w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := w.ensureLease(tx, notificationID); err != nil {
			return err
		}
		updates := map[string]any{
			"cursor":     page.lastID,
			"sent":       gorm.Expr("sent + ?", page.sent),
			"failed":     gorm.Expr("failed + ?", page.failed),
			"invalid":    gorm.Expr("invalid + ?", page.invalid),
			"attempts":   0,
			"last_error": nil,
		}
		jobResult := tx.Model(&models.NotificationJob{}).
			Where("notification_id = ? AND status = ? AND lease_owner = ?", notificationID, "processing", w.id).
			Updates(updates)
		if jobResult.Error != nil {
			return jobResult.Error
		}
		if jobResult.RowsAffected != 1 {
			return ErrLeaseLost
		}
		if len(page.invalidIDs) > 0 {
			owner := tx.Model(&models.NotificationJob{}).Select("notification_id").
				Where("notification_id = ? AND status = ? AND lease_owner = ? AND lease_until > clock_timestamp()", notificationID, "processing", w.id)
			updateResult := tx.Model(&models.DeviceToken{}).
				Where("id IN ? AND EXISTS (?)", page.invalidIDs, owner).
				Updates(map[string]any{"active": false, "updated_at": gorm.Expr("clock_timestamp()")})
			if updateResult.Error != nil {
				return updateResult.Error
			}
		}
		return nil
	})
}

func (w *Worker) completeJob(ctx context.Context, notificationID, highWatermark int64) error {
	return w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := w.ensureLease(tx, notificationID); err != nil {
			return err
		}
		result := tx.Model(&models.NotificationJob{}).
			Where("notification_id = ? AND status = ? AND lease_owner = ? AND lease_until > clock_timestamp()", notificationID, "processing", w.id).
			Updates(map[string]any{
				"cursor": highWatermark, "status": "done", "completed_at": gorm.Expr("clock_timestamp()"),
				"lease_owner": nil, "lease_until": nil, "attempts": 0, "last_error": nil,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrLeaseLost
		}
		return nil
	})
}

func (w *Worker) scheduleWholeCallRetry(ctx context.Context, notificationID int64, cause error) error {
	return w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var job models.NotificationJob
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("notification_id = ? AND status = ? AND lease_owner = ? AND lease_until > clock_timestamp()", notificationID, "processing", w.id).
			First(&job).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLeaseLost
		}
		if err != nil {
			return err
		}

		attempts := job.Attempts + 1
		updates := map[string]any{
			"attempts": attempts, "last_error": shortError(cause.Error()),
			"lease_owner": nil, "lease_until": nil,
		}
		if attempts >= maxWholeCallAttempts {
			updates["status"] = "failed"
			updates["completed_at"] = gorm.Expr("clock_timestamp()")
			updates["next_attempt_at"] = gorm.Expr("clock_timestamp()")
		} else {
			updates["status"] = "pending"
			updates["next_attempt_at"] = gorm.Expr("clock_timestamp() + (? * INTERVAL '1 second')", int64(wholeCallBackoff(attempts).Seconds()))
		}
		result := tx.Model(&models.NotificationJob{}).
			Where("notification_id = ? AND status = ? AND lease_owner = ? AND lease_until > clock_timestamp()", notificationID, "processing", w.id).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrLeaseLost
		}
		return nil
	})
}

func (w *Worker) releaseLease(notificationID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result := w.db.WithContext(ctx).Model(&models.NotificationJob{}).
		Where("notification_id = ? AND status = ? AND lease_owner = ? AND lease_until > clock_timestamp()", notificationID, "processing", w.id).
		Updates(map[string]any{
			"status": "pending", "next_attempt_at": gorm.Expr("clock_timestamp()"), "lease_owner": nil, "lease_until": nil,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrLeaseLost
	}
	return nil
}

func (w *Worker) ensureLease(tx *gorm.DB, notificationID int64) error {
	var job models.NotificationJob
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("notification_id = ? AND status = ? AND lease_owner = ? AND lease_until > clock_timestamp()", notificationID, "processing", w.id).
		First(&job).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrLeaseLost
	}
	return err
}

func (w *Worker) startHeartbeat(notificationID int64) *leaseState {
	ctx, cancel := context.WithCancel(context.Background())
	state := &leaseState{ctx: ctx, cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(state.done)
		ticker := time.NewTicker(heartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				writeCtx, stop := context.WithTimeout(ctx, 5*time.Second)
				result := w.db.WithContext(writeCtx).Model(&models.NotificationJob{}).
					Where("notification_id = ? AND status = ? AND lease_owner = ? AND lease_until > clock_timestamp()", notificationID, "processing", w.id).
					Update("lease_until", gorm.Expr("clock_timestamp() + (? * INTERVAL '1 second')", int64(leaseDuration.Seconds())))
				stop()
				if result.Error != nil || result.RowsAffected != 1 {
					state.lost.Store(true)
					cancel()
					return
				}
			}
		}
	}()
	return state
}

func wholeCallBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt >= 7 {
		return 15 * time.Minute
	}
	return 15 * time.Second * time.Duration(1<<(attempt-1))
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func shortError(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 512 {
		return value[:512]
	}
	return value
}
