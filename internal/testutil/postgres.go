package testutil

import (
	"fmt"
	"github.com/wxlbd/ruoyi-mall-go/migrations"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"net/url"
	"os"
	"testing"
	"time"
)

// PostgreSQL creates and migrates a unique schema; every pooled connection uses it.
// Tests explicitly skip without TEST_POSTGRES_DSN. CI must set it for integration acceptance.
func PostgreSQL(t testing.TB) *gorm.DB { return PostgreSQLAt(t, 0) }

// PostgreSQLAt creates a schema at a historical version for upgrade testing.
func PostgreSQLAt(t testing.TB, target uint) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN required for real PostgreSQL integration")
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := admin.DB()
	if err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("test_%d", time.Now().UnixNano())
	if err := admin.Exec("CREATE SCHEMA " + name).Error; err != nil {
		pool.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := admin.Exec("DROP SCHEMA " + name + " CASCADE").Error; err != nil {
			t.Error(err)
		}
		pool.Close()
	})
	if u, err := url.Parse(dsn); err == nil && (u.Scheme == "postgres" || u.Scheme == "postgresql") {
		q := u.Query()
		q.Set("search_path", name)
		u.RawQuery = q.Encode()
		dsn = u.String()
	} else {
		dsn += " search_path=" + name
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	p, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { p.Close() })
	if err := migrations.Apply(p, target); err != nil {
		t.Fatal(err)
	}
	return db
}
