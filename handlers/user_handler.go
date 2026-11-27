package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"alabapi/database"
	"alabapi/models"

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

func findPlayerIDByName(name string) (int, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("player name is empty")
	}

	db := database.Get()
	var player models.Player

	err := db.Where(
		"LOWER(TRIM(COALESCE(first_name, '') || ' ' || COALESCE(last_name, ''))) = LOWER(?) OR "+
			"LOWER(TRIM(COALESCE(first_name, '') || ' ' || COALESCE(middle_name, '') || ' ' || COALESCE(last_name, ''))) = LOWER(?)",
		name, name,
	).First(&player).Error
	if err == nil {
		return player.PlayerID, nil
	}

	like := "%" + name + "%"
	err = db.Where(
		"TRIM(COALESCE(first_name, '') || ' ' || COALESCE(last_name, '')) ILIKE ? OR first_name ILIKE ? OR last_name ILIKE ?",
		like, like, like,
	).First(&player).Error
	if err != nil {
		return 0, fmt.Errorf("player %q not found", name)
	}
	return player.PlayerID, nil
}

func formatPlayer(p models.Player, refMap map[int]string) fiber.Map {
	genderName, ageCatName, posName, statusName := "", "", "", ""
	if p.GenderID != nil {
		genderName = refMap[*p.GenderID]
	}
	if p.AgeCategoryID != nil {
		ageCatName = refMap[*p.AgeCategoryID]
	}
	if p.PositionID != nil {
		posName = refMap[*p.PositionID]
	}
	statusName = refMap[p.StatusID]

	var birthdate *string
	if p.Birthdate != nil {
		bd := *p.Birthdate
		if len(bd) >= 10 {
			bd = bd[:10]
		}
		birthdate = &bd
	}

	return fiber.Map{
		"player_id":       p.PlayerID,
		"first_name":      p.FirstName,
		"middle_name":     p.MiddleName,
		"last_name":       p.LastName,
		"full_name":       strings.TrimSpace(p.FirstName + " " + p.LastName),
		"birthdate":       birthdate,
		"gender_id":       p.GenderID,
		"gender":          genderName,
		"age_category_id": p.AgeCategoryID,
		"age_category":    ageCatName,
		"position_id":     p.PositionID,
		"position":        posName,
		"jersey_number":   p.JerseyNumber,
		"avatar_url":      p.AvatarURL,
		"status_id":       p.StatusID,
		"status":          statusName,
		"created_at":      p.CreatedAt,
	}
}

func GetUsers(c *fiber.Ctx) error {
	db := database.Get()
	query := db.Model(&models.User{}).
		Preload("Profile").
		Joins("LEFT JOIN user_profiles ON user_profiles.user_id = users.user_id")

	if role := strings.TrimSpace(c.Query("role")); role != "" {
		if seq := getRefSeqnumByName(role); seq != nil {
			query = query.Where("users.role_id = ?", *seq)
		}
	} else if roleID := strings.TrimSpace(c.Query("role_id")); roleID != "" {
		query = query.Where("users.role_id = ?", roleID)
	}

	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + search + "%"
		query = query.Where(
			"TRIM(COALESCE(user_profiles.first_name, '') || ' ' || COALESCE(user_profiles.last_name, '')) ILIKE ? OR "+
				"users.username ILIKE ? OR users.email ILIKE ?",
			like, like, like,
		)
	}

	var users []models.User
	query.Order("users.user_id ASC").Find(&users)

	refMap := getRefMap()
	result := make([]fiber.Map, 0, len(users))
	for _, u := range users {
		firstName, lastName, phone, avatar := "", "", "", ""
		if u.Profile != nil {
			firstName = u.Profile.FirstName
			lastName = u.Profile.LastName
			phone = u.Profile.PhoneNumber
			avatar = u.Profile.AvatarURL
		}
		fullName := strings.TrimSpace(firstName + " " + lastName)
		if fullName == "" {
			fullName = u.Username
		}
		result = append(result, fiber.Map{
			"user_id":       u.UserID,
			"username":      u.Username,
			"email":         u.Email,
			"first_name":    firstName,
			"last_name":     lastName,
			"full_name":     fullName,
			"phone_number":  phone,
			"avatar_url":    avatar,
			"role_id":       u.RoleID,
			"role_name":     refMap[u.RoleID],
			"status_id":     u.StatusID,
			"status_name":   refMap[u.StatusID],
			"is_verified":   u.IsVerified,
			"last_login_at": u.LastLoginAt,
			"created_at":    u.CreatedAt,
			"profile":       u.Profile,
		})
	}

	return c.JSON(result)
}

