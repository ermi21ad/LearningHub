package models

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	FirstName string `gorm:"type:varchar(100);not null" json:"first_name"`
	LastName  string `gorm:"type:varchar(100);not null" json:"last_name"`
	Email     string `gorm:"type:varchar(100);uniqueIndex;not null" json:"email"`
	Password  string `gorm:"type:varchar(255);not null" json:"-"`
	Phone     string `gorm:"type:varchar(20)" json:"phone"` // Add this field if missing
	Role      string `gorm:"type:varchar(20);default:'student'" json:"role"`

	// Relationships
	Courses     []Course     `gorm:"foreignKey:InstructorID" json:"courses,omitempty"`
	Enrollments []Enrollment `gorm:"foreignKey:UserID" json:"enrollments,omitempty"`
	Reviews     []Review     `gorm:"foreignKey:UserID" json:"reviews,omitempty"`
}

// HashPassword hashes the user's password before saving
func (u *User) HashPassword() error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword compares a plain password with the stored hash
func (u *User) CheckPassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
}
