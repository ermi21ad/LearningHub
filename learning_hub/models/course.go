package models

import (
	"time"

	"gorm.io/gorm"
)

type Course struct {
	gorm.Model
	Title        string  `gorm:"type:varchar(200)" json:"title" binding:"required"`
	Description  string  `gorm:"type:text" json:"description" binding:"required"`
	Price        float64 `gorm:"type:decimal(10,2)" json:"price"`
	Category     string  `gorm:"type:varchar(100)" json:"category"`
	Level        string  `gorm:"type:varchar(50)" json:"level" binding:"required,oneof=beginner intermediate advanced"`
	ImageURL     string  `gorm:"type:varchar(500)" json:"image_url"`     // Updated to 500
	ThumbnailURL string  `gorm:"type:varchar(500)" json:"thumbnail_url"` // Added thumbnail field
	Published    bool    `gorm:"default:false" json:"published"`

	// Relationships
	InstructorID uint         `json:"instructor_id"`
	Instructor   User         `gorm:"foreignKey:InstructorID" json:"instructor,omitempty"`
	Modules      []Module     `gorm:"foreignKey:CourseID" json:"modules,omitempty"`
	Enrollments  []Enrollment `gorm:"foreignKey:CourseID" json:"enrollments,omitempty"`
}

type Module struct {
	gorm.Model
	Title       string `gorm:"type:varchar(200)" json:"title" binding:"required"`
	Description string `gorm:"type:text" json:"description"`
	OrderIndex  int    `gorm:"default:0" json:"order_index"`

	// Relationships
	CourseID uint     `json:"course_id"`
	Course   Course   `gorm:"foreignKey:CourseID" json:"course,omitempty"`
	Lessons  []Lesson `gorm:"foreignKey:ModuleID" json:"lessons,omitempty"`
}

type Lesson struct {
	gorm.Model
	Title       string `gorm:"type:varchar(200)" json:"title" binding:"required"`
	Content     string `gorm:"type:text" json:"content"`
	VideoURL    string `gorm:"type:varchar(500)" json:"video_url"`    // Single VideoURL field
	DocumentURL string `gorm:"type:varchar(500)" json:"document_url"` // Added document field
	Duration    int    `gorm:"default:0" json:"duration"`             // in minutes
	OrderIndex  int    `gorm:"default:0" json:"order_index"`

	// Relationships
	ModuleID uint   `json:"module_id"`
	Module   Module `gorm:"foreignKey:ModuleID" json:"module,omitempty"`
}

// UpdateCourseInput is used for partial updates
type UpdateCourseInput struct {
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	Price        float64 `json:"price"`
	Category     string  `json:"category"`
	Level        string  `json:"level"`
	ImageURL     string  `json:"image_url"`
	ThumbnailURL string  `json:"thumbnail_url"` // Added thumbnail field
	Published    bool    `json:"published"`
}

type LessonProgress struct {
	gorm.Model
	UserID      uint      `json:"user_id"`
	LessonID    uint      `json:"lesson_id"`
	CourseID    uint      `json:"course_id"` // Denormalized for easier queries
	Completed   bool      `gorm:"default:false" json:"completed"`
	CompletedAt time.Time `json:"completed_at"`
	TimeSpent   int       `gorm:"default:0" json:"time_spent"` // in minutes

	User   User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Lesson Lesson `gorm:"foreignKey:LessonID" json:"lesson,omitempty"`
	Course Course `gorm:"foreignKey:CourseID" json:"course,omitempty"`
}

type Review struct {
	gorm.Model
	UserID    uint      `json:"user_id"`
	CourseID  uint      `json:"course_id"`
	Rating    int       `gorm:"type:int;check:rating>=1 AND rating<=5" json:"rating" binding:"required,min=1,max=5"`
	Comment   string    `gorm:"type:text" json:"comment"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Course    Course    `gorm:"foreignKey:CourseID" json:"course,omitempty"`
}

func (lp *LessonProgress) CalculateProgressPercentage(totalDuration int) float64 {
	if totalDuration == 0 {
		return 0
	}
	// Convert time spent (minutes) to percentage based on lesson duration
	progress := (float64(lp.TimeSpent) / float64(totalDuration)) * 100
	if progress > 100 {
		return 100
	}
	return progress
}
