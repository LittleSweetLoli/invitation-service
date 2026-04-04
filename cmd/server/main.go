package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	appdb "github.com/example/invitation-service/internal/db"
	"github.com/example/invitation-service/internal/handler"
	"github.com/example/invitation-service/internal/repository"
	"github.com/example/invitation-service/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	dbURL := getenv("DATABASE_URL",
		"postgres://postgres:postgres@localhost:5432/invitations?sslmode=disable")

	// TODO: Добавить ретрай с exponential backoff чтобы сервис мог запуститься
	// до того, как PostgreSQL завершит инициализацию, внутри Docker Compose.
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		slog.Error("open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx := context.Background()
	if err = db.PingContext(ctx); err != nil {
		slog.Error("ping database", "error", err)
		os.Exit(1)
	}

	if err = appdb.RunMigrations(ctx, db); err != nil {
		slog.Error("run migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations applied successfully")

	repo := repository.NewInvitationRepository(db)
	svc := service.NewInvitationService(repo)
	h := handler.NewInvitationHandler(svc)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// TODO: Добавить middleware для ограничения частоты запросов по IP и коду приглашения.

	r.Post("/api/v1/invitations/{code}", h.UseInvitation)

	addr := ":" + getenv("PORT", "3000")
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server…")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
