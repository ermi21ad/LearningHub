package handlers

import (
	"encoding/json"
	"learning_hub/models"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AssessmentHandler struct {
	db *gorm.DB
}

func NewAssessmentHandler(db *gorm.DB) *AssessmentHandler {
	return &AssessmentHandler{db: db}
}

// CreateQuiz creates a new quiz
func (h *AssessmentHandler) CreateQuiz(c *gin.Context) {
	var input struct {
		Title        string `json:"title" binding:"required"`
		Description  string `json:"description"`
		Instructions string `json:"instructions"`
		CourseID     uint   `json:"course_id" binding:"required"`
		ModuleID     *uint  `json:"module_id"`
		LessonID     *uint  `json:"lesson_id"`
		TimeLimit    int    `json:"time_limit"`
		MaxAttempts  int    `json:"max_attempts"`
		PassingScore int    `json:"passing_score"`
		Questions    []struct {
			Question      string              `json:"question" binding:"required"`
			QuestionType  models.QuestionType `json:"question_type" binding:"required"`
			Options       []string            `json:"options"`
			CorrectAnswer string              `json:"correct_answer" binding:"required"`
			Points        int                 `json:"points"`
			Explanation   string              `json:"explanation"`
			OrderIndex    int                 `json:"order_index"`
		} `json:"questions"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify course exists and user is instructor
	var course models.Course
	if err := h.db.First(&course, input.CourseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists || course.InstructorID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to create quiz for this course"})
		return
	}

	// Start transaction
	tx := h.db.Begin()

	quiz := models.Quiz{
		Title:        input.Title,
		Description:  input.Description,
		Instructions: input.Instructions,
		CourseID:     input.CourseID,
		ModuleID:     input.ModuleID,
		LessonID:     input.LessonID,
		TimeLimit:    input.TimeLimit,
		MaxAttempts:  input.MaxAttempts,
		PassingScore: input.PassingScore,
	}

	if err := tx.Create(&quiz).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create quiz"})
		return
	}

	// Create questions
	for _, qInput := range input.Questions {
		question := models.QuizQuestion{
			QuizID:        quiz.ID,
			Question:      qInput.Question,
			QuestionType:  qInput.QuestionType,
			CorrectAnswer: qInput.CorrectAnswer,
			Points:        qInput.Points,
			Explanation:   qInput.Explanation,
			OrderIndex:    qInput.OrderIndex,
		}

		// Convert options to JSON if provided
		if len(qInput.Options) > 0 {
			optionsJSON, _ := json.Marshal(qInput.Options)
			question.Options = models.JSON(optionsJSON)
		}

		if err := tx.Create(&question).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create question"})
			return
		}
	}

	tx.Commit()

	// Reload quiz with questions
	h.db.Preload("Questions").First(&quiz, quiz.ID)
	c.JSON(http.StatusCreated, quiz)
}

// StartQuizAttempt starts a new quiz attempt for a student
func (h *AssessmentHandler) StartQuizAttempt(c *gin.Context) {
	quizID := c.Param("quizId")
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	var quiz models.Quiz
	if err := h.db.Preload("Questions", func(db *gorm.DB) *gorm.DB {
		return db.Order("order_index ASC")
	}).First(&quiz, quizID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quiz not found"})
		return
	}

	// Check if quiz is published
	if !quiz.IsPublished {
		c.JSON(http.StatusForbidden, gin.H{"error": "Quiz is not available"})
		return
	}

	// Check if user is enrolled in the course
	var enrollment models.Enrollment
	if err := h.db.Where("user_id = ? AND course_id = ?", userID, quiz.CourseID).First(&enrollment).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not enrolled in this course"})
		return
	}

	// Check attempt limit
	var attemptCount int64
	h.db.Model(&models.QuizAttempt{}).
		Where("user_id = ? AND quiz_id = ?", userID, quizID).
		Count(&attemptCount)

	if quiz.MaxAttempts > 0 && int(attemptCount) >= quiz.MaxAttempts {
		c.JSON(http.StatusForbidden, gin.H{"error": "Maximum attempts reached"})
		return
	}

	// Calculate total points
	var totalPoints float64
	for _, question := range quiz.Questions {
		totalPoints += float64(question.Points)
	}

	// Create new attempt
	attempt := models.QuizAttempt{
		UserID:      userID.(uint),
		QuizID:      quiz.ID,
		StartedAt:   time.Now(),
		TotalPoints: totalPoints,
	}

	if err := h.db.Create(&attempt).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start attempt"})
		return
	}

	// Return quiz without correct answers
	safeQuiz := h.sanitizeQuiz(quiz)

	c.JSON(http.StatusOK, gin.H{
		"attempt": attempt,
		"quiz":    safeQuiz,
	})
}

// SubmitQuizAnswer submits an answer for a question
func (h *AssessmentHandler) SubmitQuizAnswer(c *gin.Context) {
	attemptID := c.Param("attemptId")
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	var input struct {
		QuestionID uint   `json:"question_id" binding:"required"`
		Answer     string `json:"answer" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify attempt belongs to user and is not completed
	var attempt models.QuizAttempt
	if err := h.db.Preload("Quiz").Preload("Quiz.Questions").
		First(&attempt, "id = ? AND user_id = ?", attemptID, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Attempt not found"})
		return
	}

	if attempt.IsCompleted {
		c.JSON(http.StatusForbidden, gin.H{"error": "Attempt already completed"})
		return
	}

	// Find the question
	var question models.QuizQuestion
	for _, q := range attempt.Quiz.Questions {
		if q.ID == input.QuestionID {
			question = q
			break
		}
	}

	if question.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Question not found"})
		return
	}

	// Check if answer already exists
	var existingAnswer models.QuizAnswer
	err := h.db.Where("attempt_id = ? AND question_id = ?", attemptID, input.QuestionID).
		First(&existingAnswer).Error

	var answer models.QuizAnswer
	if err == gorm.ErrRecordNotFound {
		// Create new answer
		answer = models.QuizAnswer{
			AttemptID:  attempt.ID,
			QuestionID: input.QuestionID,
			Answer:     input.Answer,
			IsCorrect:  h.checkAnswer(question, input.Answer),
		}

		if answer.IsCorrect {
			answer.PointsEarned = float64(question.Points)
		}

		if err := h.db.Create(&answer).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit answer"})
			return
		}
	} else {
		// Update existing answer
		existingAnswer.Answer = input.Answer
		existingAnswer.IsCorrect = h.checkAnswer(question, input.Answer)
		if existingAnswer.IsCorrect {
			existingAnswer.PointsEarned = float64(question.Points)
		} else {
			existingAnswer.PointsEarned = 0
		}

		if err := h.db.Save(&existingAnswer).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update answer"})
			return
		}
		answer = existingAnswer
	}

	c.JSON(http.StatusOK, answer)
}

