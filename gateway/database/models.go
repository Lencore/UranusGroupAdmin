package database

import (
	"time"
)

type User struct {
	ID        int64 `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	FirstName string
	LastName  string
	Username  string
}