func UpdateUserRole(c *fiber.Ctx) error {
	db := database.Get()
	var user models.User
	if err := db.Preload("Profile").First(&user, c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}

	var req struct {
		RoleID   *int   `json:"role_id"`
		Role     string `json:"role"`
		StatusID *int   `json:"status_id"`
		Status   string `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	newRoleID := user.RoleID
	if req.RoleID != nil && *req.RoleID > 0 {
		newRoleID = *req.RoleID
	} else if strings.TrimSpace(req.Role) != "" {
		seq := getRefSeqnumByName(req.Role)
		if seq == nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid role name"})
		}
		newRoleID = *seq
	}

	loggedInID, _ := c.Locals("user_id").(int)
	if user.UserID == loggedInID && newRoleID != models.RefRoleAdmin {
		return c.Status(400).JSON(fiber.Map{"error": "You cannot remove your own Admin role"})
	}

	newStatusID := user.StatusID
	if req.StatusID != nil && *req.StatusID > 0 {
		newStatusID = *req.StatusID
	} else if strings.TrimSpace(req.Status) != "" {
		seq := getRefSeqnumByName(req.Status)
		if seq == nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid status name"})
		}
		newStatusID = *seq
	}

	db.Model(&user).Updates(map[string]interface{}{
		"role_id":    newRoleID,
		"status_id":  newStatusID,
		"updated_at": time.Now(),
	})

	refMap := getRefMap()
	return c.JSON(fiber.Map{
		"message":     "User role updated",
		"user_id":     user.UserID,
		"username":    user.Username,
		"role_id":     newRoleID,
		"role_name":   refMap[newRoleID],
		"status_id":   newStatusID,
		"status_name": refMap[newStatusID],
	})
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
	query := db.Model(&models.Player{})

	if ageCat := strings.TrimSpace(c.Query("age_category")); ageCat != "" {
		if seq := getRefSeqnumByName(ageCat); seq != nil {
			query = query.Where("age_category_id = ?", *seq)
		}
	} else if ageCatID := strings.TrimSpace(c.Query("age_category_id")); ageCatID != "" {
		query = query.Where("age_category_id = ?", ageCatID)
	}

	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + search + "%"
		query = query.Where(
			"TRIM(COALESCE(first_name, '') || ' ' || COALESCE(last_name, '')) ILIKE ? OR first_name ILIKE ? OR last_name ILIKE ?",
			like, like, like,
		)
	}

	var players []models.Player
	query.Order("last_name ASC, first_name ASC").Find(&players)

	refMap := getRefMap()
	result := make([]fiber.Map, 0, len(players))
	for _, p := range players {
		result = append(result, formatPlayer(p, refMap))
	}

	return c.JSON(result)
}

func CreatePlayer(c *fiber.Ctx) error {
	var req struct {
		FirstName     string  `json:"first_name"`
		MiddleName    string  `json:"middle_name"`
		LastName      string  `json:"last_name"`
		Birthdate     *string `json:"birthdate"`
		Gender        string  `json:"gender"`
		GenderID      *int    `json:"gender_id"`
		AgeCategory   string  `json:"age_category"`
		AgeCategoryID *int    `json:"age_category_id"`
		Position      string  `json:"position"`
		PositionID    *int    `json:"position_id"`
		JerseyNumber  *int    `json:"jersey_number"`
		AvatarURL     string  `json:"avatar_url"`
		ParentUserID  *int    `json:"parent_user_id"`
		ParentName    string  `json:"parent_name"`
		Relationship  string  `json:"relationship"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	if req.FirstName == "" || req.LastName == "" {
		return c.Status(400).JSON(fiber.Map{"error": "first_name and last_name are required"})
	}

	genderID := req.GenderID
	if genderID == nil {
		genderID = getRefSeqnumByName(req.Gender)
	}
	ageCatID := req.AgeCategoryID
	if ageCatID == nil {
		ageCatID = getRefSeqnumByName(req.AgeCategory)
	}
	posID := req.PositionID
	if posID == nil {
		posID = getRefSeqnumByName(req.Position)
	}

	statusID := models.RefAccountActive
	if seq := getRefSeqnumByName("Active"); seq != nil {
		statusID = *seq
	}

	player := models.Player{
		FirstName:     req.FirstName,
		MiddleName:    strings.TrimSpace(req.MiddleName),
		LastName:      req.LastName,
		Birthdate:     req.Birthdate,
		GenderID:      genderID,
		AgeCategoryID: ageCatID,
		PositionID:    posID,
		JerseyNumber:  req.JerseyNumber,
		AvatarURL:     strings.TrimSpace(req.AvatarURL),
		StatusID:      statusID,
	}

	db := database.Get()
	if err := db.Create(&player).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

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
			PlayerID:       player.PlayerID,
			RelationshipID: getRefSeqnumByName(req.Relationship),
		}
		db.Create(&link)
	}

	return c.Status(201).JSON(fiber.Map{
		"message": "Player created",
		"data":    formatPlayer(player, getRefMap()),
	})
}

