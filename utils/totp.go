package utils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

func GenerateTOTPSecret() (string, error) {
	bytes := make([]byte, 10)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes), nil
}

func BuildTOTPQRCode(accountName, secret string) (string, string) {
	issuer := "ALAB FC"
	label := url.PathEscape(issuer + ":" + accountName)
	otpauthURL := fmt.Sprintf(
		"otpauth://totp/%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		label,
		secret,
		url.QueryEscape(issuer),
	)
	qrImageURL := "https://api.qrserver.com/v1/create-qr-code/?size=250x250&data=" + url.QueryEscape(otpauthURL)
	return otpauthURL, qrImageURL
}

func VerifyTOTPCode(secret, code string) bool {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return false
	}

	secret = strings.ToUpper(strings.TrimSpace(secret))
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return false
	}

	nowCounter := time.Now().UTC().Unix() / 30
	for offset := int64(-1); offset <= 1; offset++ {
		if generateTOTPAt(key, nowCounter+offset) == code {
			return true
		}
	}
	return false
}

func generateTOTPAt(key []byte, counter int64) string {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(counter))

	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	sum := mac.Sum(nil)

	offset := sum[len(sum)-1] & 0x0f
	binCode := (int(sum[offset])&0x7f)<<24 |
		(int(sum[offset+1])&0xff)<<16 |
		(int(sum[offset+2])&0xff)<<8 |
		(int(sum[offset+3]) & 0xff)

	otp := binCode % 1000000
	return fmt.Sprintf("%06d", otp)
}
