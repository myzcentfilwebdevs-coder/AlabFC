package handlers

import (
	"strings"
	"time"

	"alabapi/database"
	"alabapi/models"
	"alabapi/utils"

	"github.com/gofiber/fiber/v2"
)

func getRefSeqnumByName(details string) *int {
	details = strings.TrimSpace(details)
	if details == "" {
		return nil
	}

	var ref models.GenericReference
	if err := database.Get().Where("LOWER(details) = LOWER(?)", details).First(&ref).Error; err != nil {
		return nil
	}
	return &ref.Seqnum
}

func findOrCreateGoogleUser(gUser *utils.GoogleUser) (models.User, error) {
	db := database.Get()
	var user models.User

	err := db.Where("LOWER(email) = LOWER(?)", gUser.Email).First(&user).Error
	if err != nil {
		roleID := models.RefRoleParent
		if seq := getRefSeqnumByName("Parent"); seq != nil {
			roleID = *seq
		}
		statusID := models.RefAccountActive
		if seq := getRefSeqnumByName("Active"); seq != nil {
			statusID = *seq
		}

		hashedPassword, _ := utils.HashPassword(gUser.Sub)
		username := strings.Split(gUser.Email, "@")[0]

		user = models.User{
			Username:     username,
			Email:        gUser.Email,
			PasswordHash: hashedPassword,
			RoleID:       roleID,
			StatusID:     statusID,
			IsVerified:   true,
		}
		if err := db.Create(&user).Error; err != nil {
			suffix := gUser.Sub
			if len(suffix) > 4 {
				suffix = suffix[:4]
			}
			user.Username = username + "_" + suffix
			if err := db.Create(&user).Error; err != nil {
				return user, err
			}
		}

		profile := models.UserProfile{
			UserID:    user.UserID,
			FirstName: gUser.GivenName,
			LastName:  gUser.FamilyName,
			AvatarURL: gUser.Picture,
		}
		db.Create(&profile)
	}

	now := time.Now()
	db.Model(&user).Updates(map[string]interface{}{
		"last_login_at": now,
		"is_verified":   true,
	})

	return user, nil
}

func GoogleLoginRedirect(c *fiber.Ctx) error {
	url := utils.GetGoogleLoginURL("alabfc")
	return c.Redirect(url, fiber.StatusTemporaryRedirect)
}

func GoogleCallback(c *fiber.Ctx) error {
	code := c.Query("code")
	if code == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Missing code from Google"})
	}

	gUser, err := utils.ExchangeGoogleCode(code)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": err.Error()})
	}

	user, err := findOrCreateGoogleUser(gUser)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to save Google user"})
	}

	token, err := utils.GenerateJWT(user.UserID, user.Username, user.Email, user.RoleID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate token"})
	}

	return c.JSON(fiber.Map{
		"message": "Login successful",
		"token":   token,
	})
}

