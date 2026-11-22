package handlers

import (
	"alabapi/database"
	"alabapi/models"

	"github.com/gofiber/fiber/v2"
)

func GetDashboard(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(int)
	db := database.Get()

	var user models.User
	if err := db.Preload("Profile").First(&user, userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}

	roleID := user.RoleID
	roleName := getRoleName(roleID)
	moduleIDs, modules := getRoleModules(roleID)

	var playerIDs []int
	var children []models.ParentPlayer

	if roleID == models.RefRoleParent {
		db.Preload("Player.Profile").Where("parent_user_id = ?", userID).Find(&children)
		for _, ch := range children {
			playerIDs = append(playerIDs, ch.PlayerUserID)
		}
	} else if roleID == models.RefRolePlayer {
		playerIDs = []int{userID}
	}

	var upcomingTournaments []models.Tournament
	db.Where("status_id = 37 OR start_date >= CURRENT_DATE").
		Order("start_date ASC").
		Limit(5).
		Find(&upcomingTournaments)

	var totalUpcomingTournaments int64
	db.Model(&models.Tournament{}).
		Where("status_id = 37 OR start_date >= CURRENT_DATE").
		Count(&totalUpcomingTournaments)

	var totalPlayers int64
	if roleID == models.RefRoleParent {
		totalPlayers = int64(len(playerIDs))
	} else {
		db.Model(&models.User{}).Where("role_id = ?", models.RefRolePlayer).Count(&totalPlayers)
	}

	var recentAttendance []models.PlayerAttendance
	if roleID == models.RefRoleAdmin || roleID == models.RefRoleCoach {
		db.Preload("Player.Profile").Order("created_at DESC").Limit(10).Find(&recentAttendance)
	} else if len(playerIDs) > 0 {
		db.Preload("Player.Profile").Where("player_user_id IN ?", playerIDs).Order("created_at DESC").Limit(10).Find(&recentAttendance)
	} else {
		recentAttendance = []models.PlayerAttendance{}
	}

	var pendingSOAs []models.StatementOfAccount
	var totalBalance float64

	if roleID == models.RefRoleAdmin {
		db.Where("status_id IN ?", []int{models.RefSOAUnpaid, models.RefSOAPartial, 45}).
			Order("due_date ASC").
			Limit(10).
			Find(&pendingSOAs)
		db.Model(&models.StatementOfAccount{}).
			Where("status_id IN ?", []int{models.RefSOAUnpaid, models.RefSOAPartial, 45}).
			Select("COALESCE(SUM(balance), 0)").
			Scan(&totalBalance)
	} else if roleID == models.RefRoleParent {
		query := db.Where("parent_user_id = ?", userID)
		if len(playerIDs) > 0 {
			query = db.Where("parent_user_id = ? OR player_user_id IN ?", userID, playerIDs)
		}
		query.Order("due_date ASC").Limit(10).Find(&pendingSOAs)
		for _, soa := range pendingSOAs {
			totalBalance += soa.Balance
		}
	} else {
		pendingSOAs = []models.StatementOfAccount{}
	}

	return c.JSON(fiber.Map{
		"user":      user,
		"role_id":   roleID,
		"role_name": roleName,
		"access": fiber.Map{
			"module_ids": moduleIDs,
			"modules":    modules,
		},
		"stats": fiber.Map{
			"upcoming_tournaments": totalUpcomingTournaments,
			"total_players":        totalPlayers,
			"pending_soa_count":    len(pendingSOAs),
			"total_balance":        totalBalance,
		},
		"children":             children,
		"upcoming_tournaments": upcomingTournaments,
		"recent_attendance":    recentAttendance,
		"soas":                 pendingSOAs,
	})
}
