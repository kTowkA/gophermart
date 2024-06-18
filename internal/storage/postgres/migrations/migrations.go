// лучше было бы назвать migrate, но зачем конфликтовать с golang-migrate?
package migrations

import (
	"embed"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var fs embed.FS

// MigrationsUP проведение миграций для postgres
func MigrationsUP(connString string) error {
	d, err := iofs.New(fs, "migrations")
	if err != nil {
		return fmt.Errorf("создание драйвера для считывания миграций. %w", err)
	}
	// можно получить строку подключения разного вида, были с этим проблемы
	connString = strings.TrimPrefix(connString, "postgres://")
	connString = strings.TrimPrefix(connString, "postgresql://")
	connString = "pgx5://" + connString

	m, err := migrate.NewWithSourceInstance("iofs", d, connString)
	if err != nil {
		return fmt.Errorf("создание экземпляра миграций. %w", err)
	}
	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("применение миграций. %w", err)
	}
	return nil
}
