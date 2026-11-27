package handlers

import (
	"alabapi/database"
	"alabapi/models"

	"github.com/gofiber/fiber/v2"
)

func getParentChildIDs(parentUserID int) []int {
	var links []models.ParentPlayer
	database.Get().Where("parent_user_id = ?", parentUserID).Find(&links)

	ids := make([]int, 0, len(links))
	for _, l := range links {
		ids = append(ids, l.PlayerID)
	}
	return ids
}

func canAccessPlayer(c *fiber.Ctx, targetPlayerID int) bool {
	userID, _ := c.Locals("user_id").(int)
	roleID, _ := c.Locals("role_id").(int)

	if roleID == models.RefRoleAdmin || roleID == models.RefRoleCoach {
		return true
	}
	if roleID == models.RefRoleParent {
		var count int64
		database.Get().Model(&models.ParentPlayer{}).
			Where("parent_user_id = ? AND player_id = ?", userID, targetPlayerID).
			Count(&count)
		return count > 0
	}
	return false
}

func canAccessSOA(c *fiber.Ctx, soa models.StatementOfAccount) bool {
	userID, _ := c.Locals("user_id").(int)
	roleID, _ := c.Locals("role_id").(int)

	if roleID == models.RefRoleAdmin {
		return true
	}
	if roleID == models.RefRoleParent {
		if soa.ParentUserID != nil && *soa.ParentUserID == userID {
			return true
		}
		return canAccessPlayer(c, soa.PlayerID)
	}
	return false
}
