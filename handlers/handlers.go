package handlers

import (
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitHandlers(db *gorm.DB) {
	DB = db
}
