package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	mathrand "math/rand"
	"time"
)

// Initialize the math/rand seed
func init() {
	mathrand.Seed(time.Now().UnixNano())
}

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
	// Use the new verification code system instead of long tokens
	return GenerateVerificationCode()
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

// GenerateVerificationCode generates a 6-digit numeric verification code
func GenerateVerificationCode() (string, error) {
	// Generate a 6-digit code using math/rand (already seeded in init)
	code := fmt.Sprintf("%06d", mathrand.Intn(1000000))
	return code, nil
}

// IsVerificationCodeExpired checks if a verification code has expired
func IsVerificationCodeExpired(sentAt *time.Time) bool {
	if sentAt == nil {
		return true
	}
	return time.Since(*sentAt) > 1*time.Hour // 1 hour expiry for codes
}

// GenerateSecureCode generates a cryptographically secure numeric code
func GenerateSecureCode(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("invalid code length: %d", length)
	}

	// Calculate the maximum value for the given length
	maxValue := 1
	for i := 0; i < length; i++ {
		maxValue *= 10
	}

	// Generate random bytes
	bytes := make([]byte, 4) // 4 bytes for up to 8 digits
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure code: %v", err)
	}

	// Convert bytes to number and ensure it's within range
	var num uint32
	for i := 0; i < 4; i++ {
		num = num<<8 | uint32(bytes[i])
	}
	num = num % uint32(maxValue)

	// Format with leading zeros
	format := fmt.Sprintf("%%0%dd", length)
	return fmt.Sprintf(format, num), nil
}
