package handlers

import (
	"fmt"
	"learning_hub/models"
	"learning_hub/pkg/email"
	"learning_hub/pkg/jwt"
	"learning_hub/pkg/utils"
	"learning_hub/pkg/validation"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserHandler struct {
	DB *gorm.DB
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{DB: db}
}

// Update RegisterUser function to send verification email
func (h *UserHandler) RegisterUser(c *gin.Context) {
	var request struct {
		FirstName string `json:"first_name" binding:"required"`
		LastName  string `json:"last_name" binding:"required"`
		Email     string `json:"email" binding:"required,email"`
		Password  string `json:"password" binding:"required,min=6"`
		Phone     string `json:"phone" binding:"omitempty"`
		Role      string `json:"role" binding:"omitempty,oneof=student instructor admin"`
	}

	// Bind JSON input
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user data: " + err.Error(),
		})
		return
	}

	// Validate email domain
	if isValid, errMsg := validation.IsValidEmail(request.Email); !isValid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":           errMsg,
			"allowed_domains": validation.GetAllowedDomains(),
		})
		return
	}

	// Check if user already exists
	var existingUser models.User
	if err := h.DB.Where("email = ?", request.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "User with this email already exists",
		})
		return
	}

	// Generate verification token
	verificationToken, err := utils.GenerateVerificationToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate verification token",
		})
		return
	}

	// Set verification sent time
	verificationSentAt := time.Now()

	// Create new user (unverified)
	newUser := models.User{
		FirstName:          request.FirstName,
		LastName:           request.LastName,
		Email:              request.Email,
		Password:           request.Password,
		Phone:              request.Phone,
		Role:               request.Role,
		EmailVerified:      false,
		VerificationToken:  verificationToken,
		VerificationSentAt: &verificationSentAt,
	}

	// Set default role if not provided
	if newUser.Role == "" {
		newUser.Role = "student"
	}

	// Hash password
	if err := newUser.HashPassword(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to secure password: " + err.Error(),
		})
		return
	}

	// Create user in database
	if err := h.DB.Create(&newUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to register user: " + err.Error(),
		})
		return
	}

	// Send verification email in background
	go func() {
		fullName := newUser.FirstName + " " + newUser.LastName
		if err := email.SendVerificationEmail(newUser.Email, fullName, verificationToken); err != nil {
			log.Printf("Failed to send verification email: %v", err)
		}
	}()

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully. Please check your email for verification link.",
		"user": gin.H{
			"id":             newUser.ID,
			"first_name":     newUser.FirstName,
			"last_name":      newUser.LastName,
			"email":          newUser.Email,
			"phone":          newUser.Phone,
			"role":           newUser.Role,
			"email_verified": newUser.EmailVerified,
		},
		"verification_required": true,
		"allowed_domains":       validation.GetAllowedDomains(),
	})
}

// VerifyEmail handles email verification
// VerifyEmail handles email verification
func (h *UserHandler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	fmt.Printf("🔍 VerifyEmail called with token: %s\n", token)

	if token == "" {
		fmt.Printf("❌ No token provided\n")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Verification token is required",
		})
		return
	}

	// Find user by verification token
	var user models.User
	if err := h.DB.Where("verification_token = ?", token).First(&user).Error; err != nil {
		fmt.Printf("❌ User not found for token: %s, error: %v\n", token, err)

		// Check if token might be expired or already used
		var existingUser models.User
		if err := h.DB.Where("email = ?", "jerbawjerbex@gmail.com").First(&existingUser).Error; err == nil {
			fmt.Printf("ℹ️ User exists but token doesn't match. User token: '%s'\n", existingUser.VerificationToken)
			fmt.Printf("ℹ️ User verification status: %t\n", existingUser.EmailVerified)
		}

		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid or expired verification token",
		})
		return
	}

	fmt.Printf("✅ User found: ID=%d, Email=%s, Verified=%t\n", user.ID, user.Email, user.EmailVerified)

	// Check if token is expired
	if utils.IsTokenExpired(user.VerificationSentAt) {
		fmt.Printf("❌ Token expired. Sent at: %v\n", user.VerificationSentAt)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Verification token has expired. Please request a new one.",
		})
		return
	}

	// Check if already verified
	if user.EmailVerified {
		fmt.Printf("ℹ️ Email already verified\n")
		c.JSON(http.StatusOK, gin.H{
			"message": "Email is already verified",
			"user": gin.H{
				"id":    user.ID,
				"email": user.Email,
			},
		})
		return
	}

	fmt.Printf("📝 Updating user verification status...\n")

	// Update user as verified using direct SQL to be safe
	result := h.DB.Exec(`
		UPDATE users 
		SET email_verified = true, verification_token = NULL 
		WHERE id = ? AND verification_token = ?
	`, user.ID, token)

	if result.Error != nil {
		fmt.Printf("❌ Failed to update user verification: %v\n", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to verify email: " + result.Error.Error(),
		})
		return
	}

	if result.RowsAffected == 0 {
		fmt.Printf("❌ No rows updated - possible race condition\n")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to verify email - no changes made",
		})
		return
	}

	fmt.Printf("✅ Email verified successfully for user: %s\n", user.Email)

	// Send verification success email
	go func() {
		fullName := user.FirstName + " " + user.LastName
		fmt.Printf("📧 Sending verification success email to: %s\n", user.Email)
		if err := email.SendVerificationSuccessEmail(user.Email, fullName); err != nil {
			log.Printf("Failed to send verification success email: %v", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "Email verified successfully! Your account is now active.",
		"user": gin.H{
			"id":             user.ID,
			"first_name":     user.FirstName,
			"last_name":      user.LastName,
			"email":          user.Email,
			"email_verified": true,
		},
	})
}

