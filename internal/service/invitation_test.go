package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/example/invitation-service/internal/repository"
	"github.com/example/invitation-service/internal/service"
)

type mockRepo struct{ err error }

func (m *mockRepo) UseInvitation(_ context.Context, _, _ string) error { return m.err }

func newSvc(repoErr error) service.InvitationService {
	return service.NewInvitationService(&mockRepo{err: repoErr})
}

func assertErrorIs(t *testing.T, err, target error) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Fatalf("expected %v, got %v", target, err)
	}
}

func TestUseInvitation_Success(t *testing.T) {
	err := newSvc(nil).UseInvitation(context.Background(), "twitter-reg1", "user@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestUseInvitation_EmptyCode(t *testing.T) {
	assertErrorIs(t,
		newSvc(nil).UseInvitation(context.Background(), "", "user@example.com"),
		service.ErrInvalidCode,
	)
}

func TestUseInvitation_WhitespaceCode(t *testing.T) {
	assertErrorIs(t,
		newSvc(nil).UseInvitation(context.Background(), "   ", "user@example.com"),
		service.ErrInvalidCode,
	)
}

func TestUseInvitation_EmptyEmail(t *testing.T) {
	assertErrorIs(t,
		newSvc(nil).UseInvitation(context.Background(), "code", ""),
		service.ErrInvalidEmail,
	)
}

func TestUseInvitation_InvalidEmails(t *testing.T) {
	cases := []string{
		"notanemail",
		"missing@",
		"@nodomain",
		"double@@domain.com",
		"no-at-sign",
	}
	svc := newSvc(nil)
	for _, email := range cases {
		email := email
		t.Run(email, func(t *testing.T) {
			assertErrorIs(t,
				svc.UseInvitation(context.Background(), "code", email),
				service.ErrInvalidEmail,
			)
		})
	}
}

func TestUseInvitation_ValidEmails(t *testing.T) {
	cases := []string{
		"user@example.com",
		"user+tag@example.org",
		"USER@EXAMPLE.COM",
		"user.name@subdomain.example.co.uk",
	}
	svc := newSvc(nil)
	for _, email := range cases {
		email := email
		t.Run(email, func(t *testing.T) {
			if err := svc.UseInvitation(context.Background(), "code", email); err != nil {
				t.Fatalf("unexpected error for valid email %q: %v", email, err)
			}
		})
	}
}

func TestUseInvitation_InvitationNotFound(t *testing.T) {
	assertErrorIs(t,
		newSvc(repository.ErrInvitationNotFound).UseInvitation(
			context.Background(), "unknown-code", "user@example.com"),
		service.ErrInvitationNotFound,
	)
}

func TestUseInvitation_Exhausted(t *testing.T) {
	assertErrorIs(t,
		newSvc(repository.ErrInvitationExhausted).UseInvitation(
			context.Background(), "code", "user@example.com"),
		service.ErrInvitationExhausted,
	)
}

func TestUseInvitation_AlreadyRegistered(t *testing.T) {
	assertErrorIs(t,
		newSvc(repository.ErrAlreadyRegistered).UseInvitation(
			context.Background(), "code", "user@example.com"),
		service.ErrAlreadyRegistered,
	)
}

func TestUseInvitation_UnexpectedRepoError_DoesNotLeakSentinel(t *testing.T) {
	err := newSvc(errors.New("unexpected db failure")).UseInvitation(
		context.Background(), "code", "user@example.com")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	for _, sentinel := range []error{
		service.ErrInvitationNotFound,
		service.ErrInvitationExhausted,
		service.ErrAlreadyRegistered,
		service.ErrInvalidCode,
		service.ErrInvalidEmail,
	} {
		if errors.Is(err, sentinel) {
			t.Fatalf("unexpected sentinel %v for unknown repo error", sentinel)
		}
	}
}
