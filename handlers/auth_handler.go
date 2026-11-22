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

func getUserTOTPSecret(userID int) string {
	var otp models.OTPVerification
	err := database.Get().
		Where("user_id = ? AND otp_code = ?", userID, "TOTP").
		Order("otp_id DESC").
		First(&otp).Error
	if err != nil {
		return ""
	}
	return otp.Purpose
}

func Register(c *fiber.Ctx) error {
	var req struct {
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

	secret, err := utils.GenerateTOTPSecret()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate authenticator secret"})
	}

	user := models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		RoleID:       roleID,
		StatusID:     statusID,
		IsVerified:   false,
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

	totpRecord := models.OTPVerification{
		UserID:    user.UserID,
		OTPCode:   "TOTP",
		Purpose:   secret,
		ExpiresAt: time.Now().AddDate(10, 0, 0),
	}
	db.Create(&totpRecord)

	otpauthURL, qrCodeURL := utils.BuildTOTPQRCode(user.Email, secret)

	return c.Status(201).JSON(fiber.Map{
		"message":     "Registration successful",
		"user_id":     user.UserID,
		"secret":      secret,
		"otpauth_url": otpauthURL,
		"qr_code_url": qrCodeURL,
	})
}

func VerifyAuthenticator(c *fiber.Ctx) error {
	var req struct {
		UsernameOrEmail string `json:"username_or_email"`
		OTPCode         string `json:"otp_code"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	db := database.Get()
	var user models.User
	err := db.Where("username = ? OR email = ?", req.UsernameOrEmail, req.UsernameOrEmail).First(&user).Error
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Account not found"})
	}

	secret := getUserTOTPSecret(user.UserID)
	if secret == "" || !utils.VerifyTOTPCode(secret, req.OTPCode) {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid authenticator OTP code"})
	}

	db.Model(&user).Update("is_verified", true)

	return c.JSON(fiber.Map{
		"message": "Authenticator verified successfully",
	})
}

func Login(c *fiber.Ctx) error {
	var req struct {
		UsernameOrEmail string `json:"username_or_email"`
		Password        string `json:"password"`
		OTPCode         string `json:"otp_code"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	db := database.Get()
	var user models.User
	err := db.Where("username = ? OR email = ?", req.UsernameOrEmail, req.UsernameOrEmail).First(&user).Error
	if err != nil || !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid username/email or password"})
	}

	if req.OTPCode != "" {
		secret := getUserTOTPSecret(user.UserID)
		if secret != "" && !utils.VerifyTOTPCode(secret, req.OTPCode) {
			return c.Status(401).JSON(fiber.Map{"error": "Invalid authenticator OTP code"})
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
		UsernameOrEmail string `json:"username_or_email"`
		OTPCode         string `json:"otp_code"`
		NewPassword     string `json:"new_password"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.UsernameOrEmail == "" || req.OTPCode == "" || req.NewPassword == "" {
		return c.Status(400).JSON(fiber.Map{"error": "username_or_email, otp_code, and new_password are required"})
	}

	db := database.Get()
	var user models.User
	if err := db.Where("username = ? OR email = ?", req.UsernameOrEmail, req.UsernameOrEmail).First(&user).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Account not found"})
	}

	secret := getUserTOTPSecret(user.UserID)
	if secret == "" || !utils.VerifyTOTPCode(secret, req.OTPCode) {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid authenticator OTP code"})
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
