package handlers

import (
	"log"
	"net/mail"
	"os"
	"strings"
	"sync"
	"time"

	"alabapi/database"
	"alabapi/models"
	"alabapi/utils"

	"github.com/gofiber/fiber/v2"
)

type PendingRegistration struct {
	Username      string
	Email         string
	PasswordHash  string
	RoleID        int
	StatusID      int
	FirstName     string
	MiddleName    string
	LastName      string
	Birthdate     *string
	GenderID      *int
	PhoneNumber   string
	Address       string
	AvatarURL     string
	Secret        string
	EmailToken    string
	EmailVerified bool
	CreatedAt     time.Time
}

var (
	pendingMu            sync.RWMutex
	pendingRegistrations = make(map[string]*PendingRegistration)
	pendingByToken       = make(map[string]*PendingRegistration)
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

func resolveAppBaseURL(c *fiber.Ctx) string {
	base := strings.Trim(strings.TrimSpace(os.Getenv("APP_BASE_URL")), "'\"")
	if base != "" {
		return strings.TrimRight(base, "/")
	}
	return strings.TrimRight(c.BaseURL(), "/")
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

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Username == "" || req.Email == "" || req.Password == "" {
		return c.Status(400).JSON(fiber.Map{"error": "username, email, and password are required"})
	}

	if _, err := mail.ParseAddress(req.Email); err != nil || !strings.Contains(req.Email, ".") {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid email address format"})
	}

	db := database.Get()
	var existingCount int64
	db.Model(&models.User{}).
		Where("LOWER(username) = LOWER(?) OR LOWER(email) = LOWER(?)", req.Username, req.Email).
		Count(&existingCount)
	if existingCount > 0 {
		return c.Status(400).JSON(fiber.Map{"error": "Username or email already exists"})
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

	emailToken, err := utils.GenerateEmailToken()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate verification token"})
	}

	verifyLink := resolveAppBaseURL(c) + "/api/auth/verify-email?token=" + emailToken
	if err := utils.SendVerificationEmail(req.Email, req.FirstName, verifyLink); err != nil {
		log.Printf("[AUTH] Email send FAILED to %s: %v", req.Email, err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to send verification email: " + err.Error()})
	}
	log.Printf("[AUTH] Verification email SENT to %s", req.Email)

	pending := &PendingRegistration{
		Username:      req.Username,
		Email:         req.Email,
		PasswordHash:  hashedPassword,
		RoleID:        roleID,
		StatusID:      statusID,
		FirstName:     req.FirstName,
		MiddleName:    req.MiddleName,
		LastName:      req.LastName,
		Birthdate:     req.Birthdate,
		GenderID:      genderID,
		PhoneNumber:   req.PhoneNumber,
		Address:       req.Address,
		AvatarURL:     req.AvatarURL,
		Secret:        secret,
		EmailToken:    emailToken,
		EmailVerified: false,
		CreatedAt:     time.Now(),
	}

	pendingMu.Lock()
	pendingRegistrations[strings.ToLower(req.Username)] = pending
	pendingRegistrations[strings.ToLower(req.Email)] = pending
	pendingByToken[emailToken] = pending
	pendingMu.Unlock()

	return c.Status(200).JSON(fiber.Map{
		"message":        "Verification email sent. Please open your email and click Verify Account to unlock your QR code.",
		"username":       req.Username,
		"email":          req.Email,
		"email_verified": false,
	})
}

func VerifyEmailLink(c *fiber.Ctx) error {
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		return c.Status(400).Type("html").SendString("<h2>Invalid verification link</h2>")
	}

	pendingMu.Lock()
	pending, exists := pendingByToken[token]
	if exists {
		pending.EmailVerified = true
	}
	pendingMu.Unlock()

	if !exists {
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.Status(404).SendString(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><title>ALAB FC</title></head>
<body style="font-family: Arial, sans-serif; background:#070d1d; color:#fff; display:flex; align-items:center; justify-content:center; height:100vh; margin:0;">
  <div style="background:#0d152a; border:1px solid #ef4444; border-radius:16px; padding:32px; text-align:center; max-width:420px;">
    <h2 style="color:#f87171; margin-top:0;">Link Expired or Invalid</h2>
    <p style="color:#94a3b8;">This verification link is invalid or already used. Please register again.</p>
  </div>
</body>
</html>`)
	}

	frontendRedirect := strings.Trim(strings.TrimSpace(os.Getenv("FRONTEND_VERIFY_URL")), "'\"")
	if frontendRedirect != "" {
		sep := "?"
		if strings.Contains(frontendRedirect, "?") {
			sep = "&"
		}
		return c.Redirect(frontendRedirect+sep+"email_verified=true&email="+pending.Email, fiber.StatusTemporaryRedirect)
	}

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.Status(200).SendString(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><title>ALAB FC - Email Verified</title></head>
<body style="font-family: Arial, sans-serif; background:#070d1d; color:#fff; display:flex; align-items:center; justify-content:center; height:100vh; margin:0;">
  <div style="background:#0d152a; border:1px solid #22c55e; border-radius:16px; padding:36px; text-align:center; max-width:440px;">
    <h2 style="color:#4ade80; margin-top:0;">Email Verified!</h2>
    <p style="color:#cbd5e1; line-height:1.6;">Your email has been verified. You can now return to the ALAB FC registration page - your Google Authenticator QR Code is now ready to scan.</p>
  </div>
</body>
</html>`)
}

func GetRegistrationStatus(c *fiber.Ctx) error {
	key := strings.ToLower(strings.TrimSpace(c.Query("username_or_email")))
	if key == "" {
		key = strings.ToLower(strings.TrimSpace(c.Query("email")))
	}
	if key == "" {
		key = strings.ToLower(strings.TrimSpace(c.Query("username")))
	}
	if key == "" {
		return c.Status(400).JSON(fiber.Map{"error": "username_or_email is required"})
	}

	pendingMu.RLock()
	pending, exists := pendingRegistrations[key]
	var isVerified bool
	var email, secret string
	if exists {
		isVerified = pending.EmailVerified
		email = pending.Email
		secret = pending.Secret
	}
	pendingMu.RUnlock()

	if !exists {
		return c.Status(404).JSON(fiber.Map{"error": "No pending registration found"})
	}

	if !isVerified {
		return c.JSON(fiber.Map{
			"email_verified": false,
			"message":        "Waiting for email verification. Please click the Verify Account link in your email.",
		})
	}

	otpauthURL, qrCodeURL := utils.BuildTOTPQRCode(email, secret)
	return c.JSON(fiber.Map{
		"email_verified": true,
		"message":        "Email verified! Scan the QR code with Google Authenticator and enter the 6-digit OTP.",
		"secret":         secret,
		"otpauth_url":    otpauthURL,
		"qr_code_url":    qrCodeURL,
	})
}

func ResendVerificationEmail(c *fiber.Ctx) error {
	var req struct {
		UsernameOrEmail string `json:"username_or_email"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	key := strings.ToLower(strings.TrimSpace(req.UsernameOrEmail))
	pendingMu.RLock()
	pending, exists := pendingRegistrations[key]
	pendingMu.RUnlock()

	if !exists {
		return c.Status(404).JSON(fiber.Map{"error": "No pending registration found"})
	}

	verifyLink := resolveAppBaseURL(c) + "/api/auth/verify-email?token=" + pending.EmailToken
	if err := utils.SendVerificationEmail(pending.Email, pending.FirstName, verifyLink); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to resend verification email: " + err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Verification email resent",
		"email":   pending.Email,
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

	key := strings.ToLower(strings.TrimSpace(req.UsernameOrEmail))

	pendingMu.RLock()
	pending, exists := pendingRegistrations[key]
	pendingMu.RUnlock()

	if !exists {
		return c.Status(404).JSON(fiber.Map{"error": "No pending registration found for this username or email"})
	}

	if !pending.EmailVerified {
		return c.Status(403).JSON(fiber.Map{"error": "Please verify your email first by clicking the Verify Account link sent to your inbox"})
	}

	if !utils.VerifyTOTPCode(pending.Secret, req.OTPCode) {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid authenticator OTP code"})
	}

	db := database.Get()

	user := models.User{
		Username:     pending.Username,
		Email:        pending.Email,
		PasswordHash: pending.PasswordHash,
		RoleID:       pending.RoleID,
		StatusID:     pending.StatusID,
		IsVerified:   true,
	}

	if err := db.Create(&user).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Username or email already exists"})
	}

	profile := models.UserProfile{
		UserID:      user.UserID,
		FirstName:   pending.FirstName,
		MiddleName:  pending.MiddleName,
		LastName:    pending.LastName,
		Birthdate:   pending.Birthdate,
		GenderID:    pending.GenderID,
		PhoneNumber: pending.PhoneNumber,
		Address:     pending.Address,
		AvatarURL:   pending.AvatarURL,
	}
	db.Create(&profile)

	totpRecord := models.OTPVerification{
		UserID:    user.UserID,
		OTPCode:   "TOTP",
		Purpose:   pending.Secret,
		ExpiresAt: time.Now().AddDate(10, 0, 0),
	}
	db.Create(&totpRecord)

	pendingMu.Lock()
	delete(pendingRegistrations, strings.ToLower(pending.Username))
	delete(pendingRegistrations, strings.ToLower(pending.Email))
	delete(pendingByToken, pending.EmailToken)
	pendingMu.Unlock()

	return c.Status(201).JSON(fiber.Map{
		"message": "Registration successful",
		"user_id": user.UserID,
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

	moduleIDs, modules := getRoleModules(user.RoleID)

	return c.JSON(fiber.Map{
		"user":       user,
		"role_name":  getRoleName(user.RoleID),
		"module_ids": moduleIDs,
		"modules":    modules,
	})
}
