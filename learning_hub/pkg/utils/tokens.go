package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// GenerateVerificationToken creates a secure random token for email verification
func GenerateVerificationToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %v", err)
	}
	return hex.EncodeToString(bytes), nil
}

// IsTokenExpired checks if a verification token has expired (24 hours)
func IsTokenExpired(sentAt *time.Time) bool {
	if sentAt == nil {
		return true
	}
	return time.Since(*sentAt) > 24*time.Hour
}

// GenerateRandomCode generates a numeric code for alternative verification
func GenerateRandomCode(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to simple random if crypto fails
		return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	}

	code := 0
	for _, b := range bytes {
		code = (code*10 + int(b)%10)
	}

	// Ensure code has exactly 'length' digits
	format := fmt.Sprintf("%%0%dd", length)
	return fmt.Sprintf(format, code%1000000)
}
