package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

//go:embed *.sql
var migrationsFS embed.FS

// RunMigrations исполняет каждый *.sql файл в папке миграций
// Функция идемпотентна потому, что каждая миграция использует IF NOT EXISTS / ON CONFLICT guards.
// TODO: Для продакшена можно заменить на golang-migrate, для откатов, проверки checksums,
// и отслеживания примененных миграций
func RunMigrations(ctx context.Context, db *sql.DB) error {
	entries, err := fs.ReadDir(migrationsFS, ".")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		data, err := migrationsFS.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if _, err = db.ExecContext(ctx, string(data)); err != nil {
			return fmt.Errorf("exec migration %s: %w", name, err)
		}
	}
	return nil
}
