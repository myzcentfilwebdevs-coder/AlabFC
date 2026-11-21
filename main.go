package main

import (
	"log"
	"os"

	"alabapi/database"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	if err := database.InitAll(); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer database.CloseAll()

	app := fiber.New(fiber.Config{
		AppName: "AlabAPI",
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"service": "AlabAPI",
			"status":  "running",
		})
	})

	app.Get("/api/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "ok",
			"databases": database.PingAll(),
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "4000"
	}

	log.Fatal(app.Listen("0.0.0.0:" + port))
}
