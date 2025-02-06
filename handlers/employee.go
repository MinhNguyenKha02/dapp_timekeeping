package handlers

import (
	"dapp_timekeeping/models"
	"dapp_timekeeping/types"
	"dapp_timekeeping/utils"

	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AddEmployeeRequest struct {
	FullName      string    `json:"full_name" validate:"required"`
	Email         string    `json:"email" validate:"required,email"`
	PhoneNumber   string    `json:"phone_number" validate:"required"`
	Address       string    `json:"address" validate:"required"`
	DateOfBirth   time.Time `json:"date_of_birth" validate:"required"`
	Gender        string    `json:"gender" validate:"required,oneof=male female other"`
	TaxID         string    `json:"tax_id" validate:"required"`
	Position      string    `json:"position" validate:"required"`
	Location      string    `json:"location" validate:"required"`
	Department    string    `json:"department" validate:"required"`
	WalletAddress string    `json:"wallet_address" validate:"required"`
	Salary        float64   `json:"salary" validate:"required,gt=0"`
	Role          string    `json:"role" validate:"required,oneof=employee hr hr_manager accountant"`
}

func GetAllEmployees(c *fiber.Ctx) error {
	var employees []models.User
	if err := DB.Find(&employees).Error; err != nil {
		utils.Logger.Error("Failed to fetch employees", zap.Error(err))
		return c.Status(500).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrDatabaseError,
		})
	}

	return c.JSON(types.APIResponse{
		Success: true,
		Data:    employees,
	})
}

func AddEmployee(c *fiber.Ctx) error {
	var req AddEmployeeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrInvalidInput,
		})
	}

	// Create new employee
	employee := models.User{
		ID:            uuid.New().String(),
		FullName:      req.FullName,
		Email:         req.Email,
		PhoneNumber:   req.PhoneNumber,
		Address:       req.Address,
		DateOfBirth:   req.DateOfBirth,
		Gender:        req.Gender,
		TaxID:         req.TaxID,
		Position:      req.Position,
		Location:      req.Location,
		Department:    req.Department,
		WalletAddress: req.WalletAddress,
		Salary:        req.Salary,
		Role:          req.Role,
		Status:        "active",
		LeaveBalance:  20, // Default leave balance
		OnboardDate:   time.Now(),
	}

	// Start transaction
	tx := DB.Begin()

	if err := tx.Create(&employee).Error; err != nil {
		tx.Rollback()
		utils.Logger.Error("Failed to create employee", zap.Error(err))
		return c.Status(500).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrDatabaseError,
		})
	}

	tx.Commit()

	return c.JSON(types.APIResponse{
		Success: true,
		Message: "Employee added successfully",
		Data:    employee,
	})
}

func GetSalaryInfo(c *fiber.Ctx) error {
	return c.JSON(types.APIResponse{
		Success: false,
		Error:   "Not implemented",
	})
}

func UpdateEmployee(c *fiber.Ctx) error {
	id := c.Params("id")
	userID, err := uuid.Parse(id)
	if err != nil {
		return c.Status(400).JSON(types.APIResponse{
			Success: false,
			Error:   "Invalid user ID",
		})
	}

	var updateData map[string]interface{}
	if err := c.BodyParser(&updateData); err != nil {
		return c.Status(400).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrInvalidInput,
		})
	}

	// Start transaction
	tx := DB.Begin()

	// Find employee first
	var employee models.User
	if err := tx.First(&employee, "id = ?", userID).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(types.APIResponse{
				Success: false,
				Error:   "Employee not found",
			})
		}
		return c.Status(500).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrDatabaseError,
		})
	}

	// Update employee
	if err := tx.Model(&employee).Updates(updateData).Error; err != nil {
		tx.Rollback()
		utils.Logger.Error("Failed to update employee", zap.Error(err))
		return c.Status(500).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrDatabaseError,
		})
	}

	tx.Commit()

	return c.JSON(types.APIResponse{
		Success: true,
		Message: "Employee updated successfully",
		Data:    employee,
	})
}

func DeleteEmployee(c *fiber.Ctx) error {
	id := c.Params("id")
	userID, err := uuid.Parse(id)
	if err != nil {
		return c.Status(400).JSON(types.APIResponse{
			Success: false,
			Error:   "Invalid user ID",
		})
	}

	var employee models.User
	if err := DB.First(&employee, "id = ?", userID).Error; err != nil {
		return c.Status(404).JSON(types.APIResponse{
			Success: false,
			Error:   "Employee not found",
		})
	}

	// Soft delete by updating status
	if err := DB.Model(&employee).Update("status", "left_company").Error; err != nil {
		utils.Logger.Error("Failed to delete employee", zap.Error(err))
		return c.Status(500).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrDatabaseError,
		})
	}

	return c.JSON(types.APIResponse{
		Success: true,
		Message: "Employee deleted successfully",
	})
}
