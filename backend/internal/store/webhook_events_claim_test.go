package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Revati-Firke/gitactionflow/backend/internal/database"
	"github.com/Revati-Firke/gitactionflow/backend/internal/store"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://gitactionflow:gitactionflow@localhost:5432/gitactionflow?sslmode=disable"
	}
	ctx := context.Background()
	if err := database.MigrateUp(dsn); err != nil {
		t.Skipf("migrate: %v", err)
	}
	pool, err := database.NewPool(ctx, dsn)
	if err != nil {
		t.Skipf("database unavailable: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

func seedRepo(t *testing.T, pool *pgxpool.Pool) (userID, repoID uuid.UUID, githubID int64) {
	t.Helper()
	ctx := context.Background()
	userID = uuid.New()
	githubID = time.Now().UnixNano()%1_000_000_000 + 1
	_, err := pool.Exec(ctx, `
INSERT INTO users (id, github_user_id, github_username, github_access_token_encrypted)
VALUES ($1, $2, $3, $4)
`, userID, githubID, "test-"+userID.String()[:8], []byte("x"))
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	repos := store.NewRepositories(pool)
	r, err := repos.Insert(ctx, userID, githubID, "n", "o/n", "o", "main", "https://example.com", false)
	if err != nil {
		t.Fatalf("seed repo: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})
	return userID, r.ID, githubID
}

func clearClaimable(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
UPDATE webhook_events
SET status = 'processed', processed_at = COALESCE(processed_at, NOW()), locked_at = NULL, next_retry_at = NULL, updated_at = NOW()
WHERE status IN ('pending', 'processing')
`)
	if err != nil {
		t.Fatalf("clear claimable: %v", err)
	}
}

func TestClaimNext_AndProcessed(t *testing.T) {
	pool := testPool(t)
	clearClaimable(t, pool)
	_, repoID, ghID := seedRepo(t, pool)
	events := store.NewWebhookEvents(pool)
	ctx := context.Background()

	payload, _ := json.Marshal(map[string]any{"action": "opened", "repository": map[string]any{"id": ghID}})
	e, err := events.InsertPendingWithMaxRetries(ctx, repoID, uuid.NewString(), "issues", "opened", payload, 3)
	if err != nil {
		t.Fatal(err)
	}

	claimed, err := events.ClaimNext(ctx, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if claimed.ID != e.ID || claimed.Status != store.WebhookStatusProcessing || claimed.LockedAt == nil {
		t.Fatalf("%+v", claimed)
	}

	done, err := events.MarkProcessed(ctx, claimed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != store.WebhookStatusProcessed || done.ProcessedAt == nil {
		t.Fatalf("%+v", done)
	}
}

func TestScheduleRetry_AndExhaustion(t *testing.T) {
	pool := testPool(t)
	clearClaimable(t, pool)
	_, repoID, ghID := seedRepo(t, pool)
	events := store.NewWebhookEvents(pool)
	ctx := context.Background()
	payload, _ := json.Marshal(map[string]any{"action": "opened", "repository": map[string]any{"id": ghID}})
	e, err := events.InsertPendingWithMaxRetries(ctx, repoID, uuid.NewString(), "issues", "opened", payload, 2)
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := events.ClaimNext(ctx, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if claimed.ID != e.ID {
		t.Fatalf("claimed %s want %s", claimed.ID, e.ID)
	}
	retried, err := events.ScheduleRetry(ctx, claimed.ID, "boom", time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if retried.Status != store.WebhookStatusPending || retried.RetryCount != 1 || retried.LastError == nil || retried.NextRetryAt == nil {
		t.Fatalf("%+v", retried)
	}

	time.Sleep(5 * time.Millisecond)
	claimed2, err := events.ClaimNext(ctx, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	failed, err := events.ScheduleRetry(ctx, claimed2.ID, "boom2", time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != store.WebhookStatusFailed || failed.RetryCount != 2 || failed.FailedAt == nil {
		t.Fatalf("%+v", failed)
	}
}

func TestClaimNext_Concurrent(t *testing.T) {
	pool := testPool(t)
	clearClaimable(t, pool)
	_, repoID, ghID := seedRepo(t, pool)
	events := store.NewWebhookEvents(pool)
	ctx := context.Background()
	payload, _ := json.Marshal(map[string]any{"action": "opened", "repository": map[string]any{"id": ghID}})
	e, err := events.InsertPendingWithMaxRetries(ctx, repoID, uuid.NewString(), "issues", "opened", payload, 3)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	results := make(chan error, 2)
	ids := make(chan uuid.UUID, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			claimed, err := events.ClaimNext(ctx, time.Minute)
			results <- err
			if err == nil {
				ids <- claimed.ID
			}
		}()
	}
	wg.Wait()
	close(results)
	close(ids)

	ok, miss := 0, 0
	for err := range results {
		if err == nil {
			ok++
		} else if errors.Is(err, store.ErrNotFound) {
			miss++
		} else {
			t.Fatalf("%v", err)
		}
	}
	if ok != 1 || miss != 1 {
		t.Fatalf("ok=%d miss=%d", ok, miss)
	}
	id := <-ids
	if id != e.ID {
		t.Fatalf("claimed %s want %s", id, e.ID)
	}
}

func TestClaimNext_StaleProcessing(t *testing.T) {
	pool := testPool(t)
	clearClaimable(t, pool)
	_, repoID, ghID := seedRepo(t, pool)
	events := store.NewWebhookEvents(pool)
	ctx := context.Background()
	payload, _ := json.Marshal(map[string]any{"action": "opened", "repository": map[string]any{"id": ghID}})
	e, err := events.InsertPendingWithMaxRetries(ctx, repoID, uuid.NewString(), "issues", "opened", payload, 3)
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := events.ClaimNext(ctx, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `UPDATE webhook_events SET locked_at = NOW() - INTERVAL '10 minutes' WHERE id = $1`, claimed.ID)
	if err != nil {
		t.Fatal(err)
	}

	reclaimed, err := events.ClaimNext(ctx, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if reclaimed.ID != e.ID || reclaimed.RetryCount != 1 {
		t.Fatalf("%+v", reclaimed)
	}
	if reclaimed.LastError == nil || *reclaimed.LastError == "" {
		t.Fatal("expected last_error on reclaim")
	}
}
