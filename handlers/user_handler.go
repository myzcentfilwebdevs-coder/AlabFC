package handlers

import (
	"fmt"
	"strings"
	"time"

	"alabapi/database"
	"alabapi/models"
	"alabapi/utils"

	"github.com/gofiber/fiber/v2"
)

func findUserIDByName(name string, roleID int) (int, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("name is empty")
	}

	db := database.Get()
	var user models.User

	query := db.Model(&models.User{}).
		Joins("LEFT JOIN user_profiles ON user_profiles.user_id = users.user_id")

	if roleID > 0 {
		query = query.Where("users.role_id = ?", roleID)
	}

	err := query.Where(
		"LOWER(TRIM(COALESCE(user_profiles.first_name, '') || ' ' || COALESCE(user_profiles.last_name, ''))) = LOWER(?) OR "+
			"LOWER(TRIM(COALESCE(user_profiles.first_name, '') || ' ' || COALESCE(user_profiles.middle_name, '') || ' ' || COALESCE(user_profiles.last_name, ''))) = LOWER(?) OR "+
			"LOWER(users.username) = LOWER(?) OR LOWER(users.email) = LOWER(?)",
		name, name, name, name,
	).First(&user).Error

	if err == nil {
		return user.UserID, nil
	}

	like := "%" + name + "%"
	query2 := db.Model(&models.User{}).
		Joins("LEFT JOIN user_profiles ON user_profiles.user_id = users.user_id")
	if roleID > 0 {
		query2 = query2.Where("users.role_id = ?", roleID)
	}

	err = query2.Where(
		"TRIM(COALESCE(user_profiles.first_name, '') || ' ' || COALESCE(user_profiles.last_name, '')) ILIKE ? OR "+
			"user_profiles.first_name ILIKE ? OR user_profiles.last_name ILIKE ? OR users.username ILIKE ?",
		like, like, like, like,
	).First(&user).Error

	if err != nil {
		return 0, fmt.Errorf("user %q not found", name)
	}
	return user.UserID, nil
}

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

func GetParents(c *fiber.Ctx) error {
	db := database.Get()
	query := db.Model(&models.User{}).
		Preload("Profile").
		Joins("LEFT JOIN user_profiles ON user_profiles.user_id = users.user_id").
		Where("users.role_id = ?", models.RefRoleParent)

	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + search + "%"
		query = query.Where(
			"TRIM(COALESCE(user_profiles.first_name, '') || ' ' || COALESCE(user_profiles.last_name, '')) ILIKE ? OR "+
				"user_profiles.first_name ILIKE ? OR user_profiles.last_name ILIKE ? OR users.username ILIKE ? OR users.email ILIKE ?",
			like, like, like, like, like,
		)
	}

	var parents []models.User
	query.Order("user_profiles.last_name ASC, user_profiles.first_name ASC").Find(&parents)

	result := make([]fiber.Map, 0, len(parents))
	for _, p := range parents {
		firstName, middleName, lastName, phone, avatar := "", "", "", "", ""
		if p.Profile != nil {
			firstName = p.Profile.FirstName
			middleName = p.Profile.MiddleName
			lastName = p.Profile.LastName
			phone = p.Profile.PhoneNumber
			avatar = p.Profile.AvatarURL
		}
		fullName := strings.TrimSpace(firstName + " " + lastName)
		if fullName == "" {
			fullName = p.Username
		}
		result = append(result, fiber.Map{
			"user_id":      p.UserID,
			"username":     p.Username,
			"email":        p.Email,
			"first_name":   firstName,
			"middle_name":  middleName,
			"last_name":    lastName,
			"full_name":    fullName,
			"phone_number": phone,
			"avatar_url":   avatar,
			"profile":      p.Profile,
		})
	}

	return c.JSON(result)
}

