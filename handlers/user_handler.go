package handlers

import (
	"alabapi/database"
	"alabapi/models"

	"github.com/gofiber/fiber/v2"
)

func GetUsers(c *fiber.Ctx) error {
	db := database.Get()
	var users []models.User

	roleID := c.Query("role_id")
	if roleID != "" {
		db.Preload("Profile").Where("role_id = ?", roleID).Find(&users)
	} else {
		db.Preload("Profile").Find(&users)
	}

	return c.JSON(users)
}

func GetPlayers(c *fiber.Ctx) error {
	db := database.Get()
	var players []models.User

	db.Preload("Profile").Where("role_id = ?", models.RefRolePlayer).Find(&players)
	return c.JSON(players)
}

func GetUserByID(c *fiber.Ctx) error {
	var user models.User
	if err := database.Get().Preload("Profile").First(&user, c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}

	return c.JSON(user)
}

func UpdateUserProfile(c *fiber.Ctx) error {
	db := database.Get()
	var profile models.UserProfile

	if err := db.Where("user_id = ?", c.Params("id")).First(&profile).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Profile not found"})
	}

	if err := c.BodyParser(&profile); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	db.Save(&profile)
	return c.JSON(fiber.Map{
		"message": "Profile updated",
		"profile": profile,
	})
}

func LinkParentPlayer(c *fiber.Ctx) error {
	var link models.ParentPlayer
	if err := c.BodyParser(&link); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := database.Get().Create(&link).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Failed to link parent and player"})
	}

	return c.Status(201).JSON(fiber.Map{
		"message": "Parent and player linked",
		"data":    link,
	})
}

func GetParentPlayers(c *fiber.Ctx) error {
	var links []models.ParentPlayer
	database.Get().
		Preload("Player.Profile").
		Where("parent_user_id = ?", c.Params("parent_id")).
		Find(&links)

	return c.JSON(links)
}

func UnlinkParentPlayer(c *fiber.Ctx) error {
	database.Get().Delete(&models.ParentPlayer{}, c.Params("id"))
	return c.JSON(fiber.Map{"message": "Parent and player unlinked"})
}