// CompleteQuizAttempt completes a quiz attempt and calculates score
func (h *AssessmentHandler) CompleteQuizAttempt(c *gin.Context) {
	attemptID := c.Param("attemptId")
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	var attempt models.QuizAttempt
	if err := h.db.Preload("Answers").Preload("Answers.Question").
		Preload("Quiz").First(&attempt, "id = ? AND user_id = ?", attemptID, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Attempt not found"})
		return
	}

	if attempt.IsCompleted {
		c.JSON(http.StatusOK, attempt) // Already completed
		return
	}

	// Calculate total earned points
	var earnedPoints float64
	for _, answer := range attempt.Answers {
		earnedPoints += answer.PointsEarned
	}

	// Calculate score percentage
	score := (earnedPoints / attempt.TotalPoints) * 100
	isPassed := score >= float64(attempt.Quiz.PassingScore)

	// Update attempt
	attempt.Score = score
	attempt.EarnedPoints = earnedPoints
	attempt.IsCompleted = true
	attempt.IsPassed = isPassed
	now := time.Now()
	attempt.CompletedAt = &now
	attempt.TimeSpent = int(now.Sub(attempt.StartedAt).Seconds())

	if err := h.db.Save(&attempt).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete attempt"})
		return
	}

	c.JSON(http.StatusOK, attempt)
}

