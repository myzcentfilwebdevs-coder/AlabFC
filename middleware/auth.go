package middleware

import (
	"strings"

	"alabapi/utils"

	"github.com/gofiber/fiber/v2"
)

func RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := strings.TrimSpace(c.Get("Authorization"))
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Missing Authorization header",
			})
		}

		tokenStr := authHeader
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			tokenStr = strings.TrimSpace(authHeader[7:])
		}

		claims, err := utils.ParseJWT(tokenStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Unauthorized: " + err.Error(),
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
		roleVal := c.Locals("role_id")
		roleID, ok := roleVal.(int)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error":   "Forbidden: role not found in session",
			})
		}

		for _, allowed := range allowedRoles {
			if roleID == allowed {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "Forbidden: insufficient role permissions",
		})
	}
}

func GetClaims(c *fiber.Ctx) *utils.JWTClaims {
	if claims, ok := c.Locals("claims").(*utils.JWTClaims); ok {
		return claims
	}
	return nil
}
