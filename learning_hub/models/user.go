package models

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Email    string `gorm:"type:varchar(100);uniqueIndex" json:"email" binding:"required,email"`
	Password string `gorm:"type:varchar(100)" json:"password" binding:"required,min=6"`
	Name     string `gorm:"type:varchar(100)" json:"name" binding:"required"`
	Role     string `gorm:"type:varchar(20);default:'student'" json:"role" binding:"required,oneof=student instructor admin"`
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
