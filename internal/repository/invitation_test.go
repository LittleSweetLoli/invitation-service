package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"

	appdb "github.com/example/invitation-service/internal/db"
	"github.com/example/invitation-service/internal/repository"
	_ "github.com/lib/pq"
)

// getTestDB возвращает *sql.DB, подключённый к TEST_DATABASE_URL.
// Если переменная окружения отсутствует, тест пропускается — CI с только unit-тестами остаётся зелёным.
func getTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping integration tests")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err = db.Ping(); err != nil {
		t.Fatalf("ping db: %v", err)
	}
	if err = appdb.RunMigrations(context.Background(), db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	t.Cleanup(func() {
		db.ExecContext(context.Background(), "DELETE FROM invitation_uses")
		db.ExecContext(context.Background(), "DELETE FROM invitations")
		db.Close()
	})
	return db
}

// seedInvitation добавляет приглашение в базу данных для тестирования.
func seedInvitation(t *testing.T, db *sql.DB, code string, maxUses int) {
	t.Helper()
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO invitations (code, max_uses)
		 VALUES ($1, $2)
		 ON CONFLICT (code) DO UPDATE SET max_uses = $2`,
		code, maxUses,
	)
	if err != nil {
		t.Fatalf("seed invitation %q: %v", code, err)
	}
}

// TestRepository_Success проверяет успешное использование приглашения.
func TestRepository_Success(t *testing.T) {
	db := getTestDB(t)
	seedInvitation(t, db, "ok-code", 10)

	repo := repository.NewInvitationRepository(db)
	if err := repo.UseInvitation(context.Background(), "ok-code", "user@example.com"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// TestRepository_InvitationNotFound проверяет ошибку при несуществующем приглашении.
func TestRepository_InvitationNotFound(t *testing.T) {
	db := getTestDB(t)
	repo := repository.NewInvitationRepository(db)

	err := repo.UseInvitation(context.Background(), "does-not-exist", "user@example.com")
	if !errors.Is(err, repository.ErrInvitationNotFound) {
		t.Fatalf("expected ErrInvitationNotFound, got %v", err)
	}
}

// TestRepository_AlreadyRegistered проверяет, что один email не может использовать
// одно и то же приглашение дважды.
func TestRepository_AlreadyRegistered(t *testing.T) {
	db := getTestDB(t)
	seedInvitation(t, db, "dup-code", 10)
	repo := repository.NewInvitationRepository(db)

	if err := repo.UseInvitation(context.Background(), "dup-code", "dup@example.com"); err != nil {
		t.Fatalf("first use failed: %v", err)
	}
	if err := repo.UseInvitation(context.Background(), "dup-code", "dup@example.com"); !errors.Is(err, repository.ErrAlreadyRegistered) {
		t.Fatalf("expected ErrAlreadyRegistered, got %v", err)
	}
}

// TestRepository_InvitationExhausted проверяет, что приглашение нельзя использовать
// больше разрешённого количества раз (max_uses).
func TestRepository_InvitationExhausted(t *testing.T) {
	db := getTestDB(t)
	seedInvitation(t, db, "limited-code", 2)
	repo := repository.NewInvitationRepository(db)

	if err := repo.UseInvitation(context.Background(), "limited-code", "a@example.com"); err != nil {
		t.Fatalf("first use: %v", err)
	}
	if err := repo.UseInvitation(context.Background(), "limited-code", "b@example.com"); err != nil {
		t.Fatalf("second use: %v", err)
	}
	if err := repo.UseInvitation(context.Background(), "limited-code", "c@example.com"); !errors.Is(err, repository.ErrInvitationExhausted) {
		t.Fatalf("expected ErrInvitationExhausted, got %v", err)
	}
}

// TestRepository_ConcurrentSafety запускает N горутин на одно и то же приглашение
// и проверяет, что успешно проходит ровно MaxUses регистраций - ни больше, ни меньше.
// Это проверяет логику сериализации SELECT FOR UPDATE.
func TestRepository_ConcurrentSafety(t *testing.T) {
	db := getTestDB(t)

	const maxUses = 5
	const goroutines = 20

	seedInvitation(t, db, "race-code", maxUses)
	repo := repository.NewInvitationRepository(db)

	errs := make([]error, goroutines)
	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			// Каждая горутина использует уникальный email, поэтому AlreadyRegistered не является
			// ограничивающим фактором - только ErrInvitationExhausted должен остановить лишние.
			errs[idx] = repo.UseInvitation(
				context.Background(),
				"race-code",
				fmt.Sprintf("racer%d@example.com", idx),
			)
		}(i)
	}
	wg.Wait()

	var successes int
	for _, err := range errs {
		if err == nil {
			successes++
		}
	}
	if successes != maxUses {
		t.Fatalf("expected exactly %d successful registrations, got %d", maxUses, successes)
	}
}
