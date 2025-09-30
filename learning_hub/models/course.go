package models

import (
	"time"

	"gorm.io/gorm"
)

type Course struct {
	gorm.Model
	Title       string  `gorm:"type:varchar(200)" json:"title" binding:"required"`
	Description string  `gorm:"type:text" json:"description" binding:"required"`
	Price       float64 `gorm:"type:decimal(10,2)" json:"price"`
	Category    string  `gorm:"type:varchar(100)" json:"category"`
	Level       string  `gorm:"type:varchar(50)" json:"level" binding:"required,oneof=beginner intermediate advanced"`
	ImageURL    string  `gorm:"type:varchar(255)" json:"image_url"`
	Published   bool    `gorm:"default:false" json:"published"`

	// Relationships
	InstructorID uint         `json:"instructor_id"`                                       // Set from JWT, no binding required
	Instructor   User         `gorm:"foreignKey:InstructorID" json:"instructor,omitempty"` // Preload only
	Modules      []Module     `gorm:"foreignKey:CourseID" json:"modules,omitempty"`
	Enrollments  []Enrollment `gorm:"foreignKey:CourseID" json:"enrollments,omitempty"`
}

type Module struct {
	gorm.Model
	Title       string `gorm:"type:varchar(200)" json:"title" binding:"required"`
	Description string `gorm:"type:text" json:"description"`
	OrderIndex  int    `gorm:"default:0" json:"order_index"`

	// Relationships
	CourseID uint     `json:"course_id"` // Set from parent, no binding required
	Course   Course   `gorm:"foreignKey:CourseID" json:"course,omitempty"`
	Lessons  []Lesson `gorm:"foreignKey:ModuleID" json:"lessons,omitempty"`
}

type Lesson struct {
	gorm.Model
	Title      string `gorm:"type:varchar(200)" json:"title" binding:"required"`
	Content    string `gorm:"type:text" json:"content"`
	VideoURL   string `gorm:"type:varchar(255)" json:"video_url"`
	Duration   int    `gorm:"default:0" json:"duration"` // in minutes
	OrderIndex int    `gorm:"default:0" json:"order_index"`

	// Relationships
	ModuleID uint   `json:"module_id"` // Set from parent, no binding required
	Module   Module `gorm:"foreignKey:ModuleID" json:"module,omitempty"`
}

type Enrollment struct {
	gorm.Model
	UserID     uint      `json:"user_id"`
	CourseID   uint      `json:"course_id"`
	EnrolledAt time.Time `json:"enrolled_at"`
	Progress   float64   `gorm:"default:0" json:"progress"` // 0-100%

	User   User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Course Course `gorm:"foreignKey:CourseID" json:"course,omitempty"`
}

// UpdateCourseInput is used for partial updates
type UpdateCourseInput struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Category    string  `json:"category"`
	Level       string  `json:"level"`
	ImageURL    string  `json:"image_url"`
	Published   bool    `json:"published"`
}