// ResendVerificationEmail allows users to request a new verification email
func (h *UserHandler) ResendVerificationEmail(c *gin.Context) {
	var request struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	fmt.Printf("🔍 Resend verification request for email: %s\n", request.Email)

	// Find user by email
	var user models.User
	if err := h.DB.Where("email = ?", request.Email).First(&user).Error; err != nil {
		// Don't reveal if user exists or not for security
		c.JSON(http.StatusOK, gin.H{
			"message": "If the email exists, a verification link has been sent.",
		})
		return
	}

	fmt.Printf("✅ User found: ID=%d, Email=%s, Verified=%t, VerificationToken='%s'\n",
		user.ID, user.Email, user.EmailVerified, user.VerificationToken)

	// Check if already verified
	if user.EmailVerified {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Email is already verified",
		})
		return
	}

	// Generate new verification token
	newToken, err := utils.GenerateVerificationToken()
	if err != nil {
		fmt.Printf("❌ Failed to generate verification token: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate verification token",
		})
		return
	}

	fmt.Printf("✅ New verification token generated: %s\n", newToken)

	// Update user with new token using direct SQL to handle empty string case
	verificationSentAt := time.Now()

	// Use raw SQL update to avoid any ORM issues with empty strings
	result := h.DB.Exec(`
		UPDATE users 
		SET verification_token = ?, verification_sent_at = ? 
		WHERE id = ? AND email = ?
	`, newToken, verificationSentAt, user.ID, user.Email)

	if result.Error != nil {
		fmt.Printf("❌ Failed to update user with new token: %v\n", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to resend verification email: " + result.Error.Error(),
		})
		return
	}

	if result.RowsAffected == 0 {
		fmt.Printf("❌ No rows affected during update\n")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update user record",
		})
		return
	}

	fmt.Printf("✅ User updated with new verification token\n")

	// Send new verification email
	go func() {
		fullName := user.FirstName + " " + user.LastName
		fmt.Printf("📧 Sending verification email to: %s\n", user.Email)
		if err := email.SendVerificationEmail(user.Email, fullName, newToken); err != nil {
			log.Printf("❌ Failed to resend verification email: %v", err)
		} else {
			fmt.Printf("✅ Verification email sent successfully to: %s\n", user.Email)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "Verification email sent successfully. Please check your email.",
		"email":   user.Email,
	})
}

// LoginUser handles user login
func (h *UserHandler) LoginUser(c *gin.Context) {
	var loginData struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	// Bind JSON input
	if err := c.ShouldBindJSON(&loginData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid login data: " + err.Error(),
		})
		return
	}

	// Find user by email
	var user models.User
	if err := h.DB.Where("email = ?", loginData.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid email or password",
		})
		return
	}

	// Check if email is verified
	if !user.EmailVerified {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":               "Please verify your email address before logging in",
			"email_verified":      false,
			"resend_verification": true,
		})
		return
	}

	// Check password
	if err := user.CheckPassword(loginData.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid email or password",
		})
		return
	}

	// Generate JWT token
	token, err := jwt.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate token: " + err.Error(),
		})
		return
	}

	// Login successful
	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   token,
		"user": gin.H{
			"id":             user.ID,
			"first_name":     user.FirstName,
			"last_name":      user.LastName,
			"email":          user.Email,
			"phone":          user.Phone,
			"role":           user.Role,
			"email_verified": user.EmailVerified,
		},
	})
}

// GetProfile returns the authenticated user's profile
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         user.ID,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
		"email":      user.Email,
		"phone":      user.Phone, // Include phone in response
		"role":       user.Role,
		"created_at": user.CreatedAt,
	})
}

