package worker

import (
	"context"
	"errors"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/nithinK-142/notification-console/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestWholeCallBackoff(t *testing.T) {
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{attempt: -1, want: 15 * time.Second},
		{attempt: 1, want: 15 * time.Second},
		{attempt: 2, want: 30 * time.Second},
		{attempt: 3, want: time.Minute},
		{attempt: 4, want: 2 * time.Minute},
		{attempt: 5, want: 4 * time.Minute},
		{attempt: 6, want: 8 * time.Minute},
		{attempt: 7, want: 15 * time.Minute},
		{attempt: 20, want: 15 * time.Minute},
	}
	for _, test := range tests {
		t.Run(strconv.Itoa(test.attempt), func(t *testing.T) {
			if got := wholeCallBackoff(test.attempt); got != test.want {
				t.Fatalf("wholeCallBackoff(%d) = %s, want %s", test.attempt, got, test.want)
			}
		})
	}
}

type scriptedSender struct {
	mu        sync.Mutex
	responses [][]SendResult
	calls     int
}

func (s *scriptedSender) Send(_ context.Context, _, _ string, tokens []string) ([]SendResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.calls >= len(s.responses) {
		return nil, errors.New("unexpected sender call")
	}
	response := s.responses[s.calls]
	s.calls++
	if len(response) != len(tokens) {
		return nil, errors.New("scripted response count does not match token count")
	}
	return append([]SendResult(nil), response...), nil
}

func (s *scriptedSender) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func TestTransientTokenErrorRetriesThenSucceeds(t *testing.T) {
	db := openIntegrationDB(t)
	jobID := createWorkerJob(t, db, models.NotificationJob{
		Status: "processing", LeaseOwner: stringPointer("retry-worker"), LeaseUntil: timePointer(time.Now().Add(time.Minute)),
	})
	tokens := []tokenRow{{ID: 1, Token: "fake-fcm-token"}}
	sender := &scriptedSender{responses: [][]SendResult{
		{{Transient: true, Error: "temporarily unavailable"}},
		{{Success: true}},
	}}
	service := New(db, sender, "retry-worker")
	service.sleep = func(context.Context, time.Duration) error { return nil }

	result, err := service.sendWithRetries(context.Background(), context.Background(), jobID, models.Notification{Title: "test", Body: "body"}, tokens)
	if err != nil {
		t.Fatalf("sendWithRetries() error = %v", err)
	}
	if result.sent != 1 || result.failed != 0 || sender.callCount() != 2 {
		t.Fatalf("result = %+v, calls = %d; want one success after two calls", result, sender.callCount())
	}
	var batchCount int64
	if err := db.Model(&models.NotificationBatch{}).Where("notification_id = ?", jobID).Count(&batchCount).Error; err != nil {
		t.Fatal(err)
	}
	if batchCount != 2 {
		t.Fatalf("batch count = %d, want 2 FCM call records", batchCount)
	}
}

func TestTwoWorkersCannotClaimTheSameJob(t *testing.T) {
	db := openIntegrationDB(t)
	jobID := createWorkerJob(t, db, models.NotificationJob{Status: "pending", NextAttemptAt: time.Now().Add(-time.Minute)})

	workers := []*Worker{New(db, nil, "worker-a"), New(db, nil, "worker-b")}
	start := make(chan struct{})
	type claimResult struct {
		job *models.NotificationJob
		err error
	}
	results := make(chan claimResult, len(workers))
	for _, service := range workers {
		go func(service *Worker) {
			<-start
			job, err := service.claim(context.Background())
			results <- claimResult{job: job, err: err}
		}(service)
	}
	close(start)

	claims := 0
	for range workers {
		result := <-results
		if result.err != nil {
			t.Fatalf("claim() error = %v", result.err)
		}
		if result.job != nil {
			claims++
			if result.job.NotificationID != jobID {
				t.Fatalf("claimed notification %d, want %d", result.job.NotificationID, jobID)
			}
		}
	}
	if claims != 1 {
		t.Fatalf("successful claims = %d, want exactly one", claims)
	}
}

