package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"learning_hub/models"
)

type LessonHandler struct {
	db *gorm.DB
}

func NewLessonHandler(db *gorm.DB) *LessonHandler {
	return &LessonHandler{db: db}
}

// CreateLesson creates a new lesson within a module
func (h *LessonHandler) CreateLesson(c *gin.Context) {
	var input struct {
		Title       string `json:"title" binding:"required"`
		Content     string `json:"content"`
		VideoURL    string `json:"video_url"`
		DocumentURL string `json:"document_url"`
		Duration    int    `json:"duration"`
		OrderIndex  int    `json:"order_index"`
		ModuleID    uint   `json:"module_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify module exists
	var module models.Module
	if err := h.db.First(&module, input.ModuleID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Module not found"})
		return
	}

	lesson := models.Lesson{
		Title:       input.Title,
		Content:     input.Content,
		VideoURL:    input.VideoURL,
		DocumentURL: input.DocumentURL,
		Duration:    input.Duration,
		OrderIndex:  input.OrderIndex,
		ModuleID:    input.ModuleID,
	}

	if err := h.db.Create(&lesson).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create lesson"})
		return
	}

	c.JSON(http.StatusCreated, lesson)
}

// GetLesson returns a specific lesson with progress
func (h *LessonHandler) GetLesson(c *gin.Context) {
	lessonID := c.Param("id")
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	var lesson models.Lesson
	if err := h.db.Preload("Module").Preload("Module.Course").
		First(&lesson, lessonID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Lesson not found"})
		return
	}

	// Get user progress for this lesson
	var progress models.LessonProgress
	h.db.Where("user_id = ? AND lesson_id = ?", userID, lessonID).First(&progress)

	response := gin.H{
		"lesson":   lesson,
		"progress": progress,
	}

	c.JSON(http.StatusOK, response)
}

// UpdateLesson updates an existing lesson
func (h *LessonHandler) UpdateLesson(c *gin.Context) {
	lessonID := c.Param("id")

	var input struct {
		Title       string `json:"title"`
		Content     string `json:"content"`
		VideoURL    string `json:"video_url"`
		DocumentURL string `json:"document_url"`
		Duration    int    `json:"duration"`
		OrderIndex  int    `json:"order_index"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var lesson models.Lesson
	if err := h.db.First(&lesson, lessonID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Lesson not found"})
		return
	}

	// Update fields if provided
	if input.Title != "" {
		lesson.Title = input.Title
	}
	if input.Content != "" {
		lesson.Content = input.Content
	}
	if input.VideoURL != "" {
		lesson.VideoURL = input.VideoURL
	}
	if input.DocumentURL != "" {
		lesson.DocumentURL = input.DocumentURL
	}
	if input.Duration > 0 {
		lesson.Duration = input.Duration
	}
	if input.OrderIndex >= 0 {
		lesson.OrderIndex = input.OrderIndex
	}

	if err := h.db.Save(&lesson).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update lesson"})
		return
	}

	c.JSON(http.StatusOK, lesson)
}

// UpdateLessonProgress tracks user progress in a lesson (time tracking version)
func (h *LessonHandler) UpdateLessonProgress(c *gin.Context) {
	lessonID := c.Param("id")
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	var input struct {
		TimeSpent int `json:"time_spent" binding:"required"` // in seconds
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get lesson to access course ID
	var lesson models.Lesson
	if err := h.db.Preload("Module").First(&lesson, lessonID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Lesson not found"})
		return
	}

	// Check if user is enrolled in the course
	var enrollment models.Enrollment
	if err := h.db.Where("user_id = ? AND course_id = ?", userID, lesson.Module.CourseID).First(&enrollment).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not enrolled in this course"})
		return
	}

	var progress models.LessonProgress
	err := h.db.Where("user_id = ? AND lesson_id = ?", userID, lessonID).First(&progress).Error

	if err == gorm.ErrRecordNotFound {
		// Create new progress record
		progress = models.LessonProgress{
			UserID:    userID.(uint),
			LessonID:  parseUint(lessonID),
			CourseID:  lesson.Module.CourseID,
			TimeSpent: input.TimeSpent,
			Completed: false,
		}
		if err := h.db.Create(&progress).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create progress"})
			return
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	} else {
		// Update existing progress - accumulate time spent
		progress.TimeSpent += input.TimeSpent
		if err := h.db.Save(&progress).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update progress"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Lesson progress updated",
		"progress": progress,
	})
}

