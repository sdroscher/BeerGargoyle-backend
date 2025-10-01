package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	UUID            uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()"`
	Username        string
	FirstName       string
	LastName        string
	Email           string
	UntappdUserName *string
}