func GetPlayers(c *fiber.Ctx) error {
	db := database.Get()
	query := db.Model(&models.User{}).
		Preload("Profile").
		Joins("LEFT JOIN user_profiles ON user_profiles.user_id = users.user_id").
		Where("users.role_id = ?", models.RefRolePlayer)

	if ageCat := strings.TrimSpace(c.Query("age_category")); ageCat != "" {
		if seq := getRefSeqnumByName(ageCat); seq != nil {
			query = query.Where("user_profiles.age_category_id = ?", *seq)
		}
	} else if ageCatID := strings.TrimSpace(c.Query("age_category_id")); ageCatID != "" {
		query = query.Where("user_profiles.age_category_id = ?", ageCatID)
	}

	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + search + "%"
		query = query.Where(
			"TRIM(COALESCE(user_profiles.first_name, '') || ' ' || COALESCE(user_profiles.last_name, '')) ILIKE ? OR "+
				"user_profiles.first_name ILIKE ? OR user_profiles.last_name ILIKE ?",
			like, like, like,
		)
	}

	var players []models.User
	query.Order("user_profiles.last_name ASC, user_profiles.first_name ASC").Find(&players)

	refMap := getRefMap()
	result := make([]fiber.Map, 0, len(players))
	for _, p := range players {
		firstName, middleName, lastName, avatar := "", "", "", ""
		ageCatName, posName, genderName := "", "", ""
		var jersey *int
		if p.Profile != nil {
			firstName = p.Profile.FirstName
			middleName = p.Profile.MiddleName
			lastName = p.Profile.LastName
			avatar = p.Profile.AvatarURL
			jersey = p.Profile.JerseyNumber
			if p.Profile.AgeCategoryID != nil {
				ageCatName = refMap[*p.Profile.AgeCategoryID]
			}
			if p.Profile.PositionID != nil {
				posName = refMap[*p.Profile.PositionID]
			}
			if p.Profile.GenderID != nil {
				genderName = refMap[*p.Profile.GenderID]
			}
		}
		fullName := strings.TrimSpace(firstName + " " + lastName)
		if fullName == "" {
			fullName = p.Username
		}
		result = append(result, fiber.Map{
			"user_id":       p.UserID,
			"username":      p.Username,
			"email":         p.Email,
			"role_id":       p.RoleID,
			"status_id":     p.StatusID,
			"first_name":    firstName,
			"middle_name":   middleName,
			"last_name":     lastName,
			"full_name":     fullName,
			"jersey_number": jersey,
			"avatar_url":    avatar,
			"profile":       p.Profile,
			"gender":        genderName,
			"age_category":  ageCatName,
			"position":      posName,
		})
	}

	return c.JSON(result)
}

