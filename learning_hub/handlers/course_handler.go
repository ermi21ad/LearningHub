package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"learning_hub/models"
	"learning_hub/pkg/fileupload"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CourseHandler struct {
	DB *gorm.DB
}

type UploadHandler struct {
	DB *gorm.DB
}

func NewCourseHandler(db *gorm.DB) *CourseHandler {
	return &CourseHandler{DB: db}
}

func NewUploadHandler(db *gorm.DB) *UploadHandler {
	return &UploadHandler{DB: db}
}

// CreateCourse - Only instructors can create courses
func (h *CourseHandler) CreateCourse(c *gin.Context) {
	// Use a dedicated input struct without Instructor validation
	var input struct {
		Title        string  `json:"title" binding:"required"`
		Description  string  `json:"description" binding:"required"`
		Price        float64 `json:"price"`
		Category     string  `json:"category"`
		Level        string  `json:"level" binding:"required,oneof=beginner intermediate advanced"`
		ImageURL     string  `json:"image_url"`
		ThumbnailURL string  `json:"thumbnail_url"`
		Published    bool    `json:"published"`
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
		ThumbnailURL: input.ThumbnailURL,
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
	if updateData.ThumbnailURL != "" {
		course.ThumbnailURL = updateData.ThumbnailURL
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
// GetStudentCourses - Get all courses a student is enrolled in
func (h *CourseHandler) GetStudentCourses(c *gin.Context) {
	userIDInterface, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	// Convert userID to uint properly
	userID, ok := userIDInterface.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
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

// GetCourseReviews handles GET /courses/:id/reviews
func (h *CourseHandler) GetCourseReviews(c *gin.Context) {
	courseID := c.Param("id")
	var reviews []models.Review
	if err := h.DB.Where("course_id = ?", courseID).Find(&reviews).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch reviews"})
		return
	}
	c.JSON(200, gin.H{"reviews": reviews})
}

// UploadFile handles file uploads for course materials
func (h *UploadHandler) UploadFile(c *gin.Context) {
	// Get the uploaded file from the form
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No file uploaded or invalid form data",
		})
		return
	}

	// Get the file type from the request (optional, can be auto-detected)
	fileType := c.DefaultPostForm("type", "")
	if fileType == "" {
		// Auto-detect file type from extension
		detectedType, err := fileupload.DetectFileType(file.Filename)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Unsupported file type. Please specify file type or upload a supported file.",
			})
			return
		}
		fileType = detectedType
	} else {
		// Validate that the specified file type is valid
		if !isValidFileType(fileType) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid file type specified",
			})
			return
		}
	}

	// Validate the file
	validationResult := fileupload.ValidateFile(file, fileType)
	if !validationResult.IsValid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": validationResult.Error.Error(),
		})
		return
	}

	// Generate secure filename
	secureFilename, err := generateSecureFilename(file.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate secure filename",
		})
		return
	}

	// Get upload path for the file type
	uploadSubdir := fileupload.GetUploadPath(fileType)
	uploadPath := filepath.Join("uploads", uploadSubdir)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create upload directory",
		})
		return
	}

	// Construct full file path
	fullPath := filepath.Join(uploadPath, secureFilename)

	// Save the file
	if err := c.SaveUploadedFile(file, fullPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save file",
		})
		return
	}

	// Set appropriate file permissions (readable by owner/group, not executable)
	if err := os.Chmod(fullPath, 0644); err != nil {
		// Log the error but don't fail the upload
		fmt.Printf("Warning: failed to set file permissions for %s: %v\n", fullPath, err)
	}

	// Generate file URL for client access
	// Note: In production, you might want to serve files through a dedicated endpoint
	// or use a CDN/base URL configuration
	fileURL := fmt.Sprintf("/uploads/%s/%s", uploadSubdir, secureFilename)

	c.JSON(http.StatusOK, gin.H{
		"message":       "File uploaded successfully",
		"file_url":      fileURL,
		"file_name":     secureFilename,
		"file_type":     fileType,
		"file_size":     file.Size,
		"original_name": file.Filename,
	})
}

// generateSecureFilename creates a secure filename to prevent path traversal attacks
func generateSecureFilename(originalFilename string) (string, error) {
	// Extract file extension
	ext := filepath.Ext(originalFilename)
	if ext == "" {
		return "", errors.New("file must have an extension")
	}

	// Generate random prefix (16 hex characters = 64 bits of randomness)
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	randomPrefix := hex.EncodeToString(randomBytes)

	// Get current timestamp for additional uniqueness
	timestamp := time.Now().Unix()

	// Clean the original filename (remove any path components, keep only basename)
	cleanName := filepath.Base(strings.TrimSuffix(originalFilename, ext))
	// Remove any remaining dangerous characters
	cleanName = sanitizeFilename(cleanName)

	// Combine into final secure filename
	secureFilename := fmt.Sprintf("%d_%s_%s%s", timestamp, randomPrefix, cleanName, ext)

	return secureFilename, nil
}

