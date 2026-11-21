package main

import (
	"log"
	"os"

	"alabapi/database"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

const AppVersion = "1.0.0"

func healthHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Welcome to ALAB FC",
		"version": AppVersion,
	})
}

func main() {
	_ = godotenv.Load()

	if err := database.InitAll(); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer database.CloseAll()

	app := fiber.New(fiber.Config{
		AppName: "ALAB FC v" + AppVersion,
	})

	app.Get("/", healthHandler)
	app.Get("/health", healthHandler)
	app.Get("/api/health", healthHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "4000"
	}

	log.Fatal(app.Listen("0.0.0.0:" + port))
}