func GoogleAuth(c *fiber.Ctx) error {
	var req struct {
		GoogleToken string `json:"google_token"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	gUser, err := utils.VerifyGoogleToken(req.GoogleToken)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": err.Error()})
	}

	user, err := findOrCreateGoogleUser(gUser)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to save Google user"})
	}

	token, err := utils.GenerateJWT(user.UserID, user.Username, user.Email, user.RoleID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate token"})
	}

	return c.JSON(fiber.Map{
		"message": "Login successful",
		"token":   token,
	})
}

func Register(c *fiber.Ctx) error {
	var req struct {
		GoogleToken string  `json:"google_token"`
		Username    string  `json:"username"`
		Email       string  `json:"email"`
		Password    string  `json:"password"`
		FirstName   string  `json:"first_name"`
		MiddleName  string  `json:"middle_name"`
		LastName    string  `json:"last_name"`
		Birthdate   *string `json:"birthdate"`
		Gender      string  `json:"gender"`
		PhoneNumber string  `json:"phone_number"`
		Address     string  `json:"address"`
		AvatarURL   string  `json:"avatar_url"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	isVerified := false

	if req.GoogleToken != "" {
		gUser, err := utils.VerifyGoogleToken(req.GoogleToken)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"error": err.Error()})
		}
		req.Email = gUser.Email
		if req.FirstName == "" {
			req.FirstName = gUser.GivenName
		}
		if req.LastName == "" {
			req.LastName = gUser.FamilyName
		}
		if req.AvatarURL == "" {
			req.AvatarURL = gUser.Picture
		}
		if req.Username == "" {
			req.Username = strings.Split(gUser.Email, "@")[0]
		}
		if req.Password == "" {
			req.Password = gUser.Sub
		}
		isVerified = true
	}

	roleID := models.RefRoleParent
	if seq := getRefSeqnumByName("Parent"); seq != nil {
		roleID = *seq
	}

	statusID := models.RefAccountActive
	if seq := getRefSeqnumByName("Active"); seq != nil {
		statusID = *seq
	}

	genderID := getRefSeqnumByName(req.Gender)
	if req.Gender != "" && genderID == nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid gender value in genericreference"})
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to hash password"})
	}

	user := models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		RoleID:       roleID,
		StatusID:     statusID,
		IsVerified:   isVerified,
	}

	db := database.Get()
	if err := db.Create(&user).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Username or email already exists"})
	}

	profile := models.UserProfile{
		UserID:      user.UserID,
		FirstName:   req.FirstName,
		MiddleName:  req.MiddleName,
		LastName:    req.LastName,
		Birthdate:   req.Birthdate,
		GenderID:    genderID,
		PhoneNumber: req.PhoneNumber,
		Address:     req.Address,
		AvatarURL:   req.AvatarURL,
	}
	db.Create(&profile)

	return c.Status(201).JSON(fiber.Map{
		"message": "Registration successful",
		"user_id": user.UserID,
	})
}

func Login(c *fiber.Ctx) error {
	var req struct {
		GoogleToken     string `json:"google_token"`
		UsernameOrEmail string `json:"username_or_email"`
		Password        string `json:"password"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	db := database.Get()
	var user models.User

	if req.GoogleToken != "" {
		gUser, err := utils.VerifyGoogleToken(req.GoogleToken)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"error": err.Error()})
		}
		if err := db.Where("LOWER(email) = LOWER(?)", gUser.Email).First(&user).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Account not found. Please register first"})
		}
	} else {
		err := db.Where("username = ? OR email = ?", req.UsernameOrEmail, req.UsernameOrEmail).First(&user).Error
		if err != nil || !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
			return c.Status(401).JSON(fiber.Map{"error": "Invalid username/email or password"})
		}
	}

	now := time.Now()
	db.Model(&user).Update("last_login_at", now)

	token, err := utils.GenerateJWT(user.UserID, user.Username, user.Email, user.RoleID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate token"})
	}

	return c.JSON(fiber.Map{
		"message": "Login successful",
		"token":   token,
	})
}

func ForgotPassword(c *fiber.Ctx) error {
	var req struct {
		GoogleToken string `json:"google_token"`
		NewPassword string `json:"new_password"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.GoogleToken == "" || req.NewPassword == "" {
		return c.Status(400).JSON(fiber.Map{"error": "google_token and new_password are required"})
	}

	gUser, err := utils.VerifyGoogleToken(req.GoogleToken)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": err.Error()})
	}

	db := database.Get()
	var user models.User
	if err := db.Where("LOWER(email) = LOWER(?)", gUser.Email).First(&user).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Account not found"})
	}

	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to hash password"})
	}

	db.Model(&user).Updates(map[string]interface{}{
		"password_hash": hashedPassword,
		"is_verified":   true,
		"updated_at":    time.Now(),
	})

	return c.JSON(fiber.Map{
		"message": "Password updated successfully",
	})
}

func GetMe(c *fiber.Ctx) error {
	userID := c.Locals("user_id")
	var user models.User

	if err := database.Get().Preload("Profile").First(&user, userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}

	return c.JSON(user)
}
