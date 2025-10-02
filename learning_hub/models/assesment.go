package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type QuestionType string

const (
	QuestionTypeMultipleChoice QuestionType = "multiple_choice"
	QuestionTypeTrueFalse      QuestionType = "true_false"
	QuestionTypeShortAnswer    QuestionType = "short_answer"
	QuestionTypeCoding         QuestionType = "coding"
)

type Quiz struct {
	gorm.Model
	Title        string         `gorm:"type:varchar(200)" json:"title" binding:"required"`
	Description  string         `gorm:"type:text" json:"description"`
	Instructions string         `gorm:"type:text" json:"instructions"`
	CourseID     uint           `gorm:"not null" json:"course_id"`
	Course       Course         `gorm:"foreignKey:CourseID" json:"course,omitempty"`
	ModuleID     *uint          `json:"module_id"`
	Module       *Module        `gorm:"foreignKey:ModuleID" json:"module,omitempty"`
	LessonID     *uint          `json:"lesson_id"`
	Lesson       *Lesson        `gorm:"foreignKey:LessonID" json:"lesson,omitempty"`
	TimeLimit    int            `json:"time_limit"` // in minutes
	MaxAttempts  int            `gorm:"default:1" json:"max_attempts"`
	PassingScore int            `gorm:"default:70" json:"passing_score"` // percentage
	IsPublished  bool           `gorm:"default:false" json:"is_published"`
	Questions    []QuizQuestion `gorm:"foreignKey:QuizID" json:"questions,omitempty"`
}

type QuizQuestion struct {
	gorm.Model
	QuizID        uint         `gorm:"not null" json:"quiz_id"`
	Question      string       `gorm:"type:text;not null" json:"question" binding:"required"`
	QuestionType  QuestionType `gorm:"type:varchar(50);not null" json:"question_type" binding:"required"`
	Options       JSON         `gorm:"type:json" json:"options"` // For multiple choice: ["Option A", "Option B"]
	CorrectAnswer string       `gorm:"type:text;not null" json:"correct_answer" binding:"required"`
	Points        int          `gorm:"default:1" json:"points"`
	Explanation   string       `gorm:"type:text" json:"explanation"`
	OrderIndex    int          `gorm:"default:0" json:"order_index"`
}

type QuizAttempt struct {
	gorm.Model
	UserID       uint         `gorm:"not null" json:"user_id"`
	QuizID       uint         `gorm:"not null" json:"quiz_id"`
	User         User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Quiz         Quiz         `gorm:"foreignKey:QuizID" json:"quiz,omitempty"`
	Score        float64      `json:"score"` // percentage
	TotalPoints  float64      `json:"total_points"`
	EarnedPoints float64      `json:"earned_points"`
	IsCompleted  bool         `gorm:"default:false" json:"is_completed"`
	IsPassed     bool         `gorm:"default:false" json:"is_passed"`
	StartedAt    time.Time    `json:"started_at"`
	CompletedAt  *time.Time   `json:"completed_at"`
	TimeSpent    int          `json:"time_spent"` // in seconds
	Answers      []QuizAnswer `gorm:"foreignKey:AttemptID" json:"answers,omitempty"`
}

type QuizAnswer struct {
	gorm.Model
	AttemptID    uint         `gorm:"not null" json:"attempt_id"`
	QuestionID   uint         `gorm:"not null" json:"question_id"`
	Question     QuizQuestion `gorm:"foreignKey:QuestionID" json:"question,omitempty"`
	Answer       string       `gorm:"type:text;not null" json:"answer"`
	IsCorrect    bool         `json:"is_correct"`
	PointsEarned float64      `json:"points_earned"`
}

type Assignment struct {
	gorm.Model
	Title        string                 `gorm:"type:varchar(200)" json:"title" binding:"required"`
	Description  string                 `gorm:"type:text" json:"description"`
	Instructions string                 `gorm:"type:text" json:"instructions"`
	CourseID     uint                   `gorm:"not null" json:"course_id"`
	Course       Course                 `gorm:"foreignKey:CourseID" json:"course,omitempty"`
	ModuleID     *uint                  `json:"module_id"`
	Module       *Module                `gorm:"foreignKey:ModuleID" json:"module,omitempty"`
	DueDate      time.Time              `json:"due_date"`
	MaxPoints    int                    `gorm:"default:100" json:"max_points"`
	IsPublished  bool                   `gorm:"default:false" json:"is_published"`
	Submissions  []AssignmentSubmission `gorm:"foreignKey:AssignmentID" json:"submissions,omitempty"`
}

type AssignmentSubmission struct {
	gorm.Model
	AssignmentID   uint       `gorm:"not null" json:"assignment_id"`
	Assignment     Assignment `gorm:"foreignKey:AssignmentID" json:"assignment,omitempty"`
	UserID         uint       `gorm:"not null" json:"user_id"`
	User           User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	FileURL        string     `gorm:"type:varchar(500)" json:"file_url"`
	SubmissionText string     `gorm:"type:text" json:"submission_text"`
	SubmittedAt    time.Time  `json:"submitted_at"`
	Grade          *float64   `json:"grade"`
	GradedAt       *time.Time `json:"graded_at"`
	Feedback       string     `gorm:"type:text" json:"feedback"`
	IsGraded       bool       `gorm:"default:false" json:"is_graded"`
}

// JSON type for storing flexible data
type JSON json.RawMessage

func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = JSON("null")
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan JSON: value is not []byte")
	}
	*j = JSON(bytes)
	return nil
}

func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return []byte(j), nil
}