// UpdateProfile updates the authenticated user's profile
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	var updateData struct {
		FirstName string `json:"first_name" binding:"omitempty"`
		LastName  string `json:"last_name" binding:"omitempty"`
		Phone     string `json:"phone" binding:"omitempty"` // Added phone field
		Password  string `json:"password" binding:"omitempty,min=6"`
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid update data: " + err.Error()})
		return
	}

	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update fields if provided
	if updateData.FirstName != "" {
		user.FirstName = updateData.FirstName
	}
	if updateData.LastName != "" {
		user.LastName = updateData.LastName
	}
	if updateData.Phone != "" {
		user.Phone = updateData.Phone // Update phone field
	}
	if updateData.Password != "" {
		user.Password = updateData.Password
		if err := user.HashPassword(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to secure password: " + err.Error()})
			return
		}
	}

	if err := h.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"user": gin.H{
			"id":         user.ID,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"email":      user.Email,
			"phone":      user.Phone, // Include updated phone in response
			"role":       user.Role,
		},
	})
	// In UpdateProfile function, add this check:
	if updateData.Password != "" {
		// If user is changing password, require current password verification
		var currentPassword struct {
			CurrentPassword string `json:"current_password" binding:"required"`
		}

		if err := c.ShouldBindJSON(&currentPassword); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Current password is required to change password",
			})
			return
		}

		// Verify current password
		if err := user.CheckPassword(currentPassword.CurrentPassword); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Current password is incorrect",
			})
			return
		}

		user.Password = updateData.Password
		if err := user.HashPassword(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to secure password: " + err.Error()})
			return
		}
	}
}

// GetUserEnrollments returns all courses the user is enrolled in
func (h *UserHandler) GetUserEnrollments(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	var enrollments []models.Enrollment
	if err := h.DB.Preload("Course").Preload("Course.Instructor").
		Where("user_id = ? AND is_active = ?", userID, true).
		Find(&enrollments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch enrollments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"enrollments": enrollments,
		"count":       len(enrollments),
	})
}

// ForgotPassword handles password reset requests
func (h *UserHandler) ForgotPassword(c *gin.Context) {
	var request struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Valid email address is required",
		})
		return
	}

	// Find user by email
	var user models.User
	if err := h.DB.Where("email = ?", request.Email).First(&user).Error; err != nil {
		// Don't reveal if user exists for security
		c.JSON(http.StatusOK, gin.H{
			"message": "If the email exists, a password reset link has been sent.",
		})
		return
	}

	// Check if email is verified
	if !user.EmailVerified {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Please verify your email address before resetting password",
		})
		return
	}

	// Generate reset token
	resetToken, err := utils.GeneratePasswordResetToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate reset token",
		})
		return
	}

	// Set reset token and expiry
	resetSentAt := time.Now()
	resetExpiresAt := utils.CalculateResetExpiry()
	user.ResetToken = resetToken
	user.ResetSentAt = &resetSentAt
	user.ResetExpiresAt = &resetExpiresAt

	if err := h.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process reset request",
		})
		return
	}

	// Send password reset email
	go func() {
		fullName := user.FirstName + " " + user.LastName
		if err := email.SendPasswordResetEmail(user.Email, fullName, resetToken); err != nil {
			log.Printf("Failed to send password reset email: %v", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message":    "Password reset link sent to your email",
		"expires_in": "1 hour",
	})
}

// ResetPassword handles password reset with token
func (h *UserHandler) ResetPassword(c *gin.Context) {
	var request struct {
		Token    string `json:"token" binding:"required"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Token and new password are required",
		})
		return
	}

	// Find user by reset token
	var user models.User
	if err := h.DB.Where("reset_token = ?", request.Token).First(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid or expired reset token",
		})
		return
	}

	// Check if token is expired
	if utils.IsResetTokenExpired(user.ResetExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Reset token has expired. Please request a new one.",
		})
		return
	}

	// Update password
	user.Password = request.Password
	if err := user.HashPassword(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to secure new password",
		})
		return
	}

	// Clear reset token fields
	user.ResetToken = ""
	user.ResetSentAt = nil
	user.ResetExpiresAt = nil

	if err := h.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to reset password",
		})
		return
	}

	// Send success notification email
	go func() {
		fullName := user.FirstName + " " + user.LastName
		if err := email.SendPasswordResetSuccessEmail(user.Email, fullName); err != nil {
			log.Printf("Failed to send password reset success email: %v", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "Password reset successfully! You can now login with your new password.",
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
		},
	})
}

// ValidateResetToken checks if a reset token is valid
func (h *UserHandler) ValidateResetToken(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Reset token is required",
		})
		return
	}

	// Find user by reset token
	var user models.User
	if err := h.DB.Where("reset_token = ?", token).First(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid": false,
			"error": "Invalid reset token",
		})
		return
	}

	// Check if token is expired
	if utils.IsResetTokenExpired(user.ResetExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid": false,
			"error": "Reset token has expired",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid": true,
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.FirstName + " " + user.LastName,
		},
	})
}
