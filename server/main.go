package main

import (
	"context"
	"log"
	"os"

	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"

	"github.com/tylerjvollick/nori/internal/handlers"
	"github.com/tylerjvollick/nori/internal/database"

	"github.com/tylerjvollick/nori/internal/models"
	"github.com/google/uuid"
)

func main() {
	godotenv.Load(".env")

	log.Println("Connecting to database")
	database.Connect()

	user:= models.User{
		ID: uuid.New(),
		Email: "test@example.com",
	}
	database.DB.Create(&user)

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

	app := fiber.New()

	// Register routes
	handlers.RegisterHealthRoutes(app, conn)
	handlers.RegisterAuthRoutes(app, conn)

	log.Fatal(app.Listen(":8080"))
}