func CreatePlayer(c *fiber.Ctx) error {
	var req struct {
		Username     string  `json:"username"`
		Email        string  `json:"email"`
		Password     string  `json:"password"`
		FirstName    string  `json:"first_name"`
		MiddleName   string  `json:"middle_name"`
		LastName     string  `json:"last_name"`
		Birthdate    *string `json:"birthdate"`
		Gender       string  `json:"gender"`
		AgeCategory  string  `json:"age_category"`
		Position     string  `json:"position"`
		JerseyNumber *int    `json:"jersey_number"`
		PhoneNumber  string  `json:"phone_number"`
		Address      string  `json:"address"`
		AvatarURL    string  `json:"avatar_url"`
		ParentUserID *int    `json:"parent_user_id"`
		ParentName   string  `json:"parent_name"`
		Relationship string  `json:"relationship"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	if req.FirstName == "" || req.LastName == "" {
		return c.Status(400).JSON(fiber.Map{"error": "first_name and last_name are required"})
	}

	suffix := time.Now().UnixNano() % 100000
	if strings.TrimSpace(req.Username) == "" {
		req.Username = fmt.Sprintf("%s_%s_%d", strings.ToLower(req.FirstName), strings.ToLower(req.LastName), suffix)
		req.Username = strings.ReplaceAll(req.Username, " ", "")
	}
	if strings.TrimSpace(req.Email) == "" {
		req.Email = req.Username + "@alabfc.local"
	}
	if req.Password == "" {
		req.Password = "alabfc123"
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to hash password"})
	}

	roleID := models.RefRolePlayer
	if seq := getRefSeqnumByName("Player"); seq != nil {
		roleID = *seq
	}
	statusID := models.RefAccountActive
	if seq := getRefSeqnumByName("Active"); seq != nil {
		statusID = *seq
	}

	user := models.User{
		Username:     strings.TrimSpace(req.Username),
		Email:        strings.TrimSpace(req.Email),
		PasswordHash: hashedPassword,
		RoleID:       roleID,
		StatusID:     statusID,
		IsVerified:   true,
	}

	db := database.Get()
	if err := db.Create(&user).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Username or email already exists"})
	}

	profile := models.UserProfile{
		UserID:        user.UserID,
		FirstName:     req.FirstName,
		MiddleName:    req.MiddleName,
		LastName:      req.LastName,
		Birthdate:     req.Birthdate,
		GenderID:      getRefSeqnumByName(req.Gender),
		AgeCategoryID: getRefSeqnumByName(req.AgeCategory),
		PositionID:    getRefSeqnumByName(req.Position),
		JerseyNumber:  req.JerseyNumber,
		PhoneNumber:   req.PhoneNumber,
		Address:       req.Address,
		AvatarURL:     req.AvatarURL,
	}
	db.Create(&profile)

	parentID := req.ParentUserID
	if parentID == nil && strings.TrimSpace(req.ParentName) != "" {
		if pid, err := findUserIDByName(req.ParentName, models.RefRoleParent); err == nil {
			parentID = &pid
		}
	}
	if parentID == nil {
		if rID, ok := c.Locals("role_id").(int); ok && rID == models.RefRoleParent {
			if uID, ok := c.Locals("user_id").(int); ok && uID > 0 {
				parentID = &uID
			}
		}
	}
	if parentID != nil && *parentID > 0 {
		link := models.ParentPlayer{
			ParentUserID:   *parentID,
			PlayerUserID:   user.UserID,
			RelationshipID: getRefSeqnumByName(req.Relationship),
		}
		db.Create(&link)
	}

	db.Preload("Profile").First(&user, user.UserID)
	return c.Status(201).JSON(fiber.Map{
		"message": "Player created",
		"data":    user,
	})
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
	var req struct {
		ParentUserID   int      `json:"parent_user_id"`
		ParentName     string   `json:"parent_name"`
		PlayerUserID   int      `json:"player_user_id"`
		PlayerUserIDs  []int    `json:"player_user_ids"`
		PlayerName     string   `json:"player_name"`
		PlayerNames    []string `json:"player_names"`
		RelationshipID *int     `json:"relationship_id"`
		Relationship   string   `json:"relationship"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	parentID := req.ParentUserID
	if parentID == 0 && strings.TrimSpace(req.ParentName) != "" {
		pid, err := findUserIDByName(req.ParentName, models.RefRoleParent)
		if err != nil {
			pid, err = findUserIDByName(req.ParentName, 0)
			if err != nil {
				return c.Status(404).JSON(fiber.Map{"error": "Parent not found: " + req.ParentName})
			}
		}
		parentID = pid
	}

	if parentID == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "parent_name or parent_user_id is required"})
	}

	relID := req.RelationshipID
	if relID == nil && strings.TrimSpace(req.Relationship) != "" {
		relID = getRefSeqnumByName(req.Relationship)
	}

	playerIDs := make([]int, 0)
	if req.PlayerUserID > 0 {
		playerIDs = append(playerIDs, req.PlayerUserID)
	}
	for _, id := range req.PlayerUserIDs {
		if id > 0 {
			playerIDs = append(playerIDs, id)
		}
	}
	if strings.TrimSpace(req.PlayerName) != "" {
		if pid, err := findUserIDByName(req.PlayerName, models.RefRolePlayer); err == nil {
			playerIDs = append(playerIDs, pid)
		}
	}
	for _, pName := range req.PlayerNames {
		if pid, err := findUserIDByName(pName, models.RefRolePlayer); err == nil {
			playerIDs = append(playerIDs, pid)
		}
	}

	if len(playerIDs) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "At least one player is required"})
	}

	db := database.Get()
	linked := make([]models.ParentPlayer, 0)
	for _, pid := range playerIDs {
		var existing models.ParentPlayer
		if err := db.Where("parent_user_id = ? AND player_user_id = ?", parentID, pid).First(&existing).Error; err == nil {
			if relID != nil {
				existing.RelationshipID = relID
				db.Save(&existing)
			}
			linked = append(linked, existing)
			continue
		}

		link := models.ParentPlayer{
			ParentUserID:   parentID,
			PlayerUserID:   pid,
			RelationshipID: relID,
		}
		if err := db.Create(&link).Error; err == nil {
			linked = append(linked, link)
		}
	}

	var parentUser models.User
	db.Preload("Profile").First(&parentUser, parentID)
	parentFullName := parentUser.Username
	if parentUser.Profile != nil {
		if n := strings.TrimSpace(parentUser.Profile.FirstName + " " + parentUser.Profile.LastName); n != "" {
			parentFullName = n
		}
	}

	return c.Status(201).JSON(fiber.Map{
		"message":        "Parent and player(s) linked",
		"parent_user_id": parentID,
		"parent_name":    parentFullName,
		"linked_count":   len(linked),
		"data":           linked,
	})
}