// Helper functions
func (h *AssessmentHandler) sanitizeQuiz(quiz models.Quiz) models.Quiz {
	// Remove correct answers from questions
	for i := range quiz.Questions {
		quiz.Questions[i].CorrectAnswer = ""
	}
	return quiz
}

func (h *AssessmentHandler) checkAnswer(question models.QuizQuestion, answer string) bool {
	switch question.QuestionType {
	case models.QuestionTypeMultipleChoice, models.QuestionTypeTrueFalse:
		return normalizeString(question.CorrectAnswer) == normalizeString(answer)
	case models.QuestionTypeShortAnswer:
		// For short answer, we might want more flexible matching
		return strings.Contains(normalizeString(question.CorrectAnswer), normalizeString(answer)) ||
			strings.Contains(normalizeString(answer), normalizeString(question.CorrectAnswer))
	default:
		return normalizeString(question.CorrectAnswer) == normalizeString(answer)
	}
}

func normalizeString(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// CreateAssignment creates a new assignment
func (h *AssessmentHandler) CreateAssignment(c *gin.Context) {
	var input struct {
		Title        string    `json:"title" binding:"required"`
		Description  string    `json:"description"`
		Instructions string    `json:"instructions"`
		CourseID     uint      `json:"course_id" binding:"required"`
		ModuleID     *uint     `json:"module_id"`
		DueDate      time.Time `json:"due_date" binding:"required"`
		MaxPoints    int       `json:"max_points"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify course exists and user is instructor
	var course models.Course
	if err := h.db.First(&course, input.CourseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists || course.InstructorID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to create assignment for this course"})
		return
	}

	assignment := models.Assignment{
		Title:        input.Title,
		Description:  input.Description,
		Instructions: input.Instructions,
		CourseID:     input.CourseID,
		ModuleID:     input.ModuleID,
		DueDate:      input.DueDate,
		MaxPoints:    input.MaxPoints,
	}

	if err := h.db.Create(&assignment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create assignment"})
		return
	}

	c.JSON(http.StatusCreated, assignment)
}

// SubmitAssignment handles assignment submissions
func (h *AssessmentHandler) SubmitAssignment(c *gin.Context) {
	assignmentID := c.Param("assignmentId")
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	var assignment models.Assignment
	if err := h.db.First(&assignment, assignmentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found"})
		return
	}

	// Check if assignment is published
	if !assignment.IsPublished {
		c.JSON(http.StatusForbidden, gin.H{"error": "Assignment is not available"})
		return
	}

	// Check if user is enrolled in the course
	var enrollment models.Enrollment
	if err := h.db.Where("user_id = ? AND course_id = ?", userID, assignment.CourseID).First(&enrollment).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not enrolled in this course"})
		return
	}

	// Check if already submitted
	var existingSubmission models.AssignmentSubmission
	err := h.db.Where("assignment_id = ? AND user_id = ?", assignmentID, userID).
		First(&existingSubmission).Error

	if err == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Already submitted this assignment"})
		return
	}

	file, _ := c.FormFile("file")
	submissionText := c.PostForm("submission_text")

	if file == nil && submissionText == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either file or text submission is required"})
		return
	}

	var fileURL string
	if file != nil {
		// Upload submission file
		// Note: You'll need to integrate with your existing file upload system
		// For now, we'll just store the filename
		fileURL = "/uploads/assignments/" + file.Filename

		// Save the file (you can integrate with your existing upload handler)
		if err := c.SaveUploadedFile(file, "./uploads/assignments/"+file.Filename); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file"})
			return
		}
	}

	submission := models.AssignmentSubmission{
		AssignmentID:   assignment.ID,
		UserID:         userID.(uint),
		FileURL:        fileURL,
		SubmissionText: submissionText,
		SubmittedAt:    time.Now(),
	}

	if err := h.db.Create(&submission).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit assignment"})
		return
	}

	c.JSON(http.StatusCreated, submission)
}

// GradeAssignment allows instructors to grade submissions
func (h *AssessmentHandler) GradeAssignment(c *gin.Context) {
	submissionID := c.Param("submissionId")

	var input struct {
		Grade    float64 `json:"grade" binding:"required"`
		Feedback string  `json:"feedback"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var submission models.AssignmentSubmission
	if err := h.db.Preload("Assignment").First(&submission, submissionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Submission not found"})
		return
	}

	// Verify user is the course instructor
	var course models.Course
	if err := h.db.First(&course, submission.Assignment.CourseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists || course.InstructorID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to grade this assignment"})
		return
	}

	// Validate grade
	if input.Grade < 0 || input.Grade > float64(submission.Assignment.MaxPoints) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid grade"})
		return
	}

	submission.Grade = &input.Grade
	submission.Feedback = input.Feedback
	submission.IsGraded = true
	now := time.Now()
	submission.GradedAt = &now

	if err := h.db.Save(&submission).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to grade submission"})
		return
	}

	c.JSON(http.StatusOK, submission)
}

