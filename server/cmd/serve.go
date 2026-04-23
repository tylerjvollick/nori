package cmd

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	"github.com/tylerjvollick/nori/internal/app"
	"github.com/tylerjvollick/nori/internal/config"
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

		port := os.Getenv("NORI_PORT")
		if port == "" {
			port = "8080"
		}

		a := app.New(cfg)
		log.Fatal(a.Fiber.Listen(":" + port))
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
