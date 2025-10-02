package handlers

import (
	"learning_hub/models"
	"learning_hub/pkg/email"
	"learning_hub/pkg/validation"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminHandler struct {
	DB *gorm.DB
}

func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{DB: db}
}

// AdminStats returns overall platform statistics
func (h *AdminHandler) AdminStats(c *gin.Context) {
	var stats struct {
		TotalUsers        int64   `json:"total_users"`
		TotalCourses      int64   `json:"total_courses"`
		TotalEnrollments  int64   `json:"total_enrollments"`
		TotalPayments     int64   `json:"total_payments"`
		TotalRevenue      float64 `json:"total_revenue"`
		ActiveStudents    int64   `json:"active_students"`
		ActiveInstructors int64   `json:"active_instructors"`
	}

	// Get total counts
	h.DB.Model(&models.User{}).Count(&stats.TotalUsers)
	h.DB.Model(&models.Course{}).Count(&stats.TotalCourses)
	h.DB.Model(&models.Enrollment{}).Count(&stats.TotalEnrollments)
	h.DB.Model(&models.Payment{}).Count(&stats.TotalPayments)

	// Calculate total revenue from successful payments
	h.DB.Model(&models.Payment{}).Where("status = ?", models.PaymentStatusSuccess).
		Select("COALESCE(SUM(amount), 0)").Scan(&stats.TotalRevenue)

	// Count active students (users with enrollments)
	h.DB.Model(&models.User{}).Where("role = ?", "student").
		Joins("JOIN enrollments ON enrollments.user_id = users.id").
		Distinct("users.id").Count(&stats.ActiveStudents)

	// Count active instructors (users with courses)
	h.DB.Model(&models.User{}).Where("role = ?", "instructor").
		Joins("JOIN courses ON courses.instructor_id = users.id").
		Distinct("users.id").Count(&stats.ActiveInstructors)

	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
	})
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if err := h.DB.Delete(&models.User{}, id).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to delete user"})
		return
	}
	c.JSON(200, gin.H{"message": "User deleted successfully"})
}

// GetRecentPayments returns recent payment transactions
func (h *AdminHandler) GetRecentPayments(c *gin.Context) {
	var payments []models.Payment

	if err := h.DB.Preload("User", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, first_name, last_name, email")
	}).Preload("Course", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, title")
	}).Order("created_at DESC").Limit(20).Find(&payments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch payments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payments": payments,
		"count":    len(payments),
	})
}

// GetRecentEnrollments returns recent course enrollments
func (h *AdminHandler) GetRecentEnrollments(c *gin.Context) {
	var enrollments []models.Enrollment

	if err := h.DB.Preload("User", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, first_name, last_name, email")
	}).Preload("Course", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, title, instructor_id")
	}).Preload("Course.Instructor", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, first_name, last_name")
	}).Order("enrolled_at DESC").Limit(20).Find(&enrollments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch enrollments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"enrollments": enrollments,
		"count":       len(enrollments),
	})
}

// GetCourseAnalytics returns analytics for a specific course
func (h *AdminHandler) GetCourseAnalytics(c *gin.Context) {
	courseID := c.Param("id")

	var analytics struct {
		TotalEnrollments int64   `json:"total_enrollments"`
		TotalRevenue     float64 `json:"total_revenue"`
		AverageRating    float64 `json:"average_rating"`
		TotalReviews     int64   `json:"total_reviews"`
		CompletionRate   float64 `json:"completion_rate"`
	}

	// Get enrollment count
	h.DB.Model(&models.Enrollment{}).Where("course_id = ?", courseID).Count(&analytics.TotalEnrollments)

	// Get total revenue from this course
	h.DB.Model(&models.Payment{}).Where("course_id = ? AND status = ?", courseID, models.PaymentStatusSuccess).
		Select("COALESCE(SUM(amount), 0)").Scan(&analytics.TotalRevenue)

	// Get average rating
	h.DB.Model(&models.Review{}).Where("course_id = ?", courseID).
		Select("COALESCE(AVG(rating), 0)").Scan(&analytics.AverageRating)

	// Get total reviews
	h.DB.Model(&models.Review{}).Where("course_id = ?", courseID).Count(&analytics.TotalReviews)

	// Calculate completion rate (simplified - users with progress > 90%)
	var completedEnrollments int64
	h.DB.Model(&models.Enrollment{}).Where("course_id = ? AND progress >= ?", courseID, 90).Count(&completedEnrollments)

	if analytics.TotalEnrollments > 0 {
		analytics.CompletionRate = float64(completedEnrollments) / float64(analytics.TotalEnrollments) * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"course_id": courseID,
		"analytics": analytics,
	})
}