// GetQuizAttempts returns all attempts for a quiz (for instructors)
func (h *AssessmentHandler) GetQuizAttempts(c *gin.Context) {
	quizID := c.Param("quizId")
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	// Verify user is instructor of the course
	var quiz models.Quiz
	if err := h.db.Preload("Course").First(&quiz, quizID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quiz not found"})
		return
	}

	if quiz.Course.InstructorID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to view these attempts"})
		return
	}

	var attempts []models.QuizAttempt
	if err := h.db.Preload("User").Preload("Answers").
		Where("quiz_id = ?", quizID).
		Order("created_at DESC").
		Find(&attempts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch attempts"})
		return
	}

	c.JSON(http.StatusOK, attempts)
}

// GetAssignmentSubmissions returns all submissions for an assignment (for instructors)
func (h *AssessmentHandler) GetAssignmentSubmissions(c *gin.Context) {
	assignmentID := c.Param("assignmentId")
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	// Verify user is instructor of the course
	var assignment models.Assignment
	if err := h.db.Preload("Course").First(&assignment, assignmentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found"})
		return
	}

	if assignment.Course.InstructorID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to view these submissions"})
		return
	}

	var submissions []models.AssignmentSubmission
	if err := h.db.Preload("User").Where("assignment_id = ?", assignmentID).
		Order("submitted_at DESC").
		Find(&submissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch submissions"})
		return
	}

	c.JSON(http.StatusOK, submissions)
}

// GetStudentQuizAttempts returns a student's own quiz attempts
func (h *AssessmentHandler) GetStudentQuizAttempts(c *gin.Context) {
	quizID := c.Param("quizId")
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	var attempts []models.QuizAttempt
	if err := h.db.Preload("Quiz").Preload("Answers").
		Where("quiz_id = ? AND user_id = ?", quizID, userID).
		Order("created_at DESC").
		Find(&attempts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch attempts"})
		return
	}

	c.JSON(http.StatusOK, attempts)
}

// GetStudentAssignmentSubmissions returns a student's own assignment submissions
func (h *AssessmentHandler) GetStudentAssignmentSubmissions(c *gin.Context) {
	assignmentID := c.Param("assignmentId")
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	var submissions []models.AssignmentSubmission
	if err := h.db.Preload("Assignment").
		Where("assignment_id = ? AND user_id = ?", assignmentID, userID).
		Order("submitted_at DESC").
		Find(&submissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch submissions"})
		return
	}

	c.JSON(http.StatusOK, submissions)
}
