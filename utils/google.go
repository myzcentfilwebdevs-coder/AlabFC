package utils

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type GoogleUser struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified any    `json:"email_verified"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Aud           string `json:"aud"`
}

func GetGoogleLoginURL(state string) string {
	clientID := strings.Trim(strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")), "'\"")
	redirectURL := strings.Trim(strings.TrimSpace(os.Getenv("GOOGLE_REDIRECT_URL")), "'\"")
	if redirectURL == "" {
		redirectURL = "http://localhost:4000/api/auth/google/callback"
	}

	params := url.Values{}
	params.Add("client_id", clientID)
	params.Add("redirect_uri", redirectURL)
	params.Add("response_type", "code")
	params.Add("scope", "openid email profile")
	params.Add("access_type", "offline")
	params.Add("prompt", "select_account")
	if state != "" {
		params.Add("state", state)
	}

	return "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode()
}

func ExchangeGoogleCode(code string) (*GoogleUser, error) {
	clientID := strings.Trim(strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")), "'\"")
	clientSecret := strings.Trim(strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET")), "'\"")
	redirectURL := strings.Trim(strings.TrimSpace(os.Getenv("GOOGLE_REDIRECT_URL")), "'\"")
	if redirectURL == "" {
		redirectURL = "http://localhost:4000/api/auth/google/callback"
	}

	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", redirectURL)
	data.Set("grant_type", "authorization_code")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.PostForm("https://oauth2.googleapis.com/token", data)
	if err != nil {
		return nil, errors.New("failed to exchange code with google")
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, errors.New("failed to parse google token response")
	}

	if tokenResp.AccessToken != "" {
		return VerifyGoogleToken(tokenResp.AccessToken)
	}
	if tokenResp.IDToken != "" {
		return VerifyGoogleToken(tokenResp.IDToken)
	}

	return nil, errors.New("failed to get token from google: " + tokenResp.Error)
}

func VerifyGoogleToken(token string) (*GoogleUser, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errors.New("missing google token")
	}

	client := &http.Client{Timeout: 10 * time.Second}

	endpoint := "https://oauth2.googleapis.com/tokeninfo?id_token=" + url.QueryEscape(token)
	resp, err := client.Get(endpoint)
	if err != nil {
		return nil, errors.New("failed to connect to google auth")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v3/userinfo", nil)
		if err != nil {
			return nil, errors.New("invalid google token")
		}
		req.Header.Set("Authorization", "Bearer "+token)

		resp2, err := client.Do(req)
		if err != nil || resp2.StatusCode != http.StatusOK {
			return nil, errors.New("invalid or expired google token")
		}
		defer resp2.Body.Close()

		var gUser GoogleUser
		if err := json.NewDecoder(resp2.Body).Decode(&gUser); err != nil || gUser.Email == "" {
			return nil, errors.New("failed to read google user info")
		}
		return &gUser, nil
	}

	var gUser GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&gUser); err != nil || gUser.Email == "" {
		return nil, errors.New("failed to parse google token info")
	}

	clientID := strings.Trim(strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")), "'\"")
	if clientID != "" && gUser.Aud != "" && gUser.Aud != clientID {
		return nil, errors.New("google token client_id mismatch")
	}

	return &gUser, nil
}
