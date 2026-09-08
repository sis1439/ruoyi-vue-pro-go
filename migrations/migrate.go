// Package migrations applies immutable, forward-only SQL using golang-migrate v4.18.3.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"errors"

	migrate "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed *.up.sql
var Files embed.FS

// Apply runs through target (0 means latest). The caller owns db; no AutoMigrate is used.
func Apply(db *sql.DB, target uint) error {
	source, err := iofs.New(Files, ".")
	if err != nil {
		return err
	}
	defer source.Close()
	conn, err := db.Conn(context.Background())
	if err != nil {
		return err
	}
	defer conn.Close()
	driver, err := postgres.WithConnection(context.Background(), conn, &postgres.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return err
	}
	if target == 0 {
		err = m.Up()
	} else {
		err = m.Migrate(target)
	}
	if errors.Is(err, migrate.ErrNoChange) {
		return nil
	}
	if err != nil {
		// A failed SQL BEGIN/COMMIT migration can leave this connection aborted.
		// Roll it back and release our advisory lock before returning it to the caller pool.
		_, rollbackErr := conn.ExecContext(context.Background(), "ROLLBACK")
		unlockErr := driver.Unlock()
		err = errors.Join(err, rollbackErr, unlockErr)
	}
	return err
}