func UpdatePlayer(c *fiber.Ctx) error {
	db := database.Get()
	var player models.Player
	if err := db.First(&player, c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Player not found"})
	}

	var req struct {
		FirstName     string  `json:"first_name"`
		MiddleName    string  `json:"middle_name"`
		LastName      string  `json:"last_name"`
		Birthdate     *string `json:"birthdate"`
		Gender        string  `json:"gender"`
		GenderID      *int    `json:"gender_id"`
		AgeCategory   string  `json:"age_category"`
		AgeCategoryID *int    `json:"age_category_id"`
		Position      string  `json:"position"`
		PositionID    *int    `json:"position_id"`
		JerseyNumber  *int    `json:"jersey_number"`
		AvatarURL     string  `json:"avatar_url"`
		Status        string  `json:"status"`
		StatusID      *int    `json:"status_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if strings.TrimSpace(req.FirstName) != "" {
		player.FirstName = strings.TrimSpace(req.FirstName)
	}
	if req.MiddleName != "" {
		player.MiddleName = strings.TrimSpace(req.MiddleName)
	}
	if strings.TrimSpace(req.LastName) != "" {
		player.LastName = strings.TrimSpace(req.LastName)
	}
	if req.Birthdate != nil {
		player.Birthdate = req.Birthdate
	}
	if req.GenderID != nil {
		player.GenderID = req.GenderID
	} else if strings.TrimSpace(req.Gender) != "" {
		player.GenderID = getRefSeqnumByName(req.Gender)
	}
	if req.AgeCategoryID != nil {
		player.AgeCategoryID = req.AgeCategoryID
	} else if strings.TrimSpace(req.AgeCategory) != "" {
		player.AgeCategoryID = getRefSeqnumByName(req.AgeCategory)
	}
	if req.PositionID != nil {
		player.PositionID = req.PositionID
	} else if strings.TrimSpace(req.Position) != "" {
		player.PositionID = getRefSeqnumByName(req.Position)
	}
	if req.JerseyNumber != nil {
		player.JerseyNumber = req.JerseyNumber
	}
	if req.AvatarURL != "" {
		player.AvatarURL = strings.TrimSpace(req.AvatarURL)
	}
	if req.StatusID != nil {
		player.StatusID = *req.StatusID
	} else if strings.TrimSpace(req.Status) != "" {
		if seq := getRefSeqnumByName(req.Status); seq != nil {
			player.StatusID = *seq
		}
	}
	player.UpdatedAt = time.Now()

	db.Save(&player)
	return c.JSON(fiber.Map{
		"message": "Player updated",
		"data":    formatPlayer(player, getRefMap()),
	})
}

func DeletePlayer(c *fiber.Ctx) error {
	playerID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid player_id"})
	}

	res := database.Get().Where("player_id = ?", playerID).Delete(&models.Player{})
	if res.Error != nil {
		return c.Status(500).JSON(fiber.Map{"error": res.Error.Error()})
	}
	if res.RowsAffected == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "Player not found"})
	}

	return c.JSON(fiber.Map{
		"message":   "Player deleted",
		"player_id": playerID,
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
		PlayerID       int      `json:"player_id"`
		PlayerIDs      []int    `json:"player_ids"`
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

	seen := make(map[int]bool)
	playerIDs := make([]int, 0)
	addPlayer := func(id int) {
		if id > 0 && !seen[id] {
			seen[id] = true
			playerIDs = append(playerIDs, id)
		}
	}

	addPlayer(req.PlayerID)
	addPlayer(req.PlayerUserID)
	for _, id := range req.PlayerIDs {
		addPlayer(id)
	}
	for _, id := range req.PlayerUserIDs {
		addPlayer(id)
	}
	if strings.TrimSpace(req.PlayerName) != "" {
		if pid, err := findPlayerIDByName(req.PlayerName); err == nil {
			addPlayer(pid)
		}
	}
	for _, pName := range req.PlayerNames {
		if pid, err := findPlayerIDByName(pName); err == nil {
			addPlayer(pid)
		}
	}

	if len(playerIDs) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "At least one player is required"})
	}

	db := database.Get()
	linked := make([]models.ParentPlayer, 0)
	for _, pid := range playerIDs {
		var existing models.ParentPlayer
		if err := db.Where("parent_user_id = ? AND player_id = ?", parentID, pid).First(&existing).Error; err == nil {
			if relID != nil {
				existing.RelationshipID = relID
				db.Save(&existing)
			}
			linked = append(linked, existing)
			continue
		}

		link := models.ParentPlayer{
			ParentUserID:   parentID,
			PlayerID:       pid,
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
		Preload("Player").
		Where("parent_user_id = ?", parentID).
		Find(&links)

	return c.JSON(links)
}

func GetMyPlayers(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(int)
	refMap := getRefMap()

	var links []models.ParentPlayer
	database.Get().
		Preload("Player").
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
		var jersey *int
		if l.Player != nil {
			firstName = l.Player.FirstName
			lastName = l.Player.LastName
			jersey = l.Player.JerseyNumber
			if l.Player.AgeCategoryID != nil {
				ageCat = refMap[*l.Player.AgeCategoryID]
			}
			if l.Player.PositionID != nil {
				pos = refMap[*l.Player.PositionID]
			}
			if l.Player.GenderID != nil {
				gender = refMap[*l.Player.GenderID]
			}
		}
		result = append(result, fiber.Map{
			"link_id":       l.ID,
			"player_id":     l.PlayerID,
			"first_name":    firstName,
			"last_name":     lastName,
			"full_name":     strings.TrimSpace(firstName + " " + lastName),
			"jersey_number": jersey,
			"relationship":  relName,
			"player":        l.Player,
			"gender":        gender,
			"age_category":  ageCat,
			"position":      pos,
		})
	}

	return c.JSON(result)
}

func UnlinkParentPlayer(c *fiber.Ctx) error {
	database.Get().Delete(&models.ParentPlayer{}, c.Params("id"))
	return c.JSON(fiber.Map{"message": "Parent and player unlinked"})
}
