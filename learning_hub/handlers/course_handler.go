package handlers

import (
	"learning_hub/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CourseHandler struct {
	DB *gorm.DB
}

func NewCourseHandler(db *gorm.DB) *CourseHandler {
	return &CourseHandler{DB: db}
}

// CreateCourse - Only instructors can create courses
// CreateCourse - Only instructors can create courses
func (h *CourseHandler) CreateCourse(c *gin.Context) {
	// Use a dedicated input struct without Instructor validation
	var input struct {
		Title       string  `json:"title" binding:"required"`
		Description string  `json:"description" binding:"required"`
		Price       float64 `json:"price"`
		Category    string  `json:"category"`
		Level       string  `json:"level" binding:"required,oneof=beginner intermediate advanced"`
		ImageURL    string  `json:"image_url"`
		Published   bool    `json:"published"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid course data: " + err.Error(),
		})
		return
	}

	instructorID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized: User ID not found in context",
		})
		return
	}

	// Create the course model from input
	newCourse := models.Course{
		Title:        input.Title,
		Description:  input.Description,
		Price:        input.Price,
		Category:     input.Category,
		Level:        input.Level,
		ImageURL:     input.ImageURL,
		Published:    input.Published,
		InstructorID: instructorID.(uint),
	}

	if err := h.DB.Create(&newCourse).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create course: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Course created successfully",
		"course":  newCourse,
	})
}

// GetCourses - Get all published courses (public)
func (h *CourseHandler) GetCourses(c *gin.Context) {
	var courses []models.Course
	if err := h.DB.Where("published = ?", true).Preload("Instructor", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, name, email") // Only load necessary instructor fields
	}).Find(&courses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch courses: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"courses": courses,
	})
}

// GetCourseByID - Get single course with modules and lessons
func (h *CourseHandler) GetCourseByID(c *gin.Context) {
	var course models.Course
	courseID := c.Param("id")
	if err := h.DB.Preload("Modules.Lessons").Preload("Instructor").First(&course, courseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Course not found: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"course": course,
	})
}

// UpdateCourse - Only course instructor can update
func (h *CourseHandler) UpdateCourse(c *gin.Context) {
	var course models.Course
	courseID := c.Param("id")
	if err := h.DB.First(&course, courseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Course not found: " + err.Error(),
		})
		return
	}

	userID, exists := c.Get("userID")
	if !exists || course.InstructorID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Forbidden: You are not the instructor of this course",
		})
		return
	}

	// Use the dedicated update struct
	var updateData models.UpdateCourseInput
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid course data: " + err.Error(),
		})
		return
	}

	// Update only provided fields
	if updateData.Title != "" {
		course.Title = updateData.Title
	}
	if updateData.Description != "" {
		course.Description = updateData.Description
	}
	course.Price = updateData.Price // Can be 0, so no empty check
	if updateData.Category != "" {
		course.Category = updateData.Category
	}
	if updateData.Level != "" {
		course.Level = updateData.Level
	}
	if updateData.ImageURL != "" {
		course.ImageURL = updateData.ImageURL
	}
	course.Published = updateData.Published

	if err := h.DB.Save(&course).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update course: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Course updated successfully",
		"course":  course,
	})
}
