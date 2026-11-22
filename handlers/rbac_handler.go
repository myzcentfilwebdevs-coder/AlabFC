package handlers

import (
	"strconv"
	"strings"

	"alabapi/database"
	"alabapi/models"

	"github.com/gofiber/fiber/v2"
)

func parseModuleIDs(csv string) []int {
	ids := []int{}
	for _, part := range strings.Split(csv, ",") {
		part = strings.TrimSpace(part)
		if id, err := strconv.Atoi(part); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

func getRoleModules(roleID int) ([]int, []models.AppModule) {
	db := database.Get()
	var access models.RoleAccess
	if err := db.Where("roleid = ?", roleID).First(&access).Error; err != nil {
		return []int{}, []models.AppModule{}
	}

	ids := parseModuleIDs(access.ModuleIDs)
	if len(ids) == 0 {
		return []int{}, []models.AppModule{}
	}

	var modules []models.AppModule
	db.Where("seqnum IN ?", ids).Order("seqnum ASC").Find(&modules)
	return ids, modules
}

func getRoleName(roleID int) string {
	var ref models.GenericReference
	if err := database.Get().First(&ref, roleID).Error; err != nil {
		return ""
	}
	return ref.Details
}

func GetModules(c *fiber.Ctx) error {
	var modules []models.AppModule
	database.Get().Order("seqnum ASC").Find(&modules)
	return c.JSON(modules)
}

func GetRoleAccessList(c *fiber.Ctx) error {
	var list []models.RoleAccess
	database.Get().Order("roleid ASC").Find(&list)

	result := make([]fiber.Map, 0, len(list))
	for _, item := range list {
		ids, modules := getRoleModules(item.RoleID)
		result = append(result, fiber.Map{
			"seqnum":     item.Seqnum,
			"role_id":    item.RoleID,
			"role_name":  getRoleName(item.RoleID),
			"module_ids": ids,
			"modules":    modules,
		})
	}
	return c.JSON(result)
}

func GetRoleAccessByRole(c *fiber.Ctx) error {
	roleID, err := strconv.Atoi(c.Params("role_id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid role_id"})
	}

	ids, modules := getRoleModules(roleID)
	return c.JSON(fiber.Map{
		"role_id":    roleID,
		"role_name":  getRoleName(roleID),
		"module_ids": ids,
		"modules":    modules,
	})
}

func UpdateRoleAccess(c *fiber.Ctx) error {
	roleID, err := strconv.Atoi(c.Params("role_id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid role_id"})
	}

	var req struct {
		ModuleIDs string `json:"module_ids"`
		IDs       []int  `json:"ids"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	csv := strings.TrimSpace(req.ModuleIDs)
	if csv == "" && len(req.IDs) > 0 {
		parts := make([]string, 0, len(req.IDs))
		for _, id := range req.IDs {
			parts = append(parts, strconv.Itoa(id))
		}
		csv = strings.Join(parts, ",")
	}

	db := database.Get()
	var access models.RoleAccess
	if err := db.Where("roleid = ?", roleID).First(&access).Error; err != nil {
		access = models.RoleAccess{
			RoleID:    roleID,
			ModuleIDs: csv,
		}
		db.Create(&access)
	} else {
		access.ModuleIDs = csv
		db.Save(&access)
	}

	ids, modules := getRoleModules(roleID)
	return c.JSON(fiber.Map{
		"message":    "Role access updated",
		"role_id":    roleID,
		"role_name":  getRoleName(roleID),
		"module_ids": ids,
		"modules":    modules,
	})
}

func GetMyAccess(c *fiber.Ctx) error {
	roleID, _ := c.Locals("role_id").(int)
	ids, modules := getRoleModules(roleID)

	return c.JSON(fiber.Map{
		"role_id":    roleID,
		"role_name":  getRoleName(roleID),
		"module_ids": ids,
		"modules":    modules,
	})
}
