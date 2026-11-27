package routes

import (
	"alabapi/handlers"
	"alabapi/middleware"
	"alabapi/models"

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
	api.Post("/references", middleware.RequireAuth(), middleware.RequireModule(28), handlers.CreateReference)
	api.Put("/references/:seqnum", middleware.RequireAuth(), middleware.RequireModule(28), handlers.UpdateReference)
	api.Delete("/references/:seqnum", middleware.RequireAuth(), middleware.RequireModule(28), handlers.DeleteReference)

	api.Post("/auth/register", handlers.Register)
	api.Get("/auth/verify-email", handlers.VerifyEmailLink)
	api.Get("/auth/registration-status", handlers.GetRegistrationStatus)
	api.Post("/auth/resend-verification", handlers.ResendVerificationEmail)
	api.Post("/auth/verify-otp", handlers.VerifyAuthenticator)
	api.Post("/auth/login", handlers.Login)
	api.Post("/auth/forgot-password", handlers.ForgotPassword)
	api.Get("/auth/me", middleware.RequireAuth(), handlers.GetMe)

	api.Get("/dashboard", middleware.RequireAuth(), middleware.RequireModule(1), handlers.GetDashboard)

	api.Get("/rbac/my-access", middleware.RequireAuth(), handlers.GetMyAccess)
	api.Get("/rbac/modules", middleware.RequireAuth(), handlers.GetModules)
	api.Get("/rbac/role-access", middleware.RequireAuth(), handlers.GetRoleAccessList)
	api.Get("/rbac/role-access/:role_id", middleware.RequireAuth(), handlers.GetRoleAccessByRole)
	api.Put("/rbac/role-access/:role_id", middleware.RequireAuth(), middleware.RequireModule(29), handlers.UpdateRoleAccess)
	api.Patch("/rbac/role-access/:role_id/toggle", middleware.RequireAuth(), middleware.RequireModule(29), handlers.ToggleRoleModule)
	api.Delete("/rbac/role-access/:role_id/module/:module_id", middleware.RequireAuth(), middleware.RequireModule(29), handlers.RemoveRoleModule)

	api.Get("/players", handlers.GetPlayers)
	api.Post("/players", middleware.RequireAuth(), middleware.RequireModule(23), handlers.CreatePlayer)

	api.Get("/sessions", middleware.RequireAuth(), middleware.RequireModule(10), handlers.GetTrainingSessions)
	api.Post("/sessions", middleware.RequireAuth(), middleware.RequireModule(11), handlers.CreateTrainingSession)
	api.Put("/sessions/:id", middleware.RequireAuth(), middleware.RequireModule(12), handlers.UpdateTrainingSession)
	api.Delete("/sessions/:id", middleware.RequireAuth(), middleware.RequireModule(12), handlers.DeleteTrainingSession)
	api.Get("/sessions/:id/attendance", middleware.RequireAuth(), middleware.RequireModule(13), handlers.GetSessionAttendance)

	api.Get("/attendance/daily", middleware.RequireAuth(), middleware.RequireModule(13), handlers.GetDailyAttendance)
	api.Post("/attendance/daily", middleware.RequireAuth(), middleware.RequireModule(14), handlers.RecordDailyAttendance)
	api.Get("/attendance/history", middleware.RequireAuth(), middleware.RequireModule(13), handlers.GetAllPlayersAttendanceHistory)
	api.Get("/attendance/player/:player_id", middleware.RequireAuth(), middleware.RequireModule(13), handlers.GetPlayerAttendanceHistory)
	api.Post("/attendance", middleware.RequireAuth(), middleware.RequireModule(14), handlers.RecordAttendance)
	api.Post("/attendance/bulk", middleware.RequireAuth(), middleware.RequireModule(14), handlers.RecordBulkAttendance)

	api.Get("/users", middleware.RequireAuth(), middleware.RequireModule(26), handlers.GetUsers)
	api.Get("/users/:id", middleware.RequireAuth(), handlers.GetUserByID)
	api.Put("/users/:id/profile", middleware.RequireAuth(), handlers.UpdateUserProfile)
	api.Put("/users/:id/role", middleware.RequireAuth(), middleware.RequireRoles(models.RefRoleAdmin), middleware.RequireModule(27), handlers.UpdateUserRole)

	api.Get("/my-players", middleware.RequireAuth(), handlers.GetMyPlayers)
	api.Get("/parents", middleware.RequireAuth(), handlers.GetParents)
	api.Get("/parents/:parent_id/players", middleware.RequireAuth(), handlers.GetParentPlayers)
	api.Post("/parents/link-player", middleware.RequireAuth(), middleware.RequireModule(24), handlers.LinkParentPlayer)
	api.Delete("/parents/link-player/:id", middleware.RequireAuth(), middleware.RequireModule(24), handlers.UnlinkParentPlayer)

	api.Get("/soa", middleware.RequireAuth(), middleware.RequireModule(16), handlers.GetSOAs)
	api.Get("/soa/player/:player_id", middleware.RequireAuth(), middleware.RequireModule(16), handlers.GetPlayerSOAs)
	api.Get("/soa/:id", middleware.RequireAuth(), middleware.RequireModule(16), handlers.GetSOAByID)
	api.Post("/soa", middleware.RequireAuth(), middleware.RequireModule(17), handlers.CreateSOA)
	api.Put("/soa/:id", middleware.RequireAuth(), middleware.RequireModule(17), handlers.UpdateSOA)
	api.Delete("/soa/:id", middleware.RequireAuth(), middleware.RequireModule(18), handlers.DeleteSOA)
	api.Post("/soa/:id/pay", middleware.RequireAuth(), middleware.RequireModule(19), handlers.SubmitSOAPayment)
	api.Patch("/soa/payments/:payment_id/verify", middleware.RequireAuth(), middleware.RequireModule(20), handlers.VerifySOAPayment)

	api.Get("/tournaments", handlers.GetTournaments)
	api.Post("/tournaments", middleware.RequireAuth(), middleware.RequireModule(4), handlers.CreateTournament)
	api.Put("/tournaments/matches/:match_id", middleware.RequireAuth(), middleware.RequireModule(5), handlers.UpdateTournamentMatch)
	api.Delete("/tournaments/matches/:match_id", middleware.RequireAuth(), middleware.RequireModule(5), handlers.DeleteTournamentMatch)
	api.Patch("/tournaments/roster/:roster_id/consent", middleware.RequireAuth(), middleware.RequireModule(8), handlers.UpdateParentConsent)
	api.Delete("/tournaments/roster/:roster_id", middleware.RequireAuth(), middleware.RequireModule(7), handlers.RemoveTournamentRoster)
	api.Get("/tournaments/:id", handlers.GetTournamentByID)
	api.Put("/tournaments/:id", middleware.RequireAuth(), middleware.RequireModule(5), handlers.UpdateTournament)
	api.Delete("/tournaments/:id", middleware.RequireAuth(), middleware.RequireModule(6), handlers.DeleteTournament)
	api.Get("/tournaments/:id/matches", handlers.GetTournamentMatches)
	api.Post("/tournaments/:id/matches", middleware.RequireAuth(), middleware.RequireModule(4), handlers.AddTournamentMatches)
	api.Get("/tournaments/:id/roster", handlers.GetTournamentRoster)
	api.Post("/tournaments/:id/roster", middleware.RequireAuth(), middleware.RequireModule(7), handlers.AddTournamentRoster)
}
