package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/example/invitation-service/internal/model"
	"github.com/lib/pq"
)

// Ошибки-сигналы, возвращаемые репозиторием.
var (
	ErrInvitationNotFound  = errors.New("invitation not found")
	ErrInvitationExhausted = errors.New("invitation has no remaining uses")
	ErrAlreadyRegistered   = errors.New("email already used this invitation")
)

// pgUniqueViolation — код ошибки PostgreSQL для нарушения уникальности.
const pgUniqueViolation = "23505"

// InvitationRepository определяет контракт для доступа к данным приглашений.
type InvitationRepository interface {
	UseInvitation(ctx context.Context, code, email string) error
}

type invitationRepository struct {
	db *sql.DB
}

// NewInvitationRepository создаёт репозиторий с переданным *sql.DB.
func NewInvitationRepository(db *sql.DB) InvitationRepository {
	return &invitationRepository{db: db}
}

// UseInvitation атомарно проверяет и записывает новое использование приглашения.
//
// Стратегия конкурентности:
//   - SELECT … FOR UPDATE блокирует строку приглашения, поэтому конкурентные запросы
//     для одного кода сериализуются на уровне БД. Безопасно для горизонтального
//     масштабирования, так как блокировка живёт внутри PostgreSQL, а не в памяти процесса.
//   - Уникальное ограничение на (invitation_id, email) — второй уровень защиты
//     от дублирования email при гонках.
func (r *invitationRepository) UseInvitation(ctx context.Context, code, email string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var inv model.Invitation
	err = tx.QueryRowContext(ctx,
		"SELECT id, code, max_uses FROM invitations WHERE code = $1 FOR UPDATE",
		code,
	).Scan(&inv.ID, &inv.Code, &inv.MaxUses)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInvitationNotFound
		}
		return fmt.Errorf("query invitation: %w", err)
	}

	var usesCount int
	if err = tx.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM invitation_uses WHERE invitation_id = $1",
		inv.ID,
	).Scan(&usesCount); err != nil {
		return fmt.Errorf("count uses: %w", err)
	}

	if usesCount >= inv.MaxUses {
		return ErrInvitationExhausted
	}

	_, err = tx.ExecContext(ctx,
		"INSERT INTO invitation_uses (invitation_id, email) VALUES ($1, $2)",
		inv.ID, email,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == pgUniqueViolation {
			return ErrAlreadyRegistered
		}
		return fmt.Errorf("insert invitation use: %w", err)
	}

	return tx.Commit()
}
