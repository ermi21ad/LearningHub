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
	// Add prefix to make verification tokens unique from reset tokens
	return "verify_" + hex.EncodeToString(bytes), nil
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

// GeneratePasswordResetToken creates a secure random token for password reset
func GeneratePasswordResetToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate reset token: %v", err)
	}
	// Add prefix to make reset tokens unique from verification tokens
	return "reset_" + hex.EncodeToString(bytes), nil
}

// IsResetTokenExpired checks if a password reset token has expired (1 hour)
func IsResetTokenExpired(expiresAt *time.Time) bool {
	if expiresAt == nil {
		return true
	}
	return time.Now().After(*expiresAt)
}

// CalculateResetExpiry calculates when a reset token should expire
func CalculateResetExpiry() time.Time {
	return time.Now().Add(1 * time.Hour) // 1 hour expiry
}
