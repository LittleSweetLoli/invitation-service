package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/example/invitation-service/internal/repository"
)

// Ошибки-сигналы, возвращаемые service слоем.
var (
	ErrInvalidEmail        = errors.New("invalid email address")
	ErrInvalidCode         = errors.New("invitation code must not be empty")
	ErrInvitationNotFound  = errors.New("invitation not found")
	ErrInvitationExhausted = errors.New("invitation has no remaining uses")
	ErrAlreadyRegistered   = errors.New("email already used this invitation")
)

// InvitationService определяет контракт бизнес-логики для приглашений.
type InvitationService interface {
	UseInvitation(ctx context.Context, code, email string) error
}

type invitationService struct {
	repo repository.InvitationRepository
}

// NewInvitationService создаёт новый сервис, связанный с переданным репозиторием.
func NewInvitationService(repo repository.InvitationRepository) InvitationService {
	return &invitationService{repo: repo}
}

// UseInvitation валидирует входные данные и делегирует запись репозиторию.
func (s *invitationService) UseInvitation(ctx context.Context, code, email string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return ErrInvalidCode
	}

	email = strings.TrimSpace(email)
	if !isValidEmail(email) {
		return ErrInvalidEmail
	}

	err := s.repo.UseInvitation(ctx, code, email)
	if err != nil {
		return mapRepoError(err)
	}
	return nil
}

// isValidEmail возвращает true, если адрес может быть распаршен стандартной библиотекой.
func isValidEmail(email string) bool {
	if email == "" {
		return false
	}
	_, err := mail.ParseAddress(email)
	return err == nil
}

// mapRepoError преобразует ошибки-сигналы репозитория в ошибки сервисного слоя,
// чтобы вызывающие стороны никогда не зависели напрямую от пакета repository.
func mapRepoError(err error) error {
	switch {
	case errors.Is(err, repository.ErrInvitationNotFound):
		return ErrInvitationNotFound
	case errors.Is(err, repository.ErrInvitationExhausted):
		return ErrInvitationExhausted
	case errors.Is(err, repository.ErrAlreadyRegistered):
		return ErrAlreadyRegistered
	default:
		return fmt.Errorf("internal error: %w", err)
	}
}