// sanitizeFilename removes or replaces potentially dangerous characters
func sanitizeFilename(filename string) string {
	// Replace spaces and special characters with underscores
	replacements := map[rune]rune{
		' ': '_', '!': '_', '@': '_', '#': '_', '$': '_', '%': '_', '^': '_', '&': '_', '*': '_',
		'(': '_', ')': '_', '+': '_', '=': '_', '{': '_', '}': '_', '[': '_', ']': '_', '|': '_',
		'\\': '_', ':': '_', ';': '_', '"': '_', '\'': '_', '<': '_', '>': '_', ',': '_', '?': '_',
		'/': '_', '~': '_', '`': '_',
	}

	var result strings.Builder
	for _, char := range filename {
		if replacement, exists := replacements[char]; exists {
			result.WriteRune(replacement)
		} else if (char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_' || char == '.' {
			result.WriteRune(char)
		} else {
			result.WriteRune('_')
		}
	}

	sanitized := result.String()

	// Ensure filename doesn't start or end with dots or underscores
	sanitized = strings.Trim(sanitized, "._")

	// If filename becomes empty after sanitization, use a default
	if sanitized == "" {
		sanitized = "file"
	}

	return sanitized
}

// isValidFileType checks if the provided file type is valid
func isValidFileType(fileType string) bool {
	switch fileType {
	case fileupload.FileTypeImage, fileupload.FileTypeVideo, fileupload.FileTypeDocument:
		return true
	default:
		return false
	}
}

// ServeFile serves uploaded files securely
func (h *UploadHandler) ServeFile(c *gin.Context) {
	// Get file type and filename from URL parameters
	fileType := c.Param("type")
	filename := c.Param("filename")

	// Validate file type
	if !isValidFileType(fileType) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid file type",
		})
		return
	}

	// Validate filename to prevent path traversal
	// Use filepath.Base to get only the filename part
	cleanFilename := filepath.Base(filename)
	if cleanFilename != filename {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid filename - path traversal detected",
		})
		return
	}

	// Additional filename validation
	if !isValidFilename(cleanFilename) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid filename characters",
		})
		return
	}

	// Construct the file path
	uploadSubdir := fileupload.GetUploadPath(fileType)
	filePath := filepath.Join("uploads", uploadSubdir, cleanFilename)

	// Clean the path to prevent directory traversal
	cleanPath := filepath.Clean(filePath)

	// Ensure the cleaned path is still within our uploads directory
	// This prevents attacks like "../../../etc/passwd"
	uploadsDir, err := filepath.Abs("uploads")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to resolve uploads directory",
		})
		return
	}

	cleanAbsPath, err := filepath.Abs(cleanPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to resolve file path",
		})
		return
	}

	// Verify that the file is within the uploads directory
	if !strings.HasPrefix(cleanAbsPath, uploadsDir) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied - file outside allowed directory",
		})
		return
	}

	// Check if file exists
	if _, err := os.Stat(cleanAbsPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "File not found",
		})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error accessing file",
		})
		return
	}

	// Determine and set Content-Type
	contentType := mime.TypeByExtension(filepath.Ext(cleanAbsPath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Content-Type", contentType)

	// Set cache control headers for performance
	// Cache for 1 hour for static assets
	c.Header("Cache-Control", "public, max-age=3600")
	c.Header("Expires", time.Now().Add(time.Hour).Format(http.TimeFormat))

	// Set Content-Disposition for certain file types (optional)
	// This prevents automatic execution of potentially dangerous files
	if shouldForceDownload(cleanAbsPath) {
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", cleanFilename))
	}

	// Serve the file
	c.File(cleanAbsPath)
}

// isValidFilename checks if the filename contains only safe characters
func isValidFilename(filename string) bool {
	if filename == "" || filename == "." || filename == ".." {
		return false
	}

	// Check for null bytes and other dangerous characters
	if strings.Contains(filename, "\x00") {
		return false
	}

	// Allow only alphanumeric, dots, hyphens, and underscores
	for _, char := range filename {
		if !(char >= 'a' && char <= 'z') &&
			!(char >= 'A' && char <= 'Z') &&
			!(char >= '0' && char <= '9') &&
			char != '.' && char != '-' && char != '_' {
			return false
		}
	}

	return true
}

// shouldForceDownload determines if a file should be downloaded rather than displayed inline
func shouldForceDownload(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))

	// Force download for potentially executable or dangerous file types
	dangerousExtensions := map[string]bool{
		".exe": true, ".bat": true, ".sh": true, ".js": true,
		".html": true, ".htm": true, ".php": true, ".pl": true,
		".py": true, ".rb": true, ".jar": true, ".msi": true,
	}

	return dangerousExtensions[ext]
}

// (Removed duplicate CreateModule method)
func (h *CourseHandler) DeleteCourse(c *gin.Context) {
	id := c.Param("id")
	if err := h.DB.Delete(&models.Course{}, "id = ?", id).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to delete course"})
		return
	}
	c.JSON(200, gin.H{"message": "Course deleted successfully"})
}

// GetInstructorCourses returns all courses for the authenticated instructor
func (h *CourseHandler) GetInstructorCourses(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}
	instructorID := user.(*models.User).ID
	var courses []models.Course
	if err := h.DB.Where("instructor_id = ?", instructorID).Find(&courses).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch courses"})
		return
	}
	c.JSON(200, gin.H{"courses": courses})
}

// CreateModule creates a new module for a course
func (h *CourseHandler) CreateModule(c *gin.Context) {
	courseID := c.Param("id")

	var input struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
		OrderIndex  int    `json:"order_index"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify course exists
	var course models.Course
	if err := h.DB.First(&course, courseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	// Check if user is the course instructor
	userID, exists := c.Get("userID")
	if !exists || course.InstructorID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to modify this course"})
		return
	}

	module := models.Module{
		Title:       input.Title,
		Description: input.Description,
		OrderIndex:  input.OrderIndex,
		CourseID:    course.ID,
	}

	if err := h.DB.Create(&module).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create module"})
		return
	}

	c.JSON(http.StatusCreated, module)
}
