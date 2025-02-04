package handlers

import (
	"dapp_timekeeping/services"

	"gorm.io/gorm"
)

var (
	DB                *gorm.DB
	BlockchainService services.BlockchainServiceInterface
)

func InitHandlers(db *gorm.DB, blockchain services.BlockchainServiceInterface) {
	DB = db
	BlockchainService = blockchain
}