// GetUserManagement returns user list for admin management
func (h *AdminHandler) GetUserManagement(c *gin.Context) {
	var users []models.User

	if err := h.DB.Select("id, first_name, last_name, email, phone, role, created_at").
		Order("created_at DESC").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"count": len(users),
	})
}

// UpdateUserRole allows admin to change user roles
func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	userID := c.Param("id")

	var request struct {
		Role string `json:"role" binding:"required,oneof=student instructor admin"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	user.Role = request.Role
	if err := h.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User role updated successfully",
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}

// GetEmailDomains returns allowed email domains
func (h *AdminHandler) GetEmailDomains(c *gin.Context) {
	domains := validation.GetAllowedDomains()

	c.JSON(http.StatusOK, gin.H{
		"allowed_domains": domains,
		"count":           len(domains),
	})
}

// AddEmailDomain allows admin to add new email domains
func (h *AdminHandler) AddEmailDomain(c *gin.Context) {
	var request struct {
		Domain string `json:"domain" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Basic domain validation
	if !strings.Contains(request.Domain, ".") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid domain format"})
		return
	}

	validation.AddCustomDomain(request.Domain)

	c.JSON(http.StatusOK, gin.H{
		"message":         "Domain added successfully",
		"domain":          request.Domain,
		"allowed_domains": validation.GetAllowedDomains(),
	})
}

// RemoveEmailDomain allows admin to remove email domains
func (h *AdminHandler) RemoveEmailDomain(c *gin.Context) {
	domain := c.Param("domain")

	validation.RemoveCustomDomain(domain)

	c.JSON(http.StatusOK, gin.H{
		"message":         "Domain removed successfully",
		"domain":          domain,
		"allowed_domains": validation.GetAllowedDomains(),
	})
}

// TestEmailConfiguration tests the email setup
func (h *AdminHandler) TestEmailConfiguration(c *gin.Context) {
	// This would be a simple test email
	testEmail := email.EmailData{
		To:      "ermiasaddisalem18@gmail.com", // Send test to yourself
		Subject: "📧 LearnHub Email Test",
		Body: `
			<!DOCTYPE html>
			<html>
			<head>
				<style>
					body { font-family: Arial, sans-serif; padding: 20px; }
					.success { color: #10b981; }
				</style>
			</head>
			<body>
				<h1 class="success">✅ LearnHub Email Test Successful!</h1>
				<p>If you're reading this, your email configuration is working correctly.</p>
				<p><strong>Server Time:</strong> ` + time.Now().Format("2006-01-02 15:04:05") + `</p>
			</body>
			</html>
		`,
		Name: "Admin",
	}

	if err := email.SendEmail(testEmail); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Email test failed: " + err.Error(),
			"configuration": gin.H{
				"smtp_host": h.getSMTPHost(), // You'll need to add this method
				"smtp_port": h.getSMTPPort(),
				"smtp_user": h.getSMTPUser(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Test email sent successfully! Please check your inbox.",
		"sent_to": "ermiasaddisalem18@gmail.com",
	})
}

// Helper methods to get SMTP config (add these to AdminHandler)
func (h *AdminHandler) getSMTPHost() string {
	// You'll need to access config from the handler
	// This might require passing config to AdminHandler
	return "Configured in environment"
}

// Add this method to fix the compile error
func (h *AdminHandler) getSMTPUser() string {
	// You'll need to access config from the handler
	// This might require passing config to AdminHandler
	return "Configured in environment"

}
func (h *AdminHandler) getSMTPPort() int {
	// You'll need to access config from the handler
	// This might require passing config to AdminHandler
	return 587
}
