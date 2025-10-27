package main

import (
	"context"
	"log"
	"os"

	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"

	"github.com/tylerjvollick/nori/internal/app"
	"github.com/tylerjvollick/nori/internal/database"
)

func main() {
	godotenv.Load(".env")

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

	// app := fiber.New()

	// Register routes
	//handlers.RegisterHealthRoutes(app, conn)

	//userRepo := repositories.NewUserRepository(database.DB)
	//authService := services.NewAuthService(userRepo)
	//authHandler := handlers.NewAuthHandler(authService)
	//authHandler.RegisterAuthRoutes(app)

	a := app.New()
	log.Fatal(a.Fiber.Listen(":8080"))
}
