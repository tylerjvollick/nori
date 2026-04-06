package cmd

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

// MigrationsFS is set by main before Execute() to provide the embedded SQL
// migration files (server/migrations/*.sql).
var MigrationsFS embed.FS

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Database migration commands",
	Long:  "Run database migrations embedded in the nori binary.",
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Run all pending migrations",
	Long:  "Apply all pending database migrations to bring the schema up to date.",
	RunE:  runMigrateUp,
}

func init() {
	migrateCmd.AddCommand(migrateUpCmd)
	rootCmd.AddCommand(migrateCmd)
}

// buildDSN constructs a PostgreSQL connection string from DB_* environment
// variables (DB_HOST, DB_USER, DB_PASSWORD, DB_NAME, DB_PORT).
func buildDSN() string {
	host := envOrDefault("DB_HOST", "localhost")
	user := envOrDefault("DB_USER", "postgres")
	password := os.Getenv("DB_PASSWORD")
	name := envOrDefault("DB_NAME", "nori")
	port := envOrDefault("DB_PORT", "5432")
	sslmode := envOrDefault("DB_SSLMODE", "disable")

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, password, host, port, name, sslmode)
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func runMigrateUp(cmd *cobra.Command, args []string) error {
	godotenv.Load(".env")

	// The embedded FS has files at migrations/*.sql — create a sub-FS
	// rooted at "migrations" so iofs sees them at the top level.
	subFS, err := fs.Sub(MigrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create sub-filesystem: %w", err)
	}

	source, err := iofs.New(subFS, ".")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	dsn := buildDSN()

	m, err := migrate.NewWithSourceInstance("iofs", source, dsn)
	if err != nil {
		return fmt.Errorf("failed to initialise migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			log.Println("Database is already up to date.")
			return nil
		}
		return fmt.Errorf("migration failed: %w", err)
	}

	log.Println("All migrations applied successfully.")
	return nil
}
