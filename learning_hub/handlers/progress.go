package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"learning_hub/models"
	"net/http"
	"time"
)

type ProgressHandler struct {
	DB *gorm.DB
}

func NewProgressHandler(db *gorm.DB) *ProgressHandler {
	return &ProgressHandler{DB: db}
}

// GetCourseProgress returns detailed progress for a specific course
func (h *ProgressHandler) GetCourseProgress(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	courseID := c.Param("id")

	var enrollment models.Enrollment
	if err := h.DB.Preload("Course").Preload("Course.Modules").Preload("Course.Modules.Lessons").
		Where("user_id = ? AND course_id = ?", userID, courseID).
		First(&enrollment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Enrollment not found"})
		return
	}

	// Calculate detailed progress
	progress := h.calculateDetailedProgress(enrollment.CourseID, userID.(uint))

	c.JSON(http.StatusOK, gin.H{
		"enrollment": enrollment,
		"progress":   progress,
	})
}

// UpdateLessonProgress marks a lesson as completed and updates overall progress
func (h *ProgressHandler) UpdateLessonProgress(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request struct {
		LessonID  uint `json:"lesson_id" binding:"required"`
		CourseID  uint `json:"course_id" binding:"required"`
		TimeSpent int  `json:"time_spent" binding:"required"` // in minutes
		Completed bool `json:"completed"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Start transaction
	tx := h.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update or create lesson progress
	var lessonProgress models.LessonProgress
	err := tx.Where("user_id = ? AND lesson_id = ?", userID, request.LessonID).First(&lessonProgress).Error

	if err == gorm.ErrRecordNotFound {
		// Create new lesson progress
		lessonProgress = models.LessonProgress{
			UserID:    userID.(uint),
			LessonID:  request.LessonID,
			CourseID:  request.CourseID,
			Completed: request.Completed,
			TimeSpent: request.TimeSpent,
		}
		if request.Completed {
			lessonProgress.CompletedAt = time.Now()
		}
		if err := tx.Create(&lessonProgress).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create lesson progress"})
			return
		}
	} else if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update lesson progress"})
		return
	} else {
		// Update existing progress
		if request.Completed && !lessonProgress.Completed {
			lessonProgress.CompletedAt = time.Now()
		}
		lessonProgress.Completed = request.Completed
		lessonProgress.TimeSpent += request.TimeSpent
		if err := tx.Save(&lessonProgress).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update lesson progress"})
			return
		}
	}

	// Update enrollment progress
	if err := h.updateEnrollmentProgress(tx, userID.(uint), request.CourseID); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update course progress"})
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save progress"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Progress updated successfully",
		"progress": gin.H{
			"lesson_completed": request.Completed,
			"time_spent":       request.TimeSpent,
		},
	})
}

// GetStudentDashboard returns comprehensive student learning analytics
func (h *ProgressHandler) GetStudentDashboard(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var dashboard struct {
		TotalEnrollments  int64                   `json:"total_enrollments"`
		CompletedCourses  int64                   `json:"completed_courses"`
		CoursesInProgress int64                   `json:"courses_in_progress"`
		TotalLearningTime int                     `json:"total_learning_time"` // in minutes
		AverageProgress   float64                 `json:"average_progress"`
		RecentActivity    []models.LessonProgress `json:"recent_activity"`
		CertificatesCount int64                   `json:"certificates_count"`
	}

	// Get enrollment stats
	h.DB.Model(&models.Enrollment{}).Where("user_id = ?", userID).Count(&dashboard.TotalEnrollments)
	h.DB.Model(&models.Enrollment{}).Where("user_id = ? AND progress >= ?", userID, 100).Count(&dashboard.CompletedCourses)
	h.DB.Model(&models.Enrollment{}).Where("user_id = ? AND progress > ? AND progress < ?", userID, 0, 100).Count(&dashboard.CoursesInProgress)

	// Get total learning time
	h.DB.Model(&models.LessonProgress{}).Where("user_id = ?", userID).
		Select("COALESCE(SUM(time_spent), 0)").Scan(&dashboard.TotalLearningTime)

	// Get average progress
	h.DB.Model(&models.Enrollment{}).Where("user_id = ?", userID).
		Select("COALESCE(AVG(progress), 0)").Scan(&dashboard.AverageProgress)

	// Get certificates count
	h.DB.Model(&models.Certificate{}).Where("user_id = ?", userID).Count(&dashboard.CertificatesCount)

	// Get recent activity
	h.DB.Preload("Lesson").Preload("Course").
		Where("user_id = ?", userID).
		Order("updated_at DESC").Limit(10).
		Find(&dashboard.RecentActivity)

	c.JSON(http.StatusOK, gin.H{
		"dashboard": dashboard,
		"user_id":   userID,
	})
}

// GenerateCertificate generates a completion certificate for a course
func (h *ProgressHandler) GenerateCertificate(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	courseID := c.Param("id")

	// Check if enrollment exists and course is completed
	var enrollment models.Enrollment
	if err := h.DB.Preload("Course").Preload("User").
		Where("user_id = ? AND course_id = ?", userID, courseID).
		First(&enrollment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Enrollment not found"})
		return
	}

	if enrollment.Progress < 100 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Course not completed. Progress: " + fmt.Sprintf("%.1f%%", enrollment.Progress),
		})
		return
	}

	if enrollment.CertificateID != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Certificate already issued"})
		return
	}

	// Generate certificate
	certificate, err := h.createCertificate(enrollment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate certificate: " + err.Error()})
		return
	}

	// Update enrollment with certificate info
	now := time.Now()
	enrollment.CertificateID = &certificate.ID
	enrollment.CertificateIssuedAt = &now
	if err := h.DB.Save(&enrollment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update enrollment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Certificate generated successfully",
		"certificate": certificate,
	})
}

// GetCertificate returns certificate details
func (h *ProgressHandler) GetCertificate(c *gin.Context) {
	certificateID := c.Param("id")

	var certificate models.Certificate
	if err := h.DB.Preload("Enrollment").Preload("Enrollment.User").
		Preload("Enrollment.Course").
		Where("id = ?", certificateID).
		First(&certificate).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Certificate not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"certificate": certificate,
	})
}

// VerifyCertificate allows public verification of certificates
func (h *ProgressHandler) VerifyCertificate(c *gin.Context) {
	verificationCode := c.Query("code")
	certificateID := c.Query("id")

	var certificate models.Certificate
	query := h.DB.Preload("Enrollment").Preload("Enrollment.User").
		Preload("Enrollment.Course")

	if verificationCode != "" {
		query = query.Where("verification_code = ?", verificationCode)
	} else if certificateID != "" {
		query = query.Where("id = ?", certificateID)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Verification code or certificate ID required"})
		return
	}

	if err := query.First(&certificate).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"valid": false,
			"error": "Certificate not found or invalid",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid": true,
		"certificate": gin.H{
			"id":                certificate.ID,
			"student_name":      certificate.Enrollment.User.FirstName + " " + certificate.Enrollment.User.LastName,
			"course_title":      certificate.Enrollment.Course.Title,
			"issue_date":        certificate.IssueDate.Format("January 2, 2006"),
			"verification_code": certificate.VerificationCode,
		},
	})
}

// Helper function to calculate detailed progress
func (h *ProgressHandler) calculateDetailedProgress(courseID, userID uint) gin.H {
	var totalLessons int64
	var completedLessons int64
	var totalTimeSpent int

	// Count total lessons in course
	h.DB.Model(&models.Lesson{}).Joins("JOIN modules ON modules.id = lessons.module_id").
		Where("modules.course_id = ?", courseID).Count(&totalLessons)

	// Count completed lessons
	h.DB.Model(&models.LessonProgress{}).Joins("JOIN lessons ON lessons.id = lesson_progresses.lesson_id").
		Joins("JOIN modules ON modules.id = lessons.module_id").
		Where("lesson_progresses.user_id = ? AND modules.course_id = ? AND lesson_progresses.completed = ?",
			userID, courseID, true).Count(&completedLessons)

	// Calculate total time spent
	h.DB.Model(&models.LessonProgress{}).Joins("JOIN lessons ON lessons.id = lesson_progresses.lesson_id").
		Joins("JOIN modules ON modules.id = lessons.module_id").
		Where("lesson_progresses.user_id = ? AND modules.course_id = ?", userID, courseID).
		Select("COALESCE(SUM(lesson_progresses.time_spent), 0)").Scan(&totalTimeSpent)

	progress := 0.0
	if totalLessons > 0 {
		progress = float64(completedLessons) / float64(totalLessons) * 100
	}

	return gin.H{
		"progress_percentage": progress,
		"completed_lessons":   completedLessons,
		"total_lessons":       totalLessons,
		"time_spent_minutes":  totalTimeSpent,
		"time_spent_hours":    float64(totalTimeSpent) / 60,
		"remaining_lessons":   totalLessons - completedLessons,
	}
}

// Helper function to update enrollment progress
func (h *ProgressHandler) updateEnrollmentProgress(tx *gorm.DB, userID, courseID uint) error {
	var enrollment models.Enrollment
	if err := tx.Where("user_id = ? AND course_id = ?", userID, courseID).First(&enrollment).Error; err != nil {
		return err
	}

	// Calculate new progress
	progress := h.calculateDetailedProgress(courseID, userID)
	enrollment.Progress = progress["progress_percentage"].(float64)

	// Update completed lessons count
	enrollment.CompletedLessons = int(progress["completed_lessons"].(int64))
	enrollment.TotalLessons = int(progress["total_lessons"].(int64))
	enrollment.TimeSpent = progress["time_spent_minutes"].(int)
	enrollment.LastActivityAt = time.Now()

	// Check if course is completed
	if enrollment.Progress >= 100 && enrollment.CompletedAt == nil {
		now := time.Now()
		enrollment.CompletedAt = &now
	}

	return tx.Save(&enrollment).Error
}

// Helper function to create certificate
func (h *ProgressHandler) createCertificate(enrollment models.Enrollment) (*models.Certificate, error) {
	certificateID := fmt.Sprintf("LHC-%d-%s", enrollment.ID, time.Now().Format("20060102"))
	verificationCode := generateVerificationCode()

	certificate := models.Certificate{
		ID:               certificateID,
		EnrollmentID:     enrollment.ID,
		UserID:           enrollment.UserID,
		CourseID:         enrollment.CourseID,
		IssueDate:        time.Now(),
		VerificationCode: verificationCode,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Set expiry date (2 years from issue)
	expiryDate := time.Now().AddDate(2, 0, 0)
	certificate.ExpiryDate = &expiryDate

	if err := h.DB.Create(&certificate).Error; err != nil {
		return nil, err
	}

	return &certificate, nil
}

// Helper function to generate verification code
func generateVerificationCode() string {
	return fmt.Sprintf("LHC-%d", time.Now().UnixNano()%1000000)
}
