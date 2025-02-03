package handlers

import (
	"dapp_timekeeping/models"
	"dapp_timekeeping/services"
	"dapp_timekeeping/types"
	"dapp_timekeeping/utils"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	DB                *gorm.DB
	BlockchainService *services.BlockchainService
)

func InitHandlers(db *gorm.DB, blockchain *services.BlockchainService) {
	DB = db
	BlockchainService = blockchain
}

type AddEmployeeRequest struct {
	Username      string  `json:"username" validate:"required"`
	WalletAddress string  `json:"wallet_address" validate:"required"`
	Salary        float64 `json:"salary" validate:"required,gt=0"`
}

func AddEmployee(c *fiber.Ctx) error {
	var req AddEmployeeRequest

	if err := c.BodyParser(&req); err != nil {
		utils.Logger.Error("Failed to parse request")
		return c.Status(400).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrInvalidInput,
		})
	}

	tx := DB.Begin()

	user := models.User{
		Username:      req.Username,
		WalletAddress: req.WalletAddress,
		Salary:        req.Salary,
		Status:        "active",
	}

	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		utils.Logger.Error("Database error", zap.Error(err))
		return c.Status(500).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrDatabaseError,
		})
	}

	ctx := c.Context()
	if err := BlockchainService.AddEmployee(ctx, req.WalletAddress, uint64(req.Salary)); err != nil {
		tx.Rollback()
		utils.Logger.Error("Blockchain error", zap.Error(err))
		return c.Status(500).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrBlockchainError,
		})
	}

	tx.Commit()

	return c.JSON(types.APIResponse{
		Success: true,
		Message: "Employee added successfully",
		Data:    user,
	})
}

func GetSalaryInfo(c *fiber.Ctx) error {
	return c.JSON(types.APIResponse{
		Success: false,
		Error:   "Not implemented",
	})
}
