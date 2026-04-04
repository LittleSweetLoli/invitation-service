package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/example/invitation-service/internal/service"
	"github.com/go-chi/chi/v5"
)

// useInvitationRequest — это тело JSON, ожидаемое POST запросом /api/v1/invitations/{code}.
type useInvitationRequest struct {
	Email string `json:"Email"`
}

// InvitationHandler содержит HTTP обработчики для эндпоинтов приглашений.
type InvitationHandler struct {
	svc service.InvitationService
}

// NewInvitationHandler создаёт обработчик, связанный с переданным сервисом.
func NewInvitationHandler(svc service.InvitationService) *InvitationHandler {
	return &InvitationHandler{svc: svc}
}

// UseInvitation обрабатывает POST /api/v1/invitations/{code}.
func (h *InvitationHandler) UseInvitation(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	var req useInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	err := h.svc.UseInvitation(r.Context(), code, req.Email)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidEmail),
			errors.Is(err, service.ErrInvalidCode),
			errors.Is(err, service.ErrInvitationNotFound),
			errors.Is(err, service.ErrInvitationExhausted),
			errors.Is(err, service.ErrAlreadyRegistered):
			// 400 для всех ожидаемых нарушений бизнес-правил — тело пустое.
			w.WriteHeader(http.StatusBadRequest)
		default:
			// Неожиданные внутренние ошибки: логируем и возвращаем 500.
			slog.Error("UseInvitation internal error", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}
