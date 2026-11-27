package utils

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"
)

func GenerateEmailToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func envClean(key string) string {
	return strings.Trim(strings.TrimSpace(os.Getenv(key)), "'\"")
}

func buildVerificationHTML(firstName, verifyLink string) string {
	if strings.TrimSpace(firstName) == "" {
		firstName = "Member"
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: Arial, sans-serif; background-color: #070d1d; color: #ffffff; padding: 30px;">
  <div style="max-width: 520px; margin: 0 auto; background-color: #0d152a; border: 1px solid #1e293b; border-radius: 14px; padding: 32px; text-align: center;">
    <h2 style="margin: 0 0 10px; color: #ffffff;">ALAB FOOTBALL CLUB</h2>
    <p style="color: #94a3b8; font-size: 14px; margin-bottom: 24px;">Hello %s, please verify your email address to unlock your Google Authenticator QR Code and finish your registration.</p>
    <a href="%s" style="display: inline-block; background-color: #2563eb; color: #ffffff; text-decoration: none; font-weight: bold; padding: 14px 28px; border-radius: 8px; font-size: 15px;">Verify Account</a>
    <p style="color: #64748b; font-size: 12px; margin-top: 24px;">If the button doesn't work, copy and open this link:<br/><span style="color: #60a5fa;">%s</span></p>
  </div>
</body>
</html>`, firstName, verifyLink, verifyLink)
}

func SendVerificationEmail(toEmail, firstName, verifyLink string) error {
	subject := "Verify Your ALAB FC Account"
	htmlBody := buildVerificationHTML(firstName, verifyLink)

	scriptURL := envClean("GOOGLE_SCRIPT_EMAIL_URL")
	if scriptURL != "" {
		return sendViaGoogleScript(scriptURL, toEmail, subject, htmlBody)
	}

	brevoKey := envClean("BREVO_API_KEY")
	if brevoKey != "" {
		return sendViaBrevo(brevoKey, toEmail, subject, htmlBody)
	}

	resendKey := envClean("RESEND_API_KEY")
	if resendKey != "" {
		return sendViaResend(resendKey, toEmail, subject, htmlBody)
	}

	smtpEmail := envClean("SMTP_EMAIL")
	smtpPass := envClean("SMTP_PASSWORD")
	if smtpEmail != "" && smtpPass != "" {
		return sendViaGmailSMTP(smtpEmail, smtpPass, toEmail, subject, htmlBody)
	}

	return errors.New("email sender not configured: set GOOGLE_SCRIPT_EMAIL_URL in .env")
}

func sendViaGoogleScript(scriptURL, toEmail, subject, htmlBody string) error {
	payload := map[string]string{
		"to":      toEmail,
		"subject": subject,
		"html":    htmlBody,
	}

	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", scriptURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("google script returned non-JSON response (check Execute as=Me & Who has access=Anyone)")
	}
	if okVal, _ := result["ok"].(bool); !okVal {
		return fmt.Errorf("google script failed to send email")
	}
	return nil
}

func sendViaBrevo(apiKey, toEmail, subject, htmlBody string) error {
	senderEmail := envClean("SENDER_EMAIL")
	if senderEmail == "" {
		senderEmail = envClean("SMTP_EMAIL")
	}
	if senderEmail == "" {
		return errors.New("SENDER_EMAIL or SMTP_EMAIL is required for Brevo")
	}

	payload := map[string]interface{}{
		"sender": map[string]string{
			"name":  "ALAB Football Club",
			"email": senderEmail,
		},
		"to": []map[string]string{
			{"email": toEmail},
		},
		"subject":     subject,
		"htmlContent": htmlBody,
	}

	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", "https://api.brevo.com/v3/smtp/email", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("api-key", apiKey)
	req.Header.Set("content-type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("brevo email failed with status %d", resp.StatusCode)
	}
	return nil
}

func sendViaResend(apiKey, toEmail, subject, htmlBody string) error {
	senderEmail := envClean("SENDER_EMAIL")
	if senderEmail == "" {
		senderEmail = "ALAB FC <onboarding@resend.dev>"
	}

	payload := map[string]interface{}{
		"from":    senderEmail,
		"to":      []string{toEmail},
		"subject": subject,
		"html":    htmlBody,
	}

	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("resend email failed with status %d", resp.StatusCode)
	}
	return nil
}

func sendViaGmailSMTP(fromEmail, appPassword, toEmail, subject, htmlBody string) error {
	host := envClean("SMTP_HOST")
	if host == "" {
		host = "smtp.gmail.com"
	}
	port := envClean("SMTP_PORT")
	if port == "" {
		port = "587"
	}

	headers := fmt.Sprintf(
		"From: ALAB Football Club <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n",
		fromEmail, toEmail, subject,
	)
	msg := []byte(headers + htmlBody)

	auth := smtp.PlainAuth("", fromEmail, appPassword, host)
	return smtp.SendMail(host+":"+port, auth, fromEmail, []string{toEmail}, msg)
}
