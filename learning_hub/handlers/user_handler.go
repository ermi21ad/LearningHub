package handlers

import (
	"learning_hub/models"
	"learning_hub/pkg/jwt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserHandler struct {
	DB *gorm.DB
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{DB: db}
}

func (h *UserHandler) RegisterUser(c *gin.Context) {
	var newUser models.User

	if err := c.ShouldBindJSON(&newUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user data: " + err.Error(),
		})
		return
	}

	if err := newUser.HashPassword(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to secure password: " + err.Error(),
		})
		return
	}

	if err := h.DB.Create(&newUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to register user: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User has been registered successfully",
		"user": gin.H{
			"id":    newUser.ID,
			"email": newUser.Email,
			"name":  newUser.Name,
			"role":  newUser.Role,
		},
	})
}

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

	// Check password
	if err := user.CheckPassword(loginData.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid email or password",
		})
		return
	}
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
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.Role,
		},
	})
}
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
		"id":    user.ID,
		"email": user.Email,
		"name":  user.Name,
		"role":  user.Role,
	})
}
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	var updateData struct {
		Name     string `json:"name" binding:"omitempty"`
		Password string `json:"password" binding:"omitempty,min=6"`
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
	if updateData.Name != "" {
		user.Name = updateData.Name
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
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.Role,
		},
	})
}
