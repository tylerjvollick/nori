// Package dbtest provides test database lifecycle management using
// testcontainers-go with PostgreSQL. It launches a real PostgreSQL
// container, applies all embedded migrations, and offers per-test
// transaction isolation that auto-rolls-back on cleanup.
package dbtest

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"sync"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	sharedDB   *gorm.DB
	sharedDSN  string
	setupOnce  sync.Once
	setupErr   error
	container  *tcpostgres.PostgresContainer
)

// StartPostgres launches a PostgreSQL 17 container using testcontainers and
// applies all migrations from the provided embedded filesystem. It is safe
// to call from TestMain; call the returned cleanup function in the deferred
// teardown (or use SetupTestMain which handles this automatically).
//
// The migrationsFS should contain migration files at "migrations/*.sql"
// (the same embed.FS used by the production binary).
func StartPostgres(migrationsFS fs.FS) (dsn string, cleanup func(), err error) {
	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:17",
		tcpostgres.WithDatabase("nori_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
		tcpostgres.WithSQLDriver("pgx"),
	)
	if err != nil {
		return "", nil, fmt.Errorf("start postgres container: %w", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		pgContainer.Terminate(ctx) //nolint:errcheck
		return "", nil, fmt.Errorf("get connection string: %w", err)
	}

	// Apply migrations.
	subFS, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		pgContainer.Terminate(ctx) //nolint:errcheck
		return "", nil, fmt.Errorf("create migration sub-fs: %w", err)
	}

	source, err := iofs.New(subFS, ".")
	if err != nil {
		pgContainer.Terminate(ctx) //nolint:errcheck
		return "", nil, fmt.Errorf("create migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, connStr)
	if err != nil {
		pgContainer.Terminate(ctx) //nolint:errcheck
		return "", nil, fmt.Errorf("init migrator: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		pgContainer.Terminate(ctx) //nolint:errcheck
		return "", nil, fmt.Errorf("apply migrations: %w", err)
	}
	srcErr, dbErr := m.Close()
	if srcErr != nil {
		pgContainer.Terminate(ctx) //nolint:errcheck
		return "", nil, fmt.Errorf("close migration source: %w", srcErr)
	}
	if dbErr != nil {
		pgContainer.Terminate(ctx) //nolint:errcheck
		return "", nil, fmt.Errorf("close migration db: %w", dbErr)
	}

	cleanup = func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			log.Printf("dbtest: failed to terminate postgres container: %v", err)
		}
	}

	return connStr, cleanup, nil
}

// DBPool returns a lazy-initialized GORM *gorm.DB connected to the shared
// test container. It uses sync.Once so the connection is created at most once
// per test binary. Requires SetupTestMain (or manual StartPostgres + SetShared)
// to have been called first.
func DBPool() *gorm.DB {
	setupOnce.Do(func() {
		if sharedDSN == "" {
			setupErr = fmt.Errorf("dbtest: DBPool called before SetupTestMain or SetShared")
			return
		}
		db, err := gorm.Open(postgres.Open(sharedDSN), &gorm.Config{})
		if err != nil {
			setupErr = fmt.Errorf("dbtest: open gorm connection: %w", err)
			return
		}
		sharedDB = db
	})
	if setupErr != nil {
		panic(setupErr)
	}
	return sharedDB
}

// SetShared stores the DSN from StartPostgres so DBPool can lazily open a
// connection. This is called automatically by SetupTestMain.
func SetShared(dsn string) {
	sharedDSN = dsn
}

// TestTx returns a GORM *gorm.DB backed by a transaction that will be rolled
// back when the test (or subtest) completes. This provides complete isolation
// between tests without needing to manually clean up data.
func TestTx(t interface {
	Cleanup(func())
	Fatal(...interface{})
}) *gorm.DB {
	db := DBPool()

	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal("dbtest: begin transaction:", tx.Error)
	}

	t.Cleanup(func() {
		tx.Rollback()
	})

	return tx
}

// SetupTestMain is designed to be called from TestMain. It starts a PostgreSQL
// container, applies migrations, and makes the connection available via
// DBPool() and TestTx(). Returns an exit code suitable for os.Exit.
//
// Usage in a *_test.go file:
//
//	func TestMain(m *testing.M) {
//	    os.Exit(dbtest.SetupTestMain(m, migrationsFS))
//	}
func SetupTestMain(m interface{ Run() int }, migrationsFS fs.FS) int {
	dsn, cleanup, err := StartPostgres(migrationsFS)
	if err != nil {
		log.Fatalf("dbtest: %v", err)
	}
	defer cleanup()

	SetShared(dsn)

	return m.Run()
}

// MustSetupTestMain is like SetupTestMain but calls os.Exit directly.
// Convenient when TestMain has no other setup:
//
//	func TestMain(m *testing.M) {
//	    dbtest.MustSetupTestMain(m, migrationsFS)
//	}
func MustSetupTestMain(m interface{ Run() int }, migrationsFS fs.FS) {
	os.Exit(SetupTestMain(m, migrationsFS))
}
