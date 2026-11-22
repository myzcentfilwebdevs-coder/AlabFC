package main

import (
	"log"
	"os"

	"alabapi/database"
	"alabapi/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	if err := database.InitAll(); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer database.CloseAll()

	app := fiber.New(fiber.Config{
		AppName: "ALAB FC v" + routes.AppVersion,
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, PATCH, DELETE, OPTIONS",
	}))
	app.Use(logger.New())

	routes.SetupRoutes(app)

	port := os.Getenv("PORT")
	if port == "" {
		port = "4000"
	}

	log.Fatal(app.Listen("0.0.0.0:" + port))
}