func GetParentPlayers(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(int)
	roleID, _ := c.Locals("role_id").(int)
	parentID := c.Params("parent_id")

	if roleID == models.RefRoleParent && fmt.Sprintf("%d", userID) != parentID {
		return c.Status(403).JSON(fiber.Map{"error": "Forbidden: you can only view your own children"})
	}

	var links []models.ParentPlayer
	database.Get().
		Preload("Player.Profile").
		Where("parent_user_id = ?", parentID).
		Find(&links)

	return c.JSON(links)
}

func GetMyPlayers(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(int)
	refMap := getRefMap()

	var links []models.ParentPlayer
	database.Get().
		Preload("Player.Profile").
		Where("parent_user_id = ?", userID).
		Find(&links)

	result := make([]fiber.Map, 0, len(links))
	for _, l := range links {
		relName := ""
		if l.RelationshipID != nil {
			relName = refMap[*l.RelationshipID]
		}
		ageCat, pos, gender := "", "", ""
		firstName, lastName := "", ""
		if l.Player != nil && l.Player.Profile != nil {
			firstName = l.Player.Profile.FirstName
			lastName = l.Player.Profile.LastName
			if l.Player.Profile.AgeCategoryID != nil {
				ageCat = refMap[*l.Player.Profile.AgeCategoryID]
			}
			if l.Player.Profile.PositionID != nil {
				pos = refMap[*l.Player.Profile.PositionID]
			}
			if l.Player.Profile.GenderID != nil {
				gender = refMap[*l.Player.Profile.GenderID]
			}
		}
		result = append(result, fiber.Map{
			"link_id":        l.ID,
			"player_user_id": l.PlayerUserID,
			"first_name":     firstName,
			"last_name":      lastName,
			"full_name":      strings.TrimSpace(firstName + " " + lastName),
			"relationship":   relName,
			"player":         l.Player,
			"gender":         gender,
			"age_category":   ageCat,
			"position":       pos,
		})
	}

	return c.JSON(result)
}

func UnlinkParentPlayer(c *fiber.Ctx) error {
	database.Get().Delete(&models.ParentPlayer{}, c.Params("id"))
	return c.JSON(fiber.Map{"message": "Parent and player unlinked"})
}
