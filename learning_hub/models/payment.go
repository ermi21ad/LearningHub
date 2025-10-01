package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// PaymentStatus represents the status of a payment
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusSuccess   PaymentStatus = "success"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusCancelled PaymentStatus = "cancelled"
)

// Payment represents a payment transaction
type Payment struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	UserID   uint   `gorm:"not null" json:"user_id"`
	User     User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
	CourseID uint   `gorm:"not null" json:"course_id"`
	Course   Course `gorm:"foreignKey:CourseID" json:"course,omitempty"`

	// Payment Details
	Amount     float64 `gorm:"not null" json:"amount"`
	Currency   string  `gorm:"size:10;not null;default:'ETB'" json:"currency"`
	ChapaTxRef string  `gorm:"size:100;not null;uniqueIndex" json:"chapa_tx_ref"`
	ChapaRefID string  `gorm:"size:100" json:"chapa_ref_id"` // Chapa's internal reference

	// Status
	Status        PaymentStatus `gorm:"size:20;not null;default:'pending'" json:"status"`
	PaymentMethod string        `gorm:"size:50" json:"payment_method"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName specifies the table name for Payment
func (Payment) TableName() string {
	return "payments"
}

// Enrollment represents a user's enrollment in a course
type Enrollment struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	UserID   uint   `gorm:"not null;uniqueIndex:idx_user_course" json:"user_id"`
	User     User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
	CourseID uint   `gorm:"not null;uniqueIndex:idx_user_course" json:"course_id"`
	Course   Course `gorm:"foreignKey:CourseID" json:"course,omitempty"`

	// Enrollment Details
	PaymentID *uint    `gorm:"index" json:"payment_id"`
	Payment   *Payment `gorm:"foreignKey:PaymentID" json:"payment,omitempty"`
	IsActive  bool     `gorm:"not null;default:true" json:"is_active"`

	// Progress tracking
	Progress      float64 `gorm:"not null;default:0" json:"progress"` // Percentage completed
	CurrentModule *uint   `gorm:"index" json:"current_module"`
	CurrentLesson *uint   `gorm:"index" json:"current_lesson"`

	// Timestamps
	EnrolledAt  time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"enrolled_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

// TableName specifies the table name for Enrollment
func (Enrollment) TableName() string {
	return "enrollments"
}

// BeforeCreate generates a unique transaction reference
func (p *Payment) BeforeCreate(tx *gorm.DB) error {
	if p.ChapaTxRef == "" {
		p.ChapaTxRef = GenerateTxRef()
	}
	return nil
}

// GenerateTxRef generates a unique transaction reference
func GenerateTxRef() string {
	return fmt.Sprintf("learnhub-%d-%s", time.Now().Unix(), GenerateRandomString(8))
}

// GenerateRandomString generates a random string
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
