package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	"github.com/tylerjvollick/nori/internal/app"
	"github.com/tylerjvollick/nori/internal/config"
	"github.com/tylerjvollick/nori/internal/database"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Nori server",
	Long:  "Start the Nori HTTP server and connect to the database.",
	Run: func(cmd *cobra.Command, args []string) {
		godotenv.Load(".env")

		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			log.Fatalf("Failed to load configuration: %v", err)
		}

		log.Println("Connecting to database")
		database.Connect()

		log.Println("setup complete")

		connString := fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=disable",
			os.Getenv("DB_USER"),
			os.Getenv("DB_PASSWORD"),
			os.Getenv("DB_HOST"),
			os.Getenv("DB_PORT"),
			os.Getenv("DB_NAME"),
		)

		conn, err := pgx.Connect(context.Background(), connString)
		if err != nil {
			log.Fatalf("Unable to connect to database: %v\n", err)
		}
		defer conn.Close(context.Background())

		a := app.New(cfg)
		log.Fatal(a.Fiber.Listen(":8080"))
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
