package main

import (
	"log"
	"os"

	"alabapi/database"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from system environment")
	}

	if err := database.InitAll(); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer database.CloseAll()

	app := fiber.New(fiber.Config{
		AppName: "AlabAPI",
	})

	app.Get("/api/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "ok",
			"databases": database.PingAll(),
		})
	})

	port := os.Getenv("PORT")
	if port == "" || port == "8080" {
		port = "4000"
	}

	log.Fatal(app.Listen("127.0.0.1:" + port))
}