// GetModuleLessons returns all lessons for a module with user progress
func (h *LessonHandler) GetModuleLessons(c *gin.Context) {
	moduleID := c.Param("moduleId")
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	var lessons []models.Lesson
	if err := h.db.Where("module_id = ?", moduleID).
		Order("order_index ASC").
		Find(&lessons).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch lessons"})
		return
	}

	// Get user progress for all lessons in this module
	var progress []models.LessonProgress
	h.db.Where("user_id = ? AND lesson_id IN (SELECT id FROM lessons WHERE module_id = ?)",
		userID, moduleID).Find(&progress)

	progressMap := make(map[uint]models.LessonProgress)
	for _, p := range progress {
		progressMap[p.LessonID] = p
	}

	// Combine lessons with progress
	type LessonWithProgress struct {
		models.Lesson
		Progress models.LessonProgress `json:"progress"`
	}

	var result []LessonWithProgress
	for _, lesson := range lessons {
		result = append(result, LessonWithProgress{
			Lesson:   lesson,
			Progress: progressMap[lesson.ID],
		})
	}

	c.JSON(http.StatusOK, result)
}

// DeleteLesson deletes a lesson
func (h *LessonHandler) DeleteLesson(c *gin.Context) {
	lessonID := c.Param("id")

	var lesson models.Lesson
	if err := h.db.First(&lesson, lessonID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Lesson not found"})
		return
	}

	// Check if there are any progress records
	var progressCount int64
	h.db.Model(&models.LessonProgress{}).Where("lesson_id = ?", lessonID).Count(&progressCount)

	if progressCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete lesson with user progress records"})
		return
	}

	if err := h.db.Delete(&lesson).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete lesson"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Lesson deleted successfully"})
}

// GetLessonAnalytics returns analytics for a lesson (for instructors)
func (h *LessonHandler) GetLessonAnalytics(c *gin.Context) {
	lessonID := c.Param("id")

	// Verify the user is the instructor of this course
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	var lesson models.Lesson
	if err := h.db.Preload("Module.Course").First(&lesson, lessonID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Lesson not found"})
		return
	}

	// Check if user is the course instructor
	if lesson.Module.Course.InstructorID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: You are not the instructor of this course"})
		return
	}

	// Get analytics data
	var totalEnrollments int64
	h.db.Model(&models.Enrollment{}).Where("course_id = ?", lesson.Module.CourseID).Count(&totalEnrollments)

	var completedCount int64
	h.db.Model(&models.LessonProgress{}).Where("lesson_id = ? AND completed = ?", lessonID, true).Count(&completedCount)

	var averageTimeSpent float64
	h.db.Model(&models.LessonProgress{}).Where("lesson_id = ?").Select("AVG(time_spent)").Row().Scan(&averageTimeSpent)

	analytics := gin.H{
		"lesson_id":          lessonID,
		"total_enrollments":  totalEnrollments,
		"completed_count":    completedCount,
		"completion_rate":    0.0,
		"average_time_spent": averageTimeSpent,
	}

	if totalEnrollments > 0 {
		analytics["completion_rate"] = float64(completedCount) / float64(totalEnrollments) * 100
	}

	c.JSON(http.StatusOK, analytics)
}

// Helper function to parse string to uint
func parseUint(s string) uint {
	id, _ := strconv.ParseUint(s, 10, 32)
	return uint(id)
}