func TestExpiredLeaseIsTakenOverAndSendingBatchIsFailed(t *testing.T) {
	db := openIntegrationDB(t)
	cursor := int64(500)
	jobID := createWorkerJob(t, db, models.NotificationJob{
		Status: "processing", Cursor: cursor, HighWatermark: 1500, Total: 1500,
		LeaseOwner: stringPointer("dead-worker"), LeaseUntil: timePointer(time.Now().Add(-time.Minute)),
		StartedAt: timePointer(time.Now().Add(-10 * time.Minute)),
	})
	batch := models.NotificationBatch{
		NotificationID: jobID, BatchNo: 1, TokenCount: 500,
		Status: "sending", StartedAt: time.Now().Add(-time.Minute),
	}
	if err := db.Create(&batch).Error; err != nil {
		t.Fatal(err)
	}

	service := New(db, nil, "replacement-worker")
	claimed, err := service.claim(context.Background())
	if err != nil {
		t.Fatalf("claim() error = %v", err)
	}
	if claimed == nil || claimed.NotificationID != jobID || claimed.LeaseOwner == nil || *claimed.LeaseOwner != "replacement-worker" {
		t.Fatalf("claim result = %+v, want reclaimed job owned by replacement-worker", claimed)
	}
	if claimed.Cursor != cursor {
		t.Fatalf("cursor = %d, want unchanged cursor %d", claimed.Cursor, cursor)
	}
	var actual models.NotificationBatch
	if err := db.Where("notification_id = ? AND batch_no = ?", jobID, 1).First(&actual).Error; err != nil {
		t.Fatal(err)
	}
	if actual.Status != "failed" || actual.Error == nil || *actual.Error != "worker lost lease before batch commit" {
		t.Fatalf("recovered batch = %+v, want failed recovery record", actual)
	}
}

func Test150kTokensStreamInPagesOfAtMost500(t *testing.T) {
	db := openIntegrationDB(t)
	user := models.User{ExternalID: integrationID("stream-user")}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Delete(&models.User{}, user.ID).Error })

	const tokenTotal = 150_000
	const insertBatch = 1000
	tokenPrefix := integrationID("token")
	for start := 0; start < tokenTotal; start += insertBatch {
		end := start + insertBatch
		if end > tokenTotal {
			end = tokenTotal
		}
		rows := make([]models.DeviceToken, 0, end-start)
		for i := start; i < end; i++ {
			rows = append(rows, models.DeviceToken{
				UserID: user.ID, Token: tokenPrefix + "-" + strconv.Itoa(i),
				Platform: "android", Active: true,
			})
		}
		if err := db.CreateInBatches(&rows, insertBatch).Error; err != nil {
			t.Fatalf("insert token rows %d-%d: %v", start, end, err)
		}
	}

	var highWatermark int64
	if err := db.Model(&models.DeviceToken{}).Select("MAX(id)").Scan(&highWatermark).Error; err != nil {
		t.Fatal(err)
	}
	service := New(db, nil, "stream-worker")
	var cursor int64
	streamed, pages := 0, 0
	for {
		page, err := service.nextTokens(context.Background(), 0, cursor, highWatermark)
		if err != nil {
			t.Fatalf("nextTokens() error = %v", err)
		}
		if len(page) == 0 {
			break
		}
		if len(page) > tokenBatchSize {
			t.Fatalf("page contains %d tokens, exceeds %d", len(page), tokenBatchSize)
		}
		streamed += len(page)
		pages++
		cursor = page[len(page)-1].ID
	}
	if streamed != tokenTotal {
		t.Fatalf("streamed %d tokens, want %d; ensure TEST_DATABASE_URL points to an isolated database", streamed, tokenTotal)
	}
	if pages != tokenTotal/tokenBatchSize {
		t.Fatalf("pages = %d, want %d", pages, tokenTotal/tokenBatchSize)
	}
}

func openIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL unset; PostgreSQL integration test not run")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open TEST_DATABASE_URL: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get TEST_DATABASE_URL pool: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("ping TEST_DATABASE_URL: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func createWorkerJob(t *testing.T, db *gorm.DB, job models.NotificationJob) int64 {
	t.Helper()
	admin := models.Admin{Email: integrationID("admin") + "@example.test", PasswordHash: "test-only", Active: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create integration admin: %v", err)
	}
	t.Cleanup(func() { _ = db.Delete(&models.Admin{}, admin.ID).Error })
	notification := models.Notification{Title: "integration test", Body: "body", CreatedBy: admin.ID}
	if err := db.Create(&notification).Error; err != nil {
		t.Fatalf("create integration notification: %v", err)
	}
	t.Cleanup(func() { _ = db.Delete(&models.Notification{}, notification.ID).Error })
	job.NotificationID = notification.ID
	if job.NextAttemptAt.IsZero() {
		job.NextAttemptAt = time.Now().Add(-time.Minute)
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create integration job: %v", err)
	}
	return notification.ID
}

func integrationID(prefix string) string {
	return prefix + "-" + time.Now().Format("20060102150405.000000000")
}

func stringPointer(value string) *string { return &value }

func timePointer(value time.Time) *time.Time { return &value }
