package middleware

import (
	"strconv"
	"strings"

	"alabapi/database"
	"alabapi/models"
	"alabapi/utils"

	"github.com/gofiber/fiber/v2"
)

func RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := strings.TrimSpace(c.Get("Authorization"))
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing Authorization header",
			})
		}

		tokenStr := authHeader
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			tokenStr = strings.TrimSpace(authHeader[7:])
		}

		claims, err := utils.ParseJWT(tokenStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized: " + err.Error(),
			})
		}

		c.Locals("claims", claims)
		c.Locals("user_id", claims.UserID)
		c.Locals("role_id", claims.RoleID)
		return c.Next()
	}
}

func RequireRoles(allowedRoles ...int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		roleID, ok := c.Locals("role_id").(int)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Forbidden: role not found",
			})
		}

		for _, allowed := range allowedRoles {
			if roleID == allowed {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Forbidden: insufficient role permissions",
		})
	}
}

func RequireModule(moduleID int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		roleID, ok := c.Locals("role_id").(int)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Forbidden: role not found",
			})
		}

		var access models.RoleAccess
		if err := database.Get().Where("roleid = ?", roleID).First(&access).Error; err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Forbidden: no role access configured",
			})
		}

		target := strconv.Itoa(moduleID)
		for _, id := range strings.Split(access.ModuleIDs, ",") {
			if strings.TrimSpace(id) == target {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Forbidden: you do not have access to this module",
		})
	}
}

func GetClaims(c *fiber.Ctx) *utils.JWTClaims {
	if claims, ok := c.Locals("claims").(*utils.JWTClaims); ok {
		return claims
	}
	return nil
}
