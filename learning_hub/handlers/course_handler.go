package handlers

import (
	"learning_hub/models"
	"net/http"
	"time"

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

// EnrollCourse - Student enrolls in a course
func (h *CourseHandler) EnrollCourse(c *gin.Context) {
	courseID := c.Param("id")
	var course models.Course
	if err := h.DB.First(&course, courseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	// Check if already enrolled
	var existingEnrollment models.Enrollment
	if err := h.DB.Where("user_id = ? AND course_id = ?", userID, course.ID).First(&existingEnrollment).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Already enrolled in this course"})
		return
	}
	enrollment := models.Enrollment{
		UserID:     userID.(uint),
		CourseID:   course.ID,
		EnrolledAt: time.Now(),
		Progress:   0,
	}
	if err := h.DB.Create(&enrollment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enroll in course: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":    "Enrolled successfully",
		"enrollment": enrollment,
	})

}

// GetStudentCourses - Get all courses a student is enrolled in
func (h *CourseHandler) GetStudentCourses(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	var enrollments []models.Enrollment
	if err := h.DB.Preload("Course").Where("user_id = ?", userID).Find(&enrollments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch enrollments: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"enrollments": enrollments,
	})

}

// UpdateLessonProgress - Mark lesson as completed and update progress
func (h *CourseHandler) UpdateLessonProgress(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	var input struct {
		LessonID uint `json:"lesson_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	// Fetch lesson with Module preloaded to get CourseID
	var lesson models.Lesson
	if err := h.DB.Preload("Module").First(&lesson, input.LessonID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Lesson not found"})
		return
	}

	// Check if user is enrolled in the course
	var enrollment models.Enrollment
	if err := h.DB.Where("user_id = ? AND course_id = ?", userID, lesson.Module.CourseID).First(&enrollment).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not enrolled in this course"})
		return
	}

	// Mark lesson as completed
	var progress models.LessonProgress
	if err := h.DB.Where("user_id = ? AND lesson_id = ?", userID, input.LessonID).First(&progress).Error; err != nil {
		// Not found, create new progress
		progress = models.LessonProgress{
			UserID:      userID.(uint),
			LessonID:    input.LessonID,
			CourseID:    lesson.Module.CourseID,
			Completed:   true,
			CompletedAt: time.Now(), // Add completion timestamp
		}
		if err := h.DB.Create(&progress).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark lesson as completed: " + err.Error()})
			return
		}
	} else {
		// Already exists, update to completed
		progress.Completed = true
		progress.CompletedAt = time.Now() // Update timestamp
		if err := h.DB.Save(&progress).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update lesson progress: " + err.Error()})
			return
		}
	}

	// Update overall course progress
	var totalLessons int64
	h.DB.Model(&models.Lesson{}).Joins("JOIN modules ON lessons.module_id = modules.id").Where("modules.course_id = ?", lesson.Module.CourseID).Count(&totalLessons)

	var completedLessons int64
	h.DB.Model(&models.LessonProgress{}).Where("user_id = ? AND course_id = ? AND completed = ?", userID, lesson.Module.CourseID, true).Count(&completedLessons)

	if totalLessons > 0 {
		enrollment.Progress = (float64(completedLessons) / float64(totalLessons)) * 100
		h.DB.Save(&enrollment)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Lesson marked as completed",
		"lesson_progress": progress,
		"course_progress": enrollment.Progress,
		"completed":       completedLessons,
		"total":           totalLessons,
	})
}

// GetCourseProgress - Get student's progress in a specific course
func (h *CourseHandler) GetCourseProgress(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	courseID := c.Param("id")
	var enrollment models.Enrollment
	if err := h.DB.Where("user_id = ? AND course_id = ?", userID, courseID).First(&enrollment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Enrollment not found for this course"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"course_id": courseID,
		"progress":  enrollment.Progress,
	})

}

// GetStudentDashboard - Get student's learning dashboard
func (h *CourseHandler) GetStudentDashboard(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	var enrollments []models.Enrollment
	if err := h.DB.Preload("Course").Preload("Course.Instructor").
		Where("user_id = ?", userID).
		Find(&enrollments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch dashboard data"})
		return
	}

	// Calculate overall stats
	totalCourses := len(enrollments)
	completedCourses := 0
	totalProgress := 0.0

	for _, enrollment := range enrollments {
		totalProgress += enrollment.Progress
		if enrollment.Progress >= 100 {
			completedCourses++
		}
	}

	averageProgress := 0.0
	if totalCourses > 0 {
		averageProgress = totalProgress / float64(totalCourses)
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": gin.H{
			"total_courses":     totalCourses,
			"completed_courses": completedCourses,
			"average_progress":  averageProgress,
		},
		"enrollments": enrollments,
	})
}
func (h *CourseHandler) SubmitCourseReview(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	courseID := c.Param("id")
	var course models.Course
	if err := h.DB.First(&course, courseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}
	var input struct {
		Rating  int    `json:"rating" binding:"required,min=1,max=5"`
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}
	review := models.Review{
		UserID:   userID.(uint),
		CourseID: course.ID,
		Rating:   input.Rating,
		Comment:  input.Comment,
	}
	if err := h.DB.Create(&review).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit review: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Review submitted successfully",
		"review":  review,
	})
}
