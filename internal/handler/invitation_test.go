package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/invitation-service/internal/handler"
	"github.com/example/invitation-service/internal/service"
	"github.com/go-chi/chi/v5"
)

// mockService удовлетворяет интерфейсу service.InvitationService.
type mockService struct{ err error }

func (m *mockService) UseInvitation(_ context.Context, _, _ string) error { return m.err }

// executeRequest связывает chi роутер с хэндлером и выполняет запрос
func executeRequest(t *testing.T, svcErr error, code, body string) *httptest.ResponseRecorder {
	t.Helper()

	h := handler.NewInvitationHandler(&mockService{err: svcErr})

	r := chi.NewRouter()
	r.Post("/api/v1/invitations/{code}", h.UseInvitation)

	req, err := http.NewRequest(http.MethodPost,
		"/api/v1/invitations/"+code,
		bytes.NewBufferString(body),
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func jsonBody(email string) string {
	b, _ := json.Marshal(map[string]string{"Email": email})
	return string(b)
}

func assertStatus(t *testing.T, rr *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rr.Code != want {
		t.Fatalf("expected HTTP %d, got %d", want, rr.Code)
	}
}

func TestHandler_Success_Returns200(t *testing.T) {
	rr := executeRequest(t, nil, "twitter-reg1", jsonBody("user@example.com"))
	assertStatus(t, rr, http.StatusOK)
}

func TestHandler_InvalidEmail_Returns400(t *testing.T) {
	rr := executeRequest(t, service.ErrInvalidEmail, "code", jsonBody("bad-email"))
	assertStatus(t, rr, http.StatusBadRequest)
}

func TestHandler_InvalidCode_Returns400(t *testing.T) {
	rr := executeRequest(t, service.ErrInvalidCode, "%20%20", jsonBody("u@x.com"))
	assertStatus(t, rr, http.StatusBadRequest)
}

func TestHandler_InvitationNotFound_Returns400(t *testing.T) {
	rr := executeRequest(t, service.ErrInvitationNotFound, "ghost-code", jsonBody("u@x.com"))
	assertStatus(t, rr, http.StatusBadRequest)
}

func TestHandler_InvitationExhausted_Returns400(t *testing.T) {
	rr := executeRequest(t, service.ErrInvitationExhausted, "code", jsonBody("u@x.com"))
	assertStatus(t, rr, http.StatusBadRequest)
}

func TestHandler_AlreadyRegistered_Returns400(t *testing.T) {
	rr := executeRequest(t, service.ErrAlreadyRegistered, "code", jsonBody("u@x.com"))
	assertStatus(t, rr, http.StatusBadRequest)
}

func TestHandler_MalformedJSON_Returns400(t *testing.T) {
	rr := executeRequest(t, nil, "code", `{not valid json`)
	assertStatus(t, rr, http.StatusBadRequest)
}

func TestHandler_EmptyBody_ServiceSeesEmptyEmail_Returns400(t *testing.T) {
	rr := executeRequest(t, service.ErrInvalidEmail, "code", `{}`)
	assertStatus(t, rr, http.StatusBadRequest)
}

func TestHandler_InternalError_Returns500(t *testing.T) {
	rr := executeRequest(t, errors.New("unexpected db failure"), "code", jsonBody("u@x.com"))
	assertStatus(t, rr, http.StatusInternalServerError)
}

func TestHandler_NoBodyOnSuccess(t *testing.T) {
	rr := executeRequest(t, nil, "code", jsonBody("u@x.com"))
	if rr.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %q", rr.Body.String())
	}
}
