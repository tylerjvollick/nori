package dbtest_test

import (
	"embed"
	"io/fs"
	"os"
	"testing"

	"github.com/tylerjvollick/nori/internal/dbtest"
)

// Embed at testmigrations/migrations/*.sql so we can fs.Sub to get
// the "migrations/*.sql" layout that StartPostgres expects.
//
//go:embed testmigrations/migrations/*.sql
var rawMigrationsFS embed.FS

func TestMain(m *testing.M) {
	subbed, err := fs.Sub(rawMigrationsFS, "testmigrations")
	if err != nil {
		panic(err)
	}
	os.Exit(dbtest.SetupTestMain(m, subbed))
}

func TestDBPool_returns_connection(t *testing.T) {
	db := dbtest.DBPool()
	if db == nil {
		t.Fatal("expected non-nil db")
	}

	var result int
	if err := db.Raw("SELECT 1").Scan(&result).Error; err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if result != 1 {
		t.Fatalf("expected 1, got %d", result)
	}
}

func TestTestTx_rolls_back(t *testing.T) {
	tx := dbtest.TestTx(t)

	tx.Exec("CREATE TEMP TABLE dbtest_check (val INT)")
	tx.Exec("INSERT INTO dbtest_check (val) VALUES (42)")

	var val int
	tx.Raw("SELECT val FROM dbtest_check").Scan(&val)
	if val != 42 {
		t.Fatalf("expected 42, got %d", val)
	}
}

func TestTestTx_isolation(t *testing.T) {
	// Two subtests each get their own transaction — writes in one must not
	// be visible in the other.
	t.Run("writer", func(t *testing.T) {
		tx := dbtest.TestTx(t)
		tx.Exec("INSERT INTO dbtest_items (name) VALUES ('from_writer')")
	})

	t.Run("reader", func(t *testing.T) {
		tx := dbtest.TestTx(t)
		var count int64
		tx.Raw("SELECT COUNT(*) FROM dbtest_items WHERE name = 'from_writer'").Scan(&count)
		if count != 0 {
			t.Fatalf("expected 0 rows from rolled-back tx, got %d", count)
		}
	})
}

func TestMigrations_applied(t *testing.T) {
	tx := dbtest.TestTx(t)

	var exists bool
	tx.Raw(`SELECT EXISTS (
		SELECT FROM information_schema.tables
		WHERE table_name = 'dbtest_items'
	)`).Scan(&exists)

	if !exists {
		t.Fatal("expected dbtest_items table to exist after migrations")
	}
}
