package routes

import (
	"alabapi/handlers"
	"alabapi/middleware"

	"github.com/gofiber/fiber/v2"
)

const AppVersion = "1.0.0"

func healthHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Welcome to ALAB FC API",
		"version": AppVersion,
	})
}

func SetupRoutes(app *fiber.App) {
	app.Get("/", healthHandler)
	app.Get("/health", healthHandler)

	api := app.Group("/api")
	api.Get("/health", healthHandler)

	api.Get("/references", handlers.GetReferences)
	api.Get("/references/parent/:parent_id", handlers.GetReferencesByParent)
	api.Post("/references", middleware.RequireAuth(), handlers.CreateReference)
	api.Put("/references/:seqnum", middleware.RequireAuth(), handlers.UpdateReference)
	api.Delete("/references/:seqnum", middleware.RequireAuth(), handlers.DeleteReference)

	api.Post("/auth/register", handlers.Register)
	api.Post("/auth/verify-otp", handlers.VerifyAuthenticator)
	api.Post("/auth/login", handlers.Login)
	api.Post("/auth/forgot-password", handlers.ForgotPassword)
	api.Get("/auth/me", middleware.RequireAuth(), handlers.GetMe)

	api.Get("/players", handlers.GetPlayers)
	api.Get("/users", middleware.RequireAuth(), handlers.GetUsers)
	api.Get("/users/:id", middleware.RequireAuth(), handlers.GetUserByID)
	api.Put("/users/:id/profile", middleware.RequireAuth(), handlers.UpdateUserProfile)

	api.Get("/parents/:parent_id/players", middleware.RequireAuth(), handlers.GetParentPlayers)
	api.Post("/parents/link-player", middleware.RequireAuth(), handlers.LinkParentPlayer)
	api.Delete("/parents/link-player/:id", middleware.RequireAuth(), handlers.UnlinkParentPlayer)
}
