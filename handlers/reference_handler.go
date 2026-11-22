package handlers

import (
	"alabapi/database"
	"alabapi/models"

	"github.com/gofiber/fiber/v2"
)

func GetReferences(c *fiber.Ctx) error {
	db := database.Get()
	var refs []models.GenericReference

	parentID := c.Query("parent_id")
	if parentID != "" {
		db.Where("parent_id = ?", parentID).Order("seqnum ASC").Find(&refs)
	} else {
		db.Order("seqnum ASC").Find(&refs)
	}

	return c.JSON(refs)
}

func GetReferencesByParent(c *fiber.Ctx) error {
	db := database.Get()
	var refs []models.GenericReference

	db.Where("parent_id = ?", c.Params("parent_id")).Order("seqnum ASC").Find(&refs)
	return c.JSON(refs)
}

func CreateReference(c *fiber.Ctx) error {
	var ref models.GenericReference
	if err := c.BodyParser(&ref); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := database.Get().Create(&ref).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(ref)
}

func UpdateReference(c *fiber.Ctx) error {
	db := database.Get()
	var ref models.GenericReference

	if err := db.First(&ref, c.Params("seqnum")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Reference not found"})
	}

	if err := c.BodyParser(&ref); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	db.Save(&ref)
	return c.JSON(ref)
}

func DeleteReference(c *fiber.Ctx) error {
	database.Get().Delete(&models.GenericReference{}, c.Params("seqnum"))
	return c.JSON(fiber.Map{"message": "Reference deleted"})
}
