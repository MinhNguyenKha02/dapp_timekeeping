package handlers

import (
	"dapp_timekeeping/models"
	"dapp_timekeeping/types"
	"dapp_timekeeping/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AddEmployeeRequest struct {
	Username      string  `json:"username" validate:"required"`
	WalletAddress string  `json:"wallet_address" validate:"required"`
	Department    string  `json:"department" validate:"required"`
	Salary        float64 `json:"salary" validate:"required,gt=0"`
	Role          string  `json:"role" validate:"required,oneof=employee hr_manager accountant"`
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

	// Start transaction
	tx := DB.Begin()

	// Create new employee
	employee := models.User{
		ID:            uuid.New(),
		Username:      req.Username,
		WalletAddress: req.WalletAddress,
		Department:    req.Department,
		Salary:        req.Salary,
		Role:          req.Role,
		Status:        "active",
		LeaveBalance:  20, // Default leave balance
	}

	if err := tx.Create(&employee).Error; err != nil {
		tx.Rollback()
		utils.Logger.Error("Failed to create employee", zap.Error(err))
		return c.Status(500).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrDatabaseError,
		})
	}

	// Add to blockchain
	ctx := c.Context()
	if err := BlockchainService.AddEmployeeSalary(ctx, employee.WalletAddress, uint64(employee.Salary)); err != nil {
		tx.Rollback()
		utils.Logger.Error("Failed to add employee salary to blockchain", zap.Error(err))
		return c.Status(500).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrBlockchainError,
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

	// Update blockchain if salary changed
	if salary, ok := updateData["salary"].(float64); ok {
		ctx := c.Context()
		if err := BlockchainService.UpdateEmployeeSalary(ctx, employee.WalletAddress, uint64(salary)); err != nil {
			tx.Rollback()
			utils.Logger.Error("Failed to update salary on blockchain", zap.Error(err))
			return c.Status(500).JSON(types.APIResponse{
				Success: false,
				Error:   types.ErrBlockchainError,
			})
		}
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
